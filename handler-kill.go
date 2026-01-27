package into

import "net/http"

var (
	// DefaultKillPath is the path for the kill endpoint.
	DefaultKillPath = "/kill"
)

type optionKill struct{}

type OptionKill func(options *optionKill)

func killHandler(mux *http.ServeMux, opt *optionServer) {
	if opt.killOption == nil {
		return
	}

	logger.Info("init kill endpoint registered", "path", DefaultKillPath)
	mux.HandleFunc(DefaultKillPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)

			return
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Shutting down.."))

		CtxCancel()
	})
}
