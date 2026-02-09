package into

import (
	"context"
	"os"
	"strings"
)

type flagStr string

var (
	healthFlag      flagStr = "health"
	healthCheckFlag flagStr = "health-check"
)

func parseFlag() flagStr {
	for _, arg := range os.Args {
		switch strings.ToLower(arg) {
		case "-health", "--health":
			return healthFlag
		case "-health-check", "--health-check":
			return healthCheckFlag
		}
	}

	return ""
}

func getValue(fl flagStr) string {
	for i, arg := range os.Args {
		switch fl {
		case healthCheckFlag:
			switch strings.ToLower(arg) {
			case "-health-check", "--health-check":
				// Check if there's a next argument (the URL)
				if i+1 < len(os.Args) {
					return os.Args[i+1]
				}
			}
		}
	}

	return ""
}

func commands(ctx context.Context, opt *option) {
	switch parseFlag() {
	case healthFlag:
		callHealthCheck(ctx, opt.serverOptions)
	case healthCheckFlag:
		if opt.serverOptions == nil || opt.serverOptions.healthCheckOption == nil || opt.serverOptions.healthCheckOption.CustomURL == false {
			logger.Error("--health-check requires WithHealthCheckCustom option to be set")
			os.Exit(1)
		}

		healthCheckURL := getValue(healthCheckFlag)
		if healthCheckURL == "" {
			logger.Error("--health-check requires a URL argument")
			os.Exit(1)
		}

		callHealthCheckWithURL(ctx, healthCheckURL, DefaultHealthCheckTimeout)
	}
}
