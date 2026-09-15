//go:build unix

package into

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestSignalExitCode(t *testing.T) {
	tests := []struct {
		name string
		sig  os.Signal
		want int32
	}{
		{name: "sigint", sig: syscall.SIGINT, want: 130},
		{name: "sigterm", sig: syscall.SIGTERM, want: 143},
		{name: "sighup", sig: syscall.SIGHUP, want: 129},
		{name: "non syscall signal", sig: fakeSignal{}, want: 128},
		{name: "zero signal", sig: syscall.Signal(0), want: 128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := signalExitCode(tt.sig); got != tt.want {
				t.Fatalf("signalExitCode(%v) = %d, want %d", tt.sig, got, tt.want)
			}
		})
	}
}

func TestResolveExitCode(t *testing.T) {
	tests := []struct {
		name    string
		failure int32
		signal  int32
		def     int
		want    int
	}{
		{name: "clean run uses default", want: 0},
		{name: "custom default", def: 7, want: 7},
		{name: "signal only", signal: 143, want: 143},
		{name: "failure only", failure: 1, want: 1},
		{name: "failure outranks signal", failure: 1, signal: 143, want: 1},
		{name: "failure outranks default", failure: 2, def: 7, want: 2},
		{name: "signal outranks default", signal: 130, def: 7, want: 130},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveExitCode(tt.failure, tt.signal, tt.def); got != tt.want {
				t.Fatalf("resolveExitCode(%d, %d, %d) = %d, want %d",
					tt.failure, tt.signal, tt.def, got, tt.want)
			}
		})
	}
}

// TestInitExitStatus verifies the real process exit status. Init calls os.Exit,
// so every case runs in a re-executed copy of this test binary.
func TestInitExitStatus(t *testing.T) {
	tests := []struct {
		name string
		mode string
		sig  syscall.Signal
		want int
	}{
		{name: "clean return", mode: "clean", want: 0},
		{name: "sigterm is 128+15", mode: "block", sig: syscall.SIGTERM, want: 143},
		{name: "sigint is 128+2", mode: "block", sig: syscall.SIGINT, want: 130},
		{name: "run error", mode: "error", want: 1},
		{name: "shutdown timeout outranks signal", mode: "timeout", sig: syscall.SIGTERM, want: 1},
		{name: "custom signal exit code", mode: "customsignal", sig: syscall.SIGTERM, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runInitChild(t, tt.mode, tt.sig); got != tt.want {
				t.Fatalf("exit status = %d, want %d", got, tt.want)
			}
		})
	}
}

const childReadyMarker = "into-child-ready"

// runInitChild re-executes this test binary in child mode and returns its exit
// status. When sig is non-zero it is sent only after the child reports that
// Init already installed its signal handler, so the default signal disposition
// can never race the test.
func runInitChild(t *testing.T, mode string, sig syscall.Signal) int {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^TestInitExitStatus$")
	cmd.Env = append(os.Environ(), "INTO_TEST_CHILD_MODE="+mode)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}

	ready := make(chan struct{})

	go func() {
		defer close(ready)

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if scanner.Text() == childReadyMarker {
				return
			}
		}
	}()

	if sig != 0 {
		select {
		case <-ready:
		case <-time.After(30 * time.Second):
			_ = cmd.Process.Kill()
			t.Fatal("child never reported ready")
		}

		if err := cmd.Process.Signal(sig); err != nil {
			t.Fatalf("signal child: %v", err)
		}
	}

	err = cmd.Wait()

	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		if exitErr.ExitCode() < 0 {
			t.Fatalf("child was killed by a signal instead of exiting: %v", err)
		}

		return exitErr.ExitCode()
	case err != nil:
		t.Fatalf("wait child: %v", err)
	}

	return 0
}

// TestMain dispatches child processes into Init before the test framework runs.
func TestMain(m *testing.M) {
	mode := os.Getenv("INTO_TEST_CHILD_MODE")
	if mode == "" {
		os.Exit(m.Run())
	}

	opts := []Option{WithLogger(LogNoop{})}

	if mode == "customsignal" {
		opts = append(opts, WithSignalExitCode(func(os.Signal) int32 { return 1 }))
	}

	if mode == "timeout" {
		// Shorter than the goroutine leaked below, so shutdown must time out.
		opts = append(opts, WithWaitTimeout(100*time.Millisecond))
	}

	// Init always calls os.Exit, so this never returns.
	Init(func(ctx context.Context) error {
		switch mode {
		case "clean":
			return nil
		case "error":
			return errors.New("boom")
		case "timeout":
			leakGoroutine(ctx)
		}

		// Init installs the signal handler before calling this function, so
		// announcing readiness here is safe.
		fmt.Println(childReadyMarker)

		<-ctx.Done()

		return nil
	}, opts...)
}

// leakGoroutine registers a goroutine on the into wait group that outlives the
// configured shutdown timeout.
func leakGoroutine(ctx context.Context) {
	wg := WaitGroup(ctx)
	if wg == nil {
		return
	}

	wg.Add(1)

	go func() {
		defer wg.Done()

		time.Sleep(time.Minute)
	}()
}

type fakeSignal struct{}

func (fakeSignal) String() string { return "fake" }
func (fakeSignal) Signal()        {}
