# into

[![Go PKG](https://raw.githubusercontent.com/rakunlabs/.github/main/assets/badges/gopkg.svg)](https://pkg.go.dev/github.com/rakunlabs/into)

Helper function to initiate the project easly.

## Usage

```go
func main() {
	into.Run(run,
		into.WithLogger(slog.Default()),
		into.WithMsgf("myservice [%s]", "v0.1.0"),
	)
}

func run(ctx context.Context) error {
	return nil
}
```
