# go-httputils

This package provides some utility wrapper for the HTTP package. The major protagonist ist the Handler object, which
wraps the default http.Server with a mux.Router and takes care to measure durations with prometheus for every HTTP call.

## Usage example

```go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/aklinkert/go-httputils"
)

func main() {
	// handler is based on logrus for logging
	logger := logrus.New()
	httpLogger := logger.WithField("component", "http")

	// first address is the main http listener, second one is for the prometheus endpoint
	// I usually have both on different ports to have the prometheus metrics endpoint not exposed but 
	// only serve internal traffic, e.g. only inside a kubernetes cluster or other private network.
	handler := httputils.NewHandler(httpLogger, ":8080", ":9090")

	// handler provides some useful Handle*() methods that support both http.HandlerFunc http.Handler using
	// Handle() and HandleFunc() respectively. Also HandlePrefix() and HandleFuncPrefix() support a prefixed
	// catch-all approach, e.g. when you wanna do dynamic routes that are not handled by mux (serving images or so)
	handler.HandleFunc("/example", func(rw http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(rw, "Hello from example, current time is %v", time.Now())
		if err != nil {
			logger.Error(err)
		}
	})

	// default graceful shutdown duration is 2 seconds
	handler.SetGracefulShutdownDuration(10 * time.Second)

	// Serve() is blocking the main routine until a shutdown is received, using https://github.com/aklinkert/go-exitcontext
	// If you want to modify the context behavior use httputils.NewHandlerWithContext(...)
	handler.Serve()
}
```

## License

    Apache 2.0 Licence

