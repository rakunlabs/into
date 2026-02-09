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

To use custom health check endpoint, you can use `into.WithHealthCheckCustom()` option.  
This will not start the health check server, but you can call the health check function directly from your code.

```go
into.Init(run,
	into.WithMsgf("myservice [%s]", "v0.1.0"),
	into.WithHealthCheck(into.WithHealthCheckCustom()),
)
```

call custom health check endpoint:

```sh
go run main.go --health-check http://localhost:18080/healthz
```

### Kill Endpoint

You can enable kill endpoint by adding `into.WithKill()` option.

```go
into.Init(run,
	into.WithMsgf("myservice [%s]", "v0.1.0"),
	into.WithHealthCheck(),
	into.WithKill(),
)
```

Add extra header check for kill endpoint:

```go
into.KillHeaderCheck = func(h http.Header) bool {
	// check for custom header
	return h.Get("X-Kill-Secret") == "mysecret"
}

// Or use predefined map check
into.KillHeaderCheckMap(map[string]string{
	"X-Kill-Secret": "mysecret",
})
```

Example CURL request to call kill endpoint:

```sh
curl -X POST -H "X-Kill-Secret: mysecret" http://localhost:18080/kill
```
