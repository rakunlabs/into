package into

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

var (
	// DefaultHealthCheckPath is the path for the health check endpoint.
	DefaultHealthCheckPath = "/healthz"
	// DefaultHealthCheckTimeout is the default timeout for health check requests.
	DefaultHealthCheckTimeout = 5 * time.Second
)

// healthCheck holds the registered health check. The HTTP handler reads it from
// the server goroutine while the run function may register it concurrently, so
// it is stored atomically.
var healthCheck atomic.Pointer[func(ctx context.Context) error]

// SetHealthCheck registers the check behind the health check endpoint and the
// --health command. Return nil when the application is healthy, an error
// otherwise.
//
// It is safe to call at any point, including from the run function while the
// endpoint is already serving. Passing nil restores the default, which always
// reports healthy.
func SetHealthCheck(fn func(ctx context.Context) error) {
	if fn == nil {
		healthCheck.Store(nil)

		return
	}

	healthCheck.Store(&fn)
}

// runHealthCheck reports the health of the application. Without a registered
// check it always succeeds.
func runHealthCheck(ctx context.Context) error {
	if fn := healthCheck.Load(); fn != nil {
		return (*fn)(ctx)
	}

	return nil
}

type optionHealthCheck struct {
	EndpointEnabled bool
	CustomURL       bool
	Path            string
	Timeout         time.Duration
}

func getHealthCheckOptions(opt *optionHealthCheck) optionHealthCheck {
	optHealthCheck := optionHealthCheck{
		Path:    DefaultHealthCheckPath,
		Timeout: DefaultHealthCheckTimeout,
	}

	if opt != nil {
		if opt.Path != "" {
			optHealthCheck.Path = opt.Path
		}

		if opt.Timeout != 0 {
			optHealthCheck.Timeout = opt.Timeout
		}
	}

	return optHealthCheck
}

type OptionHealthCheck func(options *optionHealthCheck)

// WithHealthCheckCustom enables only the health-check URL command without starting the health check server.
func WithHealthCheckCustom() OptionHealthCheck {
	return func(options *optionHealthCheck) {
		options.EndpointEnabled = false
		options.CustomURL = true
	}
}

// WithHealthCheckPath sets the path for the health check endpoint.
//   - path: The path for the health check endpoint (e.g., "/healthz").
//     DefaultHealthCheckPath
func WithHealthCheckPath(path string) OptionHealthCheck {
	return func(options *optionHealthCheck) {
		options.Path = path
	}
}

// WithHealthCheckTimeout sets the timeout for health check requests.
//   - timeout: The timeout duration for health check requests.
//     DefaultHealthCheckTimeout
func WithHealthCheckTimeout(timeout time.Duration) OptionHealthCheck {
	return func(options *optionHealthCheck) {
		options.Timeout = timeout
	}
}

// //////////////////////////////////////////////////////////////////

func healthCheckHandler(mux *http.ServeMux, opt *optionServer) bool {
	// for health check endpoint
	if opt.healthCheckOption == nil || !opt.healthCheckOption.EndpointEnabled {
		return false
	}

	optHealthCheck := getHealthCheckOptions(opt.healthCheckOption)

	logger.Info("init health check endpoint registered", "path", optHealthCheck.Path)
	mux.HandleFunc(optHealthCheck.Path, func(w http.ResponseWriter, r *http.Request) {
		if err := runHealthCheck(r.Context()); err != nil {
			http.Error(w, "Unhealthy: "+err.Error(), http.StatusServiceUnavailable)

			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	return true
}

// //////////////////////////////////////////////////////////////////

type HealthResponse struct {
	StatusCode int
	Body       string
}

func callHealthCheck(ctx context.Context, opt *optionServer) {
	if opt == nil || opt.healthCheckOption == nil || !opt.healthCheckOption.EndpointEnabled {
		return
	}

	optHealthCheck := getHealthCheckOptions(opt.healthCheckOption)
	// The server is not running yet in this process, so dial the address it will
	// bind, with a wildcard host rewritten to loopback.
	address := localDialAddress(resolveServerAddress(opt))
	healthCheckURL := "http://" + strings.Trim(address, "/") + "/" + strings.Trim(optHealthCheck.Path, "/")

	callHealthCheckWithURL(ctx, healthCheckURL, optHealthCheck.Timeout)
}

func callHealthCheckWithURL(ctx context.Context, healthCheckURL string, timeout time.Duration) {
	resp, err := callHealthCheckHandler(ctx, healthCheckURL, timeout)
	if err != nil {
		logger.Error("health check call", "error", err.Error())
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

func callHealthCheckHandler(ctx context.Context, healthCheckURL string, timeout time.Duration) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthCheckURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: timeout,
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
