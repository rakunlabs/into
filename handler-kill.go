package into

import (
	"maps"
	"net/http"
	"sync/atomic"
)

// DefaultKillPath is the path for the kill endpoint.
var DefaultKillPath = "/kill"

type optionKill struct {
	Path string
}

type OptionKill func(options *optionKill)

func WithKillPath(path string) OptionKill {
	return func(options *optionKill) {
		options.Path = path
	}
}

func getKillOptions(opt *optionKill) optionKill {
	optKill := optionKill{
		Path: DefaultKillPath,
	}

	if opt != nil {
		if opt.Path != "" {
			optKill.Path = opt.Path
		}
	}

	return optKill
}

// killHeaderCheck authorizes kill requests. A nil value means no check has been
// registered and every request is denied: this endpoint stops the process, so
// it fails closed rather than open.
//
// The HTTP handler reads it from the server goroutine while the run function may
// register it concurrently, so it is stored atomically.
var killHeaderCheck atomic.Pointer[func(h http.Header) bool]

// SetKillHeaderCheck registers the authorization check for the kill endpoint.
// Return true to authorize the request.
//
// Until a check is registered every kill request is denied with 403, because the
// endpoint is served on DefaultServerAddress (all interfaces) so that
// orchestrator health probes can reach the health endpoint on the same port.
//
// It is safe to call at any point, including from the run function while the
// endpoint is already serving. Passing nil restores the deny-all default.
func SetKillHeaderCheck(fn func(h http.Header) bool) {
	if fn == nil {
		killHeaderCheck.Store(nil)

		return
	}

	killHeaderCheck.Store(&fn)
}

// KillHeaderCheckMap authorizes a kill request when any of the given headers
// matches its expected value.
//   - An empty map authorizes every request.
func KillHeaderCheckMap(headerMap map[string]string) {
	// Copy so a later mutation by the caller cannot race the handler.
	headers := make(map[string]string, len(headerMap))
	maps.Copy(headers, headerMap)

	SetKillHeaderCheck(func(h http.Header) bool {
		if len(headers) == 0 {
			return true
		}

		for key, expectedValue := range headers {
			if h.Get(key) == expectedValue {
				return true
			}
		}

		return false
	})
}

func killHandler(mux *http.ServeMux, opt *optionServer) bool {
	if opt.killOption == nil {
		return false
	}

	killOpt := getKillOptions(opt.killOption)

	logger.Info("init kill endpoint registered", "path", killOpt.Path)
	mux.HandleFunc(killOpt.Path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)

			return
		}

		check := killHeaderCheck.Load()
		if check == nil {
			logger.Error("kill endpoint denied a request: no authorization check is registered",
				"hint", "call into.SetKillHeaderCheck or into.KillHeaderCheckMap")
			http.Error(w, "Forbidden kill request", http.StatusForbidden)

			return
		}

		if !(*check)(r.Header) {
			http.Error(w, "Forbidden kill request", http.StatusForbidden)

			return
		}

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("Shutting down.."))

		logger.Info("init kill endpoint triggered, shutting down service")
		CtxCancel()
	})

	return true
}
