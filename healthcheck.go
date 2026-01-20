package into

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	// DefaultHealthCheckPath is the path for the health check endpoint.
	DefaultHealthCheckPath = "/healthz"
	// DefaultHealthCheckAddress is the default address for the health check server.
	DefaultHealthCheckAddress = "127.0.0.1:18080"
	// DefaultHealthCheckTimeout is the default timeout for health check requests.
	DefaultHealthCheckTimeout = 5 * time.Second

	// DefaultHealthCheckShutdownTimeout is the default timeout for health check server shutdown.
	DefaultHealthCheckShutdownTimeout = 5 * time.Second
)

// HealthCheck is a function that performs a health check.
//   - It should return nil if the application is healthy, or an error otherwise.
var HealthCheck = func(ctx context.Context) error {
	return nil
}

type optionsHealthCheck struct {
	Address *string
	Path    *string
	Timeout *time.Duration
}

func getHealthCheckOptions(opt *optionsHealthCheck) optionsHealthCheck {
	optHealthCheck := optionsHealthCheck{
		Address: &DefaultHealthCheckAddress,
		Path:    &DefaultHealthCheckPath,
		Timeout: &DefaultHealthCheckTimeout,
	}

	if opt != nil {
		if opt.Address != nil {
			optHealthCheck.Address = opt.Address
		}

		if opt.Path != nil {
			optHealthCheck.Path = opt.Path
		}

		if opt.Timeout != nil {
			optHealthCheck.Timeout = opt.Timeout
		}
	}

	return optHealthCheck
}

type OptionHealthCheck func(options *optionsHealthCheck)

// WithHealthCheckAddress sets the address for the health check server.
//   - addr: The address to listen on (e.g., "127.0.0.1:18080").
//     DefaultHealthCheckAddress
func WithHealthCheckAddress(addr string) OptionHealthCheck {
	return func(options *optionsHealthCheck) {
		options.Address = &addr
	}
}

// WithHealthCheckPath sets the path for the health check endpoint.
//   - path: The path for the health check endpoint (e.g., "/health").
//     DefaultHealthCheckPath
func WithHealthCheckPath(path string) OptionHealthCheck {
	return func(options *optionsHealthCheck) {
		options.Path = &path
	}
}

// WithHealthCheckTimeout sets the timeout for health check requests.
//   - timeout: The timeout duration for health check requests.
//     DefaultHealthCheckTimeout
func WithHealthCheckTimeout(timeout time.Duration) OptionHealthCheck {
	return func(options *optionsHealthCheck) {
		options.Timeout = &timeout
	}
}

// ///////////////////////////////////////////////////////////////////

func startHealthCheckServer(ctx context.Context, opt *optionsHealthCheck) {
	if opt == nil {
		return
	}

	optHealthCheck := getHealthCheckOptions(opt)

	serveHealthCheck(ctx, *optHealthCheck.Address, *optHealthCheck.Path)
}

func serveHealthCheck(ctx context.Context, addr, path string) {
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if HealthCheck != nil {
			if err := HealthCheck(r.Context()); err != nil {
				http.Error(w, "Unhealthy: "+err.Error(), http.StatusServiceUnavailable)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Shutdown the server when the context is done.
	context.AfterFunc(ctx, func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultHealthCheckShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("health check server shutdown error: " + err.Error())
		}
	})

	// Start the server in a separate goroutine.
	go func() {
		logger.Info("starting health check server", "addr", addr, "path", path)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("health check server error: " + err.Error())
		}
	}()
}

// //////////////////////////////////////////////////////////////////

type HealthResponse struct {
	StatusCode int
	Body       string
}

func callHealthCheck(ctx context.Context, opt *optionsHealthCheck) {
	if opt == nil {
		return
	}

	resp, err := callHealthCheckHandler(ctx, opt)
	if err != nil {
		logger.Error("health check call error: " + err.Error())
		os.Exit(1)
	}

	// if not 2xx status code, exit with error
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Error("health check failed", "status_code", resp.StatusCode, "body", resp.Body)
		os.Exit(1)
	}

	logger.Info("health check passed", "status_code", resp.StatusCode, "body", resp.Body)

	os.Exit(0)
}

// callHealthCheckHandler calls the health check endpoint and logs the response.
func callHealthCheckHandler(ctx context.Context, opt *optionsHealthCheck) (*HealthResponse, error) {
	optHealthCheck := getHealthCheckOptions(opt)

	healthCheckURL := "http://" + strings.Trim(*optHealthCheck.Address, "/") + "/" + strings.Trim(*optHealthCheck.Path, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthCheckURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: *optHealthCheck.Timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// read all body to ensure connection reuse
	body, _ := io.ReadAll(resp.Body)

	return &HealthResponse{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}, nil
}
