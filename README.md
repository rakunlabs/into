# into

[![License](https://img.shields.io/github/license/rakunlabs/into?color=red&style=flat-square)](https://raw.githubusercontent.com/rakunlabs/into/main/LICENSE)
[![Go PKG](https://raw.githubusercontent.com/rakunlabs/.github/main/assets/badges/gopkg.svg)](https://pkg.go.dev/github.com/rakunlabs/into)

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

### Exit Codes

| Situation | Code |
| --- | --- |
| `run` returned `nil` | `DefaultExitCode` (`0`) |
| `run` returned an error | `1`, or `WithErrExitCode` |
| Shutdown exceeded the wait-group timeout | `1` |
| `SIGTERM` received, shutdown completed | `143` (`128+15`) |
| `SIGINT` received, shutdown completed | `130` (`128+2`) |

A graceful shutdown that ran to completion is not a failure, so it reports the
conventional `128+signum` instead of the generic failure code `1`. That keeps a
shutdown which timed out distinguishable from one that finished cleanly. A run
error or a timeout always outranks the signal code.

Tell your supervisor that the signal code is expected. With systemd:

```ini
[Service]
SuccessExitStatus=143
```

Otherwise the unit is left in `failed` state after every intentional
`systemctl stop`.

> Before `v0.6.0` any caught signal produced exit code `1`. To keep that
> behaviour, use `into.WithSignalExitCode(func(os.Signal) int32 { return 1 })`.

### Health Check

You can enable health check endpoint by adding `into.WithHealthCheck()` option.
The endpoint is served at `/healthz` on `DefaultServerAddress`.

```go
func main() {
	into.Init(run,
		into.WithMsgf("myservice [%s]", "v0.1.0"),
		into.WithHealthCheck(),
	)
}

func run(ctx context.Context) error {
	// Register the function answering health check requests. Safe to call while
	// the endpoint is already serving.
	into.SetHealthCheck(func(ctx context.Context) error {
		return errors.New("some problem")
	})

	<-ctx.Done()

	return nil
}
```

Register it inside `run`, not after `into.Init`: `Init` never returns.

Call health check from your command line, which is what an `exec` probe does:

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
# the equals form works too
go run main.go --health-check=http://localhost:18080/healthz
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

The kill endpoint stops the process, and it is served on every interface so that
orchestrator probes can reach the health endpoint on the same port. It therefore
**denies every request with 403 until an authorization check is registered**:

```go
func run(ctx context.Context) error {
	into.SetKillHeaderCheck(func(h http.Header) bool {
		// check for custom header
		return h.Get("X-Kill-Secret") == "mysecret"
	})

	// Or use the predefined map check: authorized when any header matches.
	into.KillHeaderCheckMap(map[string]string{
		"X-Kill-Secret": "mysecret",
	})

	<-ctx.Done()

	return nil
}
```

Both are safe to call while the endpoint is already serving.
`into.KillHeaderCheckMap(nil)` authorizes every request, so use it only when the
address is already restricted with `into.WithServerAddress`.

Example CURL request to call kill endpoint:

```sh
curl -X POST -H "X-Kill-Secret: mysecret" http://localhost:18080/kill
```

## Migrating to v0.6.0

- A caught signal now exits with `128+signum` (`143` for `SIGTERM`) instead of
  `1`. Add `SuccessExitStatus=143` to your systemd unit, or restore the old
  behaviour with `into.WithSignalExitCode(func(os.Signal) int32 { return 1 })`.
- `into.HealthCheck = fn` became `into.SetHealthCheck(fn)`, and
  `into.KillHeaderCheck = fn` became `into.SetKillHeaderCheck(fn)`. The variables
  were read from the server goroutine while being assigned from `run`, which is a
  data race.
- The kill endpoint denies requests until a check is registered. Previously it
  authorized everyone by default while listening on all interfaces.
- `into.WithHealthCheck()` now actually registers the endpoint. It was gated on a
  flag that nothing ever enabled, so `/healthz` answered 404 and `--health`
  silently started the service instead of probing it.
- A failure to bind the management server is now fatal instead of being logged
  from a background goroutine.
- The minimum Go version is 1.24. `log/slog` already required 1.21.
