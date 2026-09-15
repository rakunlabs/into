package into

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func listenLocal() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

// goroutineCount reports the goroutine count once it stops changing, so finished
// goroutines are not counted as leaks.
func goroutineCount() int {
	last := runtime.NumGoroutine()

	for range 20 {
		time.Sleep(10 * time.Millisecond)

		current := runtime.NumGoroutine()
		if current == last {
			return current
		}

		last = current
	}

	return last
}

// TestHealthCheckEndpointIsRegistered guards the regression where the endpoint
// gate was never enabled, so WithHealthCheck produced a 404.
func TestHealthCheckEndpointIsRegistered(t *testing.T) {
	opt := newOption(WithHealthCheck())

	if opt.serverOptions == nil || opt.serverOptions.healthCheckOption == nil {
		t.Fatal("WithHealthCheck did not create the health check option")
	}

	if !opt.serverOptions.healthCheckOption.EndpointEnabled {
		t.Fatal("WithHealthCheck left the endpoint disabled")
	}

	mux := http.NewServeMux()
	if !healthCheckHandler(mux, opt.serverOptions) {
		t.Fatal("healthCheckHandler did not register the endpoint")
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, DefaultHealthCheckPath, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestHealthCheckCustomKeepsEndpointDisabled keeps WithHealthCheckCustom opting
// out of the server, which is the point of that option.
func TestHealthCheckCustomKeepsEndpointDisabled(t *testing.T) {
	opt := newOption(WithHealthCheck(WithHealthCheckCustom()))

	if opt.serverOptions.healthCheckOption.EndpointEnabled {
		t.Fatal("WithHealthCheckCustom enabled the endpoint")
	}

	if !opt.serverOptions.healthCheckOption.CustomURL {
		t.Fatal("WithHealthCheckCustom did not set CustomURL")
	}

	if healthCheckHandler(http.NewServeMux(), opt.serverOptions) {
		t.Fatal("healthCheckHandler registered a disabled endpoint")
	}
}

func TestHealthCheckReportsUnhealthy(t *testing.T) {
	t.Cleanup(func() { SetHealthCheck(nil) })

	SetHealthCheck(func(context.Context) error {
		return errors.New("some problem")
	})

	mux := http.NewServeMux()
	healthCheckHandler(mux, newOption(WithHealthCheck()).serverOptions)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, DefaultHealthCheckPath, nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	if !strings.Contains(rec.Body.String(), "some problem") {
		t.Fatalf("body = %q, want it to mention the failure", rec.Body.String())
	}

	SetHealthCheck(nil)

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, DefaultHealthCheckPath, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("after reset status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestKillEndpointDeniesWithoutCheck pins the fail-closed default: the endpoint
// is served on every interface, so an unconfigured check must not authorize.
func TestKillEndpointDeniesWithoutCheck(t *testing.T) {
	t.Cleanup(func() { SetKillHeaderCheck(nil) })

	SetKillHeaderCheck(nil)

	mux := http.NewServeMux()
	if !killHandler(mux, newOption(WithKill()).serverOptions) {
		t.Fatal("killHandler did not register the endpoint")
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, DefaultKillPath, nil))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestKillEndpointAuthorization(t *testing.T) {
	t.Cleanup(func() { SetKillHeaderCheck(nil) })

	tests := []struct {
		name    string
		headers map[string]string
		method  string
		send    map[string]string
		want    int
	}{
		{
			name:    "matching header",
			headers: map[string]string{"X-Kill-Secret": "mysecret"},
			method:  http.MethodPost,
			send:    map[string]string{"X-Kill-Secret": "mysecret"},
			want:    http.StatusAccepted,
		},
		{
			name:    "wrong value",
			headers: map[string]string{"X-Kill-Secret": "mysecret"},
			method:  http.MethodPost,
			send:    map[string]string{"X-Kill-Secret": "nope"},
			want:    http.StatusForbidden,
		},
		{
			name:    "missing header",
			headers: map[string]string{"X-Kill-Secret": "mysecret"},
			method:  http.MethodPost,
			want:    http.StatusForbidden,
		},
		{
			name:    "empty map authorizes",
			headers: map[string]string{},
			method:  http.MethodPost,
			want:    http.StatusAccepted,
		},
		{
			name:    "get is rejected",
			headers: map[string]string{},
			method:  http.MethodGet,
			want:    http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			KillHeaderCheckMap(tt.headers)

			mux := http.NewServeMux()
			killHandler(mux, newOption(WithKill()).serverOptions)

			req := httptest.NewRequest(tt.method, DefaultKillPath, nil)
			for k, v := range tt.send {
				req.Header.Set(k, v)
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

// TestKillHeaderCheckMapCopiesInput makes sure mutating the caller's map after
// registration cannot change authorization.
func TestKillHeaderCheckMapCopiesInput(t *testing.T) {
	t.Cleanup(func() { SetKillHeaderCheck(nil) })

	headers := map[string]string{"X-Kill-Secret": "mysecret"}
	KillHeaderCheckMap(headers)

	headers["X-Kill-Secret"] = "changed"

	mux := http.NewServeMux()
	killHandler(mux, newOption(WithKill()).serverOptions)

	req := httptest.NewRequest(http.MethodPost, DefaultKillPath, nil)
	req.Header.Set("X-Kill-Secret", "mysecret")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
}

func TestResolveServerAddress(t *testing.T) {
	tests := []struct {
		name string
		opt  *optionServer
		want string
	}{
		{name: "nil option", opt: nil, want: DefaultServerAddress},
		{name: "empty address", opt: &optionServer{}, want: DefaultServerAddress},
		{name: "explicit address", opt: &optionServer{serverAddress: "127.0.0.1:9000"}, want: "127.0.0.1:9000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveServerAddress(tt.opt); got != tt.want {
				t.Fatalf("resolveServerAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLocalDialAddress covers the --health command: a wildcard bind address has
// to be dialled through loopback.
func TestLocalDialAddress(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{name: "ipv4 wildcard", addr: "0.0.0.0:18080", want: "127.0.0.1:18080"},
		{name: "ipv6 wildcard", addr: "[::]:18080", want: "127.0.0.1:18080"},
		{name: "empty host", addr: ":18080", want: "127.0.0.1:18080"},
		{name: "explicit host kept", addr: "10.0.0.5:18080", want: "10.0.0.5:18080"},
		{name: "loopback kept", addr: "127.0.0.1:18080", want: "127.0.0.1:18080"},
		{name: "unparseable kept", addr: "not-an-address", want: "not-an-address"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := localDialAddress(tt.addr); got != tt.want {
				t.Fatalf("localDialAddress(%q) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}

// TestServeReportsBindFailure pins the fail-fast behaviour for an occupied port.
func TestServeReportsBindFailure(t *testing.T) {

	// Hold the port so the bind below must fail.
	listener, err := listenLocal()
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	defer listener.Close()

	opt := newOption(WithHealthCheck(), WithServer(WithServerAddress(listener.Addr().String())))

	if err := serve(context.Background(), opt.serverOptions); err == nil {
		t.Fatal("serve did not report the bind failure")
	}
}

// TestServeWithoutHandlersDoesNotListen keeps the no-handler shortcut.
func TestServeWithoutHandlersDoesNotListen(t *testing.T) {

	if err := serve(context.Background(), nil); err != nil {
		t.Fatalf("serve(nil) = %v, want nil", err)
	}

	// Options present but no endpoint enabled: still nothing to listen on.
	opt := newOption(WithServer(WithServerAddress("127.0.0.1:0")))
	if err := serve(context.Background(), opt.serverOptions); err != nil {
		t.Fatalf("serve without handlers = %v, want nil", err)
	}
}

func TestServeShutsDownWithContext(t *testing.T) {

	t.Cleanup(func() { SetHealthCheck(nil) })

	// A concrete free port is required to probe the endpoint.
	listener, err := listenLocal()
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	addr := listener.Addr().String()
	listener.Close()

	opt := newOption(WithHealthCheck(), WithServer(WithServerAddress(addr)))

	ctx, cancel := context.WithCancel(context.Background())

	if err := serve(ctx, opt.serverOptions); err != nil {
		t.Fatalf("serve: %v", err)
	}

	client := &http.Client{Timeout: 2 * time.Second}

	var resp *http.Response

	for range 50 {
		resp, err = client.Get("http://" + addr + DefaultHealthCheckPath)
		if err == nil {
			break
		}

		time.Sleep(20 * time.Millisecond)
	}

	if err != nil {
		cancel()
		t.Fatalf("health endpoint never answered: %v", err)
	}

	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	cancel()

	// The endpoint must stop answering once the context is done.
	var lastErr error

	for range 100 {
		resp, lastErr = client.Get("http://" + addr + DefaultHealthCheckPath)
		if lastErr != nil {
			break
		}

		resp.Body.Close()
		time.Sleep(20 * time.Millisecond)
	}

	if lastErr == nil {
		t.Fatal("server kept serving after the context was cancelled")
	}
}

func TestParseFlagAndGetValue(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantFlag  flagStr
		wantValue string
	}{
		{name: "no flag", args: []string{"svc"}, wantFlag: ""},
		{name: "health single dash", args: []string{"svc", "-health"}, wantFlag: healthFlag},
		{name: "health double dash", args: []string{"svc", "--health"}, wantFlag: healthFlag},
		{
			name:      "health-check space separated",
			args:      []string{"svc", "--health-check", "http://localhost:1/healthz"},
			wantFlag:  healthCheckFlag,
			wantValue: "http://localhost:1/healthz",
		},
		{
			name:      "health-check equals form",
			args:      []string{"svc", "--health-check=http://localhost:2/healthz"},
			wantFlag:  healthCheckFlag,
			wantValue: "http://localhost:2/healthz",
		},
		{
			name:      "health-check equals keeps value case",
			args:      []string{"svc", "--health-check=http://localhost:3/HealthZ"},
			wantFlag:  healthCheckFlag,
			wantValue: "http://localhost:3/HealthZ",
		},
		{
			name:     "health-check without value",
			args:     []string{"svc", "--health-check"},
			wantFlag: healthCheckFlag,
		},
		{
			name:     "program name is not scanned",
			args:     []string{"--health"},
			wantFlag: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := os.Args
			os.Args = tt.args

			t.Cleanup(func() { os.Args = original })

			if got := parseFlag(); got != tt.wantFlag {
				t.Fatalf("parseFlag() = %q, want %q", got, tt.wantFlag)
			}

			if got := getValue(healthCheckFlag); got != tt.wantValue {
				t.Fatalf("getValue() = %q, want %q", got, tt.wantValue)
			}
		})
	}
}

// TestShutdownRunAllowsAddFromCallback pins that the shutdown mutex is released
// before callbacks run, so ShutdownAdd from a callback cannot deadlock.
func TestShutdownRunAllowsAddFromCallback(t *testing.T) {

	s := &shutdownType{}

	done := make(chan struct{})

	s.Add(func() error {
		s.Add(func() error { return nil }, "nested")
		close(done)

		return nil
	}, "outer")

	finished := make(chan struct{})

	go func() {
		defer close(finished)

		s.Run()
	}()

	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown Run deadlocked")
	}

	select {
	case <-done:
	default:
		t.Fatal("shutdown function did not run")
	}
}

func TestShutdownRunReverseOrderAndNilSafe(t *testing.T) {

	s := &shutdownType{}

	var (
		mutex sync.Mutex
		order []string
	)

	record := func(name string) func() error {
		return func() error {
			mutex.Lock()
			defer mutex.Unlock()

			order = append(order, name)

			return nil
		}
	}

	s.Add(record("first"), "first")
	s.Add(record("second"), "second")
	s.Add(nil, "nil-fn")
	s.Add(func() error { return errors.New("ignored") }, "erroring")

	s.Run()

	mutex.Lock()
	defer mutex.Unlock()

	// A nil function and an erroring one must not stop the rest, and neither
	// records itself, so only the recording pair is expected, reversed.
	if len(order) != 2 || order[0] != "second" || order[1] != "first" {
		t.Fatalf("order = %v, want [second first]", order)
	}
}

// TestCtxCancelIsNilSafe covers CtxCancel before Init wired the cancel function.
func TestCtxCancelIsNilSafe(t *testing.T) {
	s := &shutdownType{}
	s.CtxCancel()
}

func TestTimeoutWaitReturnsOnCompletion(t *testing.T) {

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		time.Sleep(10 * time.Millisecond)
		wg.Done()
	}()

	if newTimeout(10*time.Second, nil).wait(wg) {
		t.Fatal("wait reported a timeout for a completed wait group")
	}
}

func TestTimeoutWaitReportsTimeout(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	t.Cleanup(wg.Done)

	called := false

	if !newTimeout(50*time.Millisecond, func() { called = true }).wait(wg) {
		t.Fatal("wait did not report the timeout")
	}

	if !called {
		t.Fatal("WaitFn was not called on timeout")
	}
}

// TestTimeoutWaitDoesNotLeakTimerGoroutine guards the rewrite: the previous
// implementation left a goroutine blocked on the timer channel forever.
func TestTimeoutWaitDoesNotLeakTimerGoroutine(t *testing.T) {

	wg := &sync.WaitGroup{}

	before := goroutineCount()

	// An hour long timeout: any goroutine still waiting on it cannot have exited.
	if newTimeout(time.Hour, nil).wait(wg) {
		t.Fatal("wait reported a timeout for an empty wait group")
	}

	if after := goroutineCount(); after > before {
		t.Fatalf("goroutines leaked: before=%d after=%d", before, after)
	}
}
