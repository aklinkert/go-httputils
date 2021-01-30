package httputils

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

var _ http.Handler = &OkHandler{}

// OkHandler handles http requests and answers with "ok" :)
type OkHandler struct {
	logger logrus.FieldLogger
}

// NewOkHandler returns a new OkHandler instance
func NewOkHandler(logger logrus.FieldLogger) *OkHandler {
	return &OkHandler{
		logger: logger,
	}
}

// ServeHTTP actually answers the http request. Surprising, isn't it?
func (h *OkHandler) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	if _, err := fmt.Fprintf(rw, "ok"); err != nil {
		h.logger.Debugf("failed to write ok response: %v", err)
	}
}
