package httputils

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	_ http.Handler = (*Timer)(nil)

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_duration_seconds",
		Help: "Duration of HTTP requests",
	}, []string{"path"})
)

// Timer measures request durations for the nested handler
type Timer struct {
	handler http.Handler
	logger  logrus.FieldLogger
}

// NewTimer returns a new Timer instance
func NewTimer(logger logrus.FieldLogger, handler http.Handler) http.Handler {
	return &Timer{
		handler: handler,
		logger:  logger,
	}
}

// ServeHTTP times the request and collects request infos
func (h *Timer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route := mux.CurrentRoute(r)
	path, _ := route.GetPathTemplate()
	timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))

	h.handler.ServeHTTP(rw, r)

	timer.ObserveDuration()
}
