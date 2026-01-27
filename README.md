# into

[![License](https://img.shields.io/github/license/rakunlabs/into?color=red&style=flat-square)](https://raw.githubusercontent.com/rakunlabs/into/main/LICENSE)
[![Go PKG](https://raw.githubusercontent.com/rakunlabs/.github/main/assets/badges/gopkg.svg)](https://pkg.go.dev/github.com/rakunlabs/into)
[![Go Report Card](https://goreportcard.com/badge/github.com/rakunlabs/into?style=flat-square)](https://goreportcard.com/report/github.com/rakunlabs/into)

Helper function to initiate the project easly.

```sh
go get github.com/rakunlabs/into
```

## Usage

```go
func main() {
	into.Init(run,
		into.WithMsgf("myservice [%s]", "v0.1.0"),
	)
}

func run(ctx context.Context) error {
	return nil
}
```

### Health Check

You can enable health check endpoint by adding `into.WithHealthCheck()` option.

```go
into.Init(run,
	into.WithMsgf("myservice [%s]", "v0.1.0"),
	into.WithHealthCheck(),
	into.WithKill(),
)


// The health check endpoint will be available at /healthz
// Set function to respond to health check requests.
into.HealthCheck = func(ctx context.Context) error {
	return errors.New("some problem")
}
```

Call health check from your command line:

```sh
go run main.go --health
```
