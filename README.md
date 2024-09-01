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
	into.Run(run,
		into.WithLogger(slog.Default()),
		into.WithMsgf("myservice [%s]", "v0.1.0"),
	)
}

func run(ctx context.Context) error {
	return nil
}
```
