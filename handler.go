package httputils

import (
	"context"
	"net/http"
	"time"

	"github.com/aklinkert/go-exitcontext"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Handler is a custom wrapper around http.Server that cares about graceful termination.
type Handler struct {
	ctx                      context.Context
	gracefulShutdownDuration time.Duration
	logger                   logrus.FieldLogger
	okHandler                http.Handler
	router                   *mux.Router
	server                   *http.Server
	metricsListen            string
}

// NewHandler constructs the whole internal HTTP routing / handlers
func NewHandler(logger logrus.FieldLogger, listen, metricsListen string) *Handler {
	return NewHandlerWithContext(exitcontext.New(), logger, listen, metricsListen)
}

// NewHandlerWithContext constructs the whole internal HTTP routing / handlers and accepts a context
func NewHandlerWithContext(ctx context.Context, logger logrus.FieldLogger, listen, metricsListen string) *Handler {
	handler := &Handler{
		ctx:                      ctx,
		gracefulShutdownDuration: 2 * time.Second,
		logger:                   logger,
		okHandler:                NewOkHandler(logger),
		router:                   mux.NewRouter(),
		server: &http.Server{
			Addr: listen,
		},
		metricsListen: metricsListen,
	}

	return handler
}

// SetGracefulShutdownDuration overrides the default duration to wait for requests to be finished during graceful
// shutdown before it is enforced
func (h *Handler) SetGracefulShutdownDuration(duration time.Duration) {
	h.gracefulShutdownDuration = duration
}

// Handle registers a new route with a matcher for the URL path.
// See Route.Path() and Route.Handler(). Wrapper around mux.router.
func (h *Handler) Handle(path string, handler http.Handler) *mux.Route {
	return h.router.Handle(path, NewTimer(h.logger, handler))
}

// HandlePrefix registers a new route with a matcher for the URL path.
// See Route.Prefix() and Route.Handler(). Wrapper around mux.router.
func (h *Handler) HandlePrefix(prefix string, handler http.Handler) *mux.Route {
	return h.router.PathPrefix(prefix).Handler(NewTimer(h.logger, handler))
}

// HandleFunc registers a new route with a matcher for the URL path.
// See Route.Path() and Route.HandlerFunc(). Wrapper around mux.router.
func (h *Handler) HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) *mux.Route {
	return h.Handle(path, http.HandlerFunc(f))
}

// HandleFuncPrefix registers a new route with a matcher for the URL path.
// See Route.Prefix() and Route.HandleFunc(). Wrapper around mux.router.
func (h *Handler) HandleFuncPrefix(prefix string, f func(http.ResponseWriter, *http.Request)) *mux.Route {
	return h.HandlePrefix(prefix, http.HandlerFunc(f))
}

// AddOkHandler adds an OKHandler to the given path. Useful for custom uptime check or health check URLs.
func (h *Handler) AddOkHandler(path string) {
	h.router.Handle(path, h.okHandler)
}

func (h *Handler) registerDefaultRoutes() {
	h.router.Handle("/_healthz", h.okHandler)
	h.router.Handle("/_health", h.okHandler)
	h.router.Handle("/up", h.okHandler)
}

func (h *Handler) wrapHandlers(logger logrus.FieldLogger, handler http.Handler) http.Handler {
	handler = handlers.RecoveryHandler(
		handlers.RecoveryLogger(logger),
		handlers.PrintRecoveryStack(true),
	)(handler)

	return handler
}

// Serve starts the server and blocks until the process is terminated by signals
func (h *Handler) Serve() {
	h.registerDefaultRoutes()
	h.server.Handler = h.wrapHandlers(h.logger, h.router)

	h.logger.Infof("Listening on %v", h.server.Addr)

	go func() {
		srv := &http.Server{
			Addr:    h.metricsListen,
			Handler: promhttp.Handler(),
		}
		if err := srv.ListenAndServe(); err != nil {
			h.logger.Fatalf("failed to listen on metrics port %v: %v", h.server.Addr, err)
		}
	}()

	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.Fatalf("failed to listen on %v: %v", h.server.Addr, err)
		}
	}()

	// wait for signals
	<-h.ctx.Done()
	h.logger.Info("Shutting down ...")

	// grant the http server a certain time to finish handling of http requests
	shutdownCtx, cancel := context.WithTimeout(context.Background(), h.gracefulShutdownDuration)
	defer func() {
		cancel()
	}()

	if err := h.server.Shutdown(shutdownCtx); err != nil {
		h.logger.Fatalf("failed to shutdown http server: %v", err)
	}

	<-shutdownCtx.Done()
}
