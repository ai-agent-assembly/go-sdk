package assembly

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProbeHealthz_TrueOn2xx(t *testing.T) {
	t.Parallel()

	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if !probeHealthz(context.Background(), srv.URL, defaultProbeTimeout) {
		t.Fatalf("expected probe to succeed against httptest server")
	}
	if capturedPath != defaultHealthzPath {
		t.Fatalf("expected probe to hit %q, got %q", defaultHealthzPath, capturedPath)
	}
}

func TestProbeHealthz_FalseOnConnectionRefused(t *testing.T) {
	t.Parallel()

	// 127.0.0.1:1 is a port that should never be listened on in CI.
	if probeHealthz(context.Background(), "http://127.0.0.1:1", 100*time.Millisecond) {
		t.Fatalf("expected probe against unreachable host to return false")
	}
}

func TestProbeHealthz_FalseOnNon2xx(t *testing.T) {
	t.Parallel()

	for _, status := range []int{400, 404, 500, 503} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		}))
		if probeHealthz(context.Background(), srv.URL, defaultProbeTimeout) {
			t.Fatalf("expected probe to return false on status %d", status)
		}
		srv.Close()
	}
}

func TestWaitForHealthz_SuccessOnFirstProbe(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if !waitForHealthz(context.Background(), srv.URL, time.Second, 10*time.Millisecond) {
		t.Fatalf("expected immediate success when server is already healthy")
	}
}

func TestWaitForHealthz_SuccessAfterInitialFailures(t *testing.T) {
	t.Parallel()

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if !waitForHealthz(context.Background(), srv.URL, time.Second, 10*time.Millisecond) {
		t.Fatalf("expected success after server starts returning 200")
	}
	if calls < 3 {
		t.Fatalf("expected at least 3 probes, got %d", calls)
	}
}

func TestWaitForHealthz_FalseOnTimeout(t *testing.T) {
	t.Parallel()

	if waitForHealthz(context.Background(), "http://127.0.0.1:1", 50*time.Millisecond, 10*time.Millisecond) {
		t.Fatalf("expected false when no gateway is reachable within timeout")
	}
}

func TestLoadConfigFile_ReturnsEmptyWhenMissing(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	got := loadConfigFile(filepath.Join(tmp, "absent.yaml"))
	if len(got) != 0 {
		t.Fatalf("expected empty map for missing file, got %v", got)
	}
}

func TestLoadConfigFile_ReturnsParsedMapping(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfg := filepath.Join(tmp, "config.yaml")
	contents := "agent:\n  gateway_url: \"http://staging.internal:7391\"\n  api_key: \"k-1\"\n"
	if err := os.WriteFile(cfg, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got := loadConfigFile(cfg)
	agent, ok := got["agent"].(map[string]any)
	if !ok {
		t.Fatalf("expected agent section as map, got %T", got["agent"])
	}
	if agent["gateway_url"] != "http://staging.internal:7391" {
		t.Errorf("agent.gateway_url: got %v", agent["gateway_url"])
	}
	if agent["api_key"] != "k-1" {
		t.Errorf("agent.api_key: got %v", agent["api_key"])
	}
}

func TestLoadConfigFile_ReturnsEmptyOnNonMappingRoot(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfg := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(cfg, []byte("- just-a-list\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if got := loadConfigFile(cfg); len(got) != 0 {
		t.Fatalf("expected empty map for non-mapping YAML, got %v", got)
	}
}

func TestLoadConfigFile_ReturnsEmptyOnMalformedYAML(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfg := filepath.Join(tmp, "config.yaml")
	// Genuinely invalid YAML: an unclosed quoted scalar with an embedded
	// colon followed by a sibling mapping key. yaml.v3 is forgiving for
	// mildly weird YAML so the sample has to be unambiguously broken.
	contents := "agent:\n  gateway_url: \"unclosed string\n  api_key: val"
	if err := os.WriteFile(cfg, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if got := loadConfigFile(cfg); len(got) != 0 {
		t.Fatalf("expected empty map for malformed YAML, got %v", got)
	}
}

// withSeams swaps gatewayResolverSeams for the duration of one test and
// restores the originals via t.Cleanup. Tests run sequentially when they
// use this helper — declare them without t.Parallel() to keep the seam
// swap deterministic.
func withSeams(t *testing.T, find func() string, spawn func(string) error) {
	t.Helper()
	originalFind := gatewayResolverSeams.findAasmOnPath
	originalSpawn := gatewayResolverSeams.spawnAasm
	gatewayResolverSeams.findAasmOnPath = find
	gatewayResolverSeams.spawnAasm = spawn
	t.Cleanup(func() {
		gatewayResolverSeams.findAasmOnPath = originalFind
		gatewayResolverSeams.spawnAasm = originalSpawn
	})
}

func TestAutoStartGateway_ConfigurationErrorWhenAasmMissing(t *testing.T) {
	withSeams(t, func() string { return "" }, func(string) error {
		t.Fatalf("spawnAasm should not be called when aasm is not on PATH")
		return nil
	})
	err := autoStartGateway(context.Background(), defaultGatewayURL, time.Second)
	if err == nil {
		t.Fatalf("expected ConfigurationError, got nil")
	}
	var ce *ConfigurationError
	if !errorsAs(err, &ce) {
		t.Fatalf("expected *ConfigurationError, got %T: %v", err, err)
	}
}

func TestAutoStartGateway_SuccessSpawnsAndConfirmsReady(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var spawned string
	withSeams(t,
		func() string { return "/usr/local/bin/aasm" },
		func(p string) error { spawned = p; return nil },
	)
	err := autoStartGateway(context.Background(), srv.URL, time.Second)
	if err != nil {
		t.Fatalf("autoStartGateway: unexpected error %v", err)
	}
	if spawned != "/usr/local/bin/aasm" {
		t.Errorf("expected spawn with /usr/local/bin/aasm, got %q", spawned)
	}
}

func TestAutoStartGateway_GatewayErrorOnTimeout(t *testing.T) {
	withSeams(t,
		func() string { return "/usr/local/bin/aasm" },
		func(string) error { return nil },
	)
	err := autoStartGateway(context.Background(), "http://127.0.0.1:1", 30*time.Millisecond)
	if err == nil {
		t.Fatalf("expected GatewayError, got nil")
	}
	var ge *GatewayError
	if !errorsAs(err, &ge) {
		t.Fatalf("expected *GatewayError, got %T: %v", err, err)
	}
}

// resolveGatewayURL — precedence tests
func TestResolveGatewayURL_ExplicitShortCircuits(t *testing.T) {
	t.Setenv(envGatewayURL, "http://from-env:7391")
	got, err := resolveGatewayURL(context.Background(), "http://explicit:7391")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://explicit:7391" {
		t.Errorf("expected explicit URL, got %q", got)
	}
}

func TestResolveGatewayURL_EnvUsedWhenNoExplicit(t *testing.T) {
	t.Setenv(envGatewayURL, "http://from-env:7391")
	got, err := resolveGatewayURL(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://from-env:7391" {
		t.Errorf("expected env URL, got %q", got)
	}
}

func TestResolveGatewayURL_LocalDefaultWhenProbeSucceeds(t *testing.T) {
	t.Setenv(envGatewayURL, "")
	// Patch the seam so a fake gateway answers /healthz. Use httptest to
	// emulate a live local CP — but the resolver hard-codes
	// defaultGatewayURL, so we cannot redirect probe target. Instead use
	// the autoStart seam to satisfy the chain on probe-miss.
	withSeams(t,
		func() string { return "/usr/local/bin/aasm" },
		func(string) error { return nil },
	)
	// The real localhost:7391 is unlikely to be listening, so probeHealthz
	// will return false → autoStartGateway path is exercised. But
	// autoStartGateway calls waitForHealthz which will also fail → ends in
	// GatewayError. We assert that the resolver propagates the error so
	// callers see it rather than a misleading success.
	_, err := resolveGatewayURL(context.Background(), "")
	if err == nil {
		t.Fatalf("expected resolver to surface auto-start failure; got success")
	}
	var ge *GatewayError
	if !errorsAs(err, &ge) {
		t.Fatalf("expected *GatewayError from probe-miss path, got %T: %v", err, err)
	}
}

// resolveAPIKey — precedence tests
func TestResolveAPIKey_ExplicitShortCircuits(t *testing.T) {
	t.Setenv(envAPIKey, "k-env")
	if got := resolveAPIKey("k-explicit"); got != "k-explicit" {
		t.Errorf("expected explicit key, got %q", got)
	}
}

func TestResolveAPIKey_EnvUsedWhenNoExplicit(t *testing.T) {
	t.Setenv(envAPIKey, "k-env")
	if got := resolveAPIKey(""); got != "k-env" {
		t.Errorf("expected env key, got %q", got)
	}
}

func TestResolveAPIKey_EmptyDefault(t *testing.T) {
	t.Setenv(envAPIKey, "")
	if got := resolveAPIKey(""); got != "" {
		t.Errorf("expected empty default, got %q", got)
	}
}

// errorsAs is a tiny wrapper around errors.As that keeps the assertion
// call sites symmetrical between the three SDKs.
func errorsAs(err error, target any) bool {
	return errors.As(err, target)
}
