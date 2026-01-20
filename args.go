package into

import (
	"context"
	"os"
	"strings"
)

type flagStr string

var healthCheckFlag flagStr = "health"

func parseFlag() flagStr {
	for _, arg := range os.Args {
		switch strings.ToLower(arg) {
		case "-health-check", "--health-check":
			return healthCheckFlag
		}
	}

	return ""
}

func commands(ctx context.Context, opt *option) {
	switch parseFlag() {
	case healthCheckFlag:
		callHealthCheck(ctx, opt.healthCheckOptions)
	}
}
