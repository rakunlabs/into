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

// args returns the arguments without the program name.
func args() []string {
	if len(os.Args) < 2 {
		return nil
	}

	return os.Args[1:]
}

// flagName normalizes an argument to its flag name, dropping an "=value" suffix.
func flagName(arg string) string {
	if i := strings.IndexByte(arg, '='); i >= 0 {
		arg = arg[:i]
	}

	return strings.ToLower(arg)
}

func parseFlag() flagStr {
	for _, arg := range args() {
		switch flagName(arg) {
		case "-health", "--health":
			return healthFlag
		case "-health-check", "--health-check":
			return healthCheckFlag
		}
	}

	return ""
}

// getValue returns the value of a flag, accepting both "--flag value" and
// "--flag=value".
func getValue(fl flagStr) string {
	if fl != healthCheckFlag {
		return ""
	}

	list := args()

	for i, arg := range list {
		switch flagName(arg) {
		case "-health-check", "--health-check":
			// Cut on the raw argument so the value keeps its original case.
			if eq := strings.IndexByte(arg, '='); eq >= 0 {
				return arg[eq+1:]
			}

			// Otherwise the value is the next argument.
			if i+1 < len(list) {
				return list[i+1]
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
