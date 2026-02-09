package into

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	// DefaultHealthCheckPath is the path for the health check endpoint.
	DefaultHealthCheckPath = "/healthz"
	// DefaultHealthCheckTimeout is the default timeout for health check requests.
	DefaultHealthCheckTimeout = 5 * time.Second
)

// HealthCheck is a function that performs a health check.
//   - It should return nil if the application is healthy, or an error otherwise.
var HealthCheck = func(ctx context.Context) error {
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
		if HealthCheck != nil {
			if err := HealthCheck(r.Context()); err != nil {
				http.Error(w, "Unhealthy: "+err.Error(), http.StatusServiceUnavailable)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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
	healthCheckURL := "http://" + strings.Trim(opt.serverAddress, "/") + "/" + strings.Trim(optHealthCheck.Path, "/")

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
