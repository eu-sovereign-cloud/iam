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

	// internal/controller holds all business logic: one small controller
	// per operation, each depending only on the ports it actually needs
	// (ADR 0014). internal/service (REST) and internal/web (HTML) are both
	// just presentation bridges over the same controllers.
	createTenant := controller.NewCreateTenant(store, clock)
	listTenants := controller.NewListTenants(store)
	deleteTenant := controller.NewDeleteTenant(store)

	createUser := controller.NewCreateUser(store, clock)
	getUser := controller.NewGetUser(store)
	listUsers := controller.NewListUsers(store)
	setUserAdmin := controller.NewSetUserAdmin(store)
	deleteUser := controller.NewDeleteUser(store)

	createGrant := controller.NewCreateGrant(store, store, store, clock)
	listUserGrants := controller.NewListUserGrants(store)
	deleteGrant := controller.NewDeleteGrant(store)

	createPAT := controller.NewCreatePAT(store, store, signer, clock, cfg.JWTIssuer, cfg.JWTAudience)
	listUserPATs := controller.NewListUserPATs(store)
	getPAT := controller.NewGetPAT(store)
	revokePAT := controller.NewRevokePAT(store)
	authenticatePAT := controller.NewAuthenticatePAT(store, signer, clock)

	authenticateUser := controller.NewAuthenticateUser(authenticatePAT, store)
	ensureBootstrapAdmin := controller.NewEnsureBootstrapAdmin(listUsers, createUser, createPAT)

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
		CreateGrant: createGrant, ListUserGrants: listUserGrants, DeleteGrant: deleteGrant,
		CreatePAT: createPAT, ListUserPATs: listUserPATs, GetPAT: getPAT, RevokePAT: revokePAT,
	}
	webUI, err := web.New(
		authenticateUser,
		createTenant, listTenants, deleteTenant,
		createUser, getUser, listUsers, deleteUser,
		createGrant, listUserGrants, deleteGrant,
		createPAT, listUserPATs, getPAT, revokePAT,
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
