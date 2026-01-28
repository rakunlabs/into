package into

import "net/http"

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

// KillHeaderCheck is a function that checks the headers of a kill request.
//   - It should return true if the request is authorized, or false otherwise.
var KillHeaderCheck = func(h http.Header) bool {
	return true
}

// KillHeaderCheckMap is a map of header keys and expected values for kill request authorization.
//   - If any of the headers match, the request is authorized.
//   - If the map is empty, all requests are authorized.
func KillHeaderCheckMap(headerMap map[string]string) {
	KillHeaderCheck = func(h http.Header) bool {
		if len(headerMap) == 0 {
			return true
		}

		for key, expectedValue := range headerMap {
			if h.Get(key) == expectedValue {
				return true
			}
		}

		return false
	}
}

func killHandler(mux *http.ServeMux, opt *optionServer) {
	if opt.killOption == nil {
		return
	}

	killOpt := getKillOptions(opt.killOption)

	logger.Info("init kill endpoint registered", "path", killOpt.Path)
	mux.HandleFunc(killOpt.Path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)

			return
		}

		if KillHeaderCheck != nil && !KillHeaderCheck(r.Header) {
			http.Error(w, "Forbidden kill request", http.StatusForbidden)

			return
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Shutting down.."))

		logger.Info("init kill endpoint triggered, shutting down service")
		CtxCancel()
	})
}
