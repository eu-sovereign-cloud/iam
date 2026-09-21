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

	"github.com/eu-sovereign-cloud/iam/internal/adapter/kubecrypt"
	"github.com/eu-sovereign-cloud/iam/internal/adapter/kubestore"
	"github.com/eu-sovereign-cloud/iam/internal/adapter/system"
	"github.com/eu-sovereign-cloud/iam/internal/config"
	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/pkg/kube"
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

	clientset, err := kube.BuildClientset(cfg.KubeconfigPath)
	if err != nil {
		return err
	}

	store := kubestore.New(clientset, cfg.Namespace)
	slog.Info("loading state from Kubernetes", "namespace", cfg.Namespace)
	if err := store.Load(ctx); err != nil {
		return err
	}

	signer, err := kubecrypt.LoadOrCreate(ctx, clientset, cfg.Namespace)
	if err != nil {
		return err
	}
	clock := system.Clock{}

	// internal/controller holds all business logic: one small controller
	// per operation, each depending only on the ports it actually needs
	// (ADR 0014). internal/service (REST) and internal/web (HTML) are both
	// just presentation bridges over the same controllers.
	createTenant := &controller.CreateTenant{Tenants: store, Clock: clock}
	listTenants := &controller.ListTenants{Tenants: store}
	deleteTenant := &controller.DeleteTenant{Tenants: store}

	createUser := &controller.CreateUser{Users: store, Clock: clock}
	getUser := &controller.GetUser{Users: store}
	listUsers := &controller.ListUsers{Users: store}
	setUserAdmin := &controller.SetUserAdmin{Users: store}
	deleteUser := &controller.DeleteUser{Users: store}

	createGrant := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock}
	listUserGrants := &controller.ListUserGrants{Grants: store}
	deleteGrant := &controller.DeleteGrant{Grants: store}
	setGrantAdmin := &controller.SetGrantAdmin{Grants: store}

	createPAT := &controller.CreatePAT{PATs: store, Grants: store, Signer: signer, Clock: clock, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience}
	listUserPATs := &controller.ListUserPATs{PATs: store}
	revokePAT := &controller.RevokePAT{PATs: store}
	authenticatePAT := &controller.AuthenticatePAT{PATs: store, Signer: signer, Clock: clock}

	authenticateUser := &controller.AuthenticateUser{PATs: authenticatePAT, Users: store}
	ensureBootstrapAdmin := &controller.EnsureBootstrapAdmin{ListUsers: listUsers, CreateUser: createUser, CreatePAT: createPAT}

	if rawPAT, created, err := ensureBootstrapAdmin.Do(ctx); err != nil {
		return err
	} else if created {
		slog.Warn("created bootstrap admin PAT - copy it now, it will not be shown again",
			"subject", controller.BootstrapAdminSubject, "pat", rawPAT)
	}

	svc := &service.Service{
		AuthenticateUser: authenticateUser,
		CreateTenant:     createTenant, ListTenants: listTenants, DeleteTenant: deleteTenant,
		CreateUser: createUser, ListUsers: listUsers, SetUserAdmin: setUserAdmin, DeleteUser: deleteUser,
		CreateGrant: createGrant, ListUserGrants: listUserGrants, DeleteGrant: deleteGrant, SetGrantAdmin: setGrantAdmin,
		CreatePAT: createPAT, ListUserPATs: listUserPATs, RevokePAT: revokePAT,
	}
	webUI, err := web.New(
		authenticateUser,
		createTenant, listTenants, deleteTenant,
		createUser, getUser, listUsers, deleteUser,
		createGrant, listUserGrants, deleteGrant, setGrantAdmin,
		createPAT, listUserPATs, revokePAT,
	)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", svc.Router())
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
