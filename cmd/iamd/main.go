// Command iamd is the SECA IAM polyfill service: it mints gateway-compatible
// JWTs from Personal Access Tokens (see the top-level README and doc/adr).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eu-sovereign-cloud/iam/internal/adapter"
	"github.com/eu-sovereign-cloud/iam/internal/config"
	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/service"
	"github.com/eu-sovereign-cloud/iam/internal/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("iamd exiting", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogging(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	clientset, err := adapter.BuildClientset(cfg.KubeconfigPath)
	if err != nil {
		return err
	}

	store := adapter.NewStore(clientset, cfg.Namespace)
	slog.Info("loading state from Kubernetes", "namespace", cfg.Namespace)
	if err := store.Load(ctx); err != nil {
		return err
	}

	signer, err := adapter.LoadOrCreateSigner(ctx, clientset, cfg.Namespace)
	if err != nil {
		return err
	}

	clock := adapter.SystemClock{}
	userSvc := service.NewUserService(store, clock)
	tenantSvc := service.NewTenantService(store, clock)
	grantSvc := service.NewGrantService(store, store, store, clock)
	patSvc := service.NewPATService(store, store, signer, clock, cfg.JWTIssuer, cfg.JWTAudience)
	authSvc := service.NewAuthService(patSvc, store)

	if rawPAT, created, err := service.EnsureBootstrapAdmin(ctx, userSvc, patSvc); err != nil {
		return err
	} else if created {
		slog.Warn("created bootstrap admin PAT - copy it now, it will not be shown again",
			"subject", service.BootstrapAdminSubject, "pat", rawPAT)
	}

	ctl := &controller.Controller{
		Auth: authSvc, Users: userSvc, Tenants: tenantSvc, Grants: grantSvc, PATs: patSvc,
	}
	webUI, err := web.New(authSvc, userSvc, tenantSvc, grantSvc, patSvc)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", ctl.Router())
	mux.Handle("/web/", webUI.Router())

	srv := &http.Server{Addr: cfg.ListenAddr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func setupLogging(level string) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})))
}
