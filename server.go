package into

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

var (
	// DefaultServerAddress is the default address for the health check server.
	//
	// It binds every interface on purpose: orchestrators such as Kubernetes
	// dial HTTP probes against the pod address, not loopback. The kill endpoint
	// therefore denies every request until an authorization check is registered,
	// see SetKillHeaderCheck.
	DefaultServerAddress = "0.0.0.0:18080"
	// DefaultRequestShutdownTimeout is the default timeout for health check server shutdown.
	DefaultRequestShutdownTimeout = 5 * time.Second
	// DefaultReadHeaderTimeout bounds how long a client may take to send its
	// request headers.
	DefaultReadHeaderTimeout = 5 * time.Second
)

type optionServer struct {
	healthCheckOption *optionHealthCheck
	killOption        *optionKill

	serverAddress string
}

type OptionServer func(options *optionServer)

// WithServerAddress sets the address for server.
//   - addr: The address to listen on (e.g., "0.0.0.0:18080").
//     DefaultServerAddress
func WithServerAddress(addr string) OptionServer {
	return func(options *optionServer) {
		options.serverAddress = addr
	}
}

// resolveServerAddress returns the address of the management server, falling
// back to DefaultServerAddress.
//
// The fallback lives here rather than being written into the option, because the
// --health command reads the address before the server is started.
func resolveServerAddress(opt *optionServer) string {
	if opt == nil || opt.serverAddress == "" {
		return DefaultServerAddress
	}

	return opt.serverAddress
}

// localDialAddress rewrites a wildcard bind address into one that can be dialled
// from this host, so the --health command works against the default address.
func localDialAddress(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}

	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return net.JoinHostPort("127.0.0.1", port)
	}

	return addr
}

// serve starts the management server when a handler is registered.
//
// Binding happens synchronously: an occupied or misconfigured port is reported
// to the caller instead of leaving the service running without the endpoints an
// orchestrator probes.
func serve(ctx context.Context, opt *optionServer) error {
	if opt == nil {
		return nil
	}

	address := resolveServerAddress(opt)

	mux := http.NewServeMux()

	handlerAdded := false

	// register health check handler
	if healthCheckHandler(mux, opt) {
		handlerAdded = true
	}
	// register kill handler
	if killHandler(mux, opt) {
		handlerAdded = true
	}

	if !handlerAdded {
		return nil
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("init server listen %s: %w", address, err)
	}

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
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
		logger.Info("init server starting", "addr", address)

		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("init server error: " + err.Error())
		}
	}()

	return nil
}
