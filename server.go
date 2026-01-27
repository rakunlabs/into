package into

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var (
	// DefaultServerAddress is the default address for the health check server.
	DefaultServerAddress = "127.0.0.1:18080"
	// DefaultRequestShutdownTimeout is the default timeout for health check server shutdown.
	DefaultRequestShutdownTimeout = 5 * time.Second
)

type optionServer struct {
	healthCheckOption *optionHealthCheck
	killOption        *optionKill

	serverAddress *string
}

type OptionServer func(options *optionServer)

// WithServerAddress sets the address for server.
//   - addr: The address to listen on (e.g., "127.0.0.1:18080").
//     DefaultServerAddress
func WithServerAddress(addr string) OptionServer {
	return func(options *optionServer) {
		options.serverAddress = &addr
	}
}

func serve(ctx context.Context, opt *optionServer) {
	if opt == nil {
		return
	}

	if opt.serverAddress == nil {
		opt.serverAddress = &DefaultServerAddress
	}

	mux := http.NewServeMux()
	// register health check handler
	healthCheckHandler(mux, opt)
	// register kill handler
	killHandler(mux, opt)

	server := &http.Server{
		Addr:    *opt.serverAddress,
		Handler: mux,
	}

	// Shutdown the server when the context is done.
	context.AfterFunc(ctx, func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultRequestShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("init server shutdown error: " + err.Error())
		}
	})

	// Start the server in a separate goroutine.
	go func() {
		logger.Info("init server starting", "addr", *opt.serverAddress)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("init server error: " + err.Error())
		}
	}()
}
