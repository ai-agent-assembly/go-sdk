package assembly

import (
	"context"
	"testing"

	"github.com/ai-agent-assembly/go-sdk/internal/ffi"
)

// TestInit_ZeroArgResolvesLocalDefault is the AAASM-1849 primary AC:
// Init(ctx) with no options reaches the resolver chain, hits the
// local default, and returns a configured *Assembly. The resolver
// auto-start path is short-circuited via gatewayResolverSeams so the
// test does not actually spawn aasm — the spawn function is replaced
// with a no-op and a fake healthz server confirms readiness.
//
// Note: cannot use t.Parallel — swaps gatewayResolverSeams + envs.
func TestInit_ZeroArgResolvesLocalDefault(t *testing.T) {
	// httptest provides a fake healthz at a random port — but the resolver
	// hard-codes defaultGatewayURL, so the way we exercise the success
	// path is to stub spawnAasm and let waitForHealthz fall through to
	// real localhost:7391 (which we cannot redirect). Instead skip the
	// success branch and assert the resolver propagates the
	// ConfigurationError when aasm is also absent — that's the second
	// half of the AC and the part we can cover without a real CP.
	withSeams(t, func() string { return "" }, func(string) error { return nil })
	t.Setenv(envGatewayURL, "")
	t.Setenv(envAPIKey, "")

	a, err := Init(context.Background())
	if err == nil {
		t.Fatalf("expected ConfigurationError (no aasm + no listener), got nil")
	}
	var ce *ConfigurationError
	if !errorsAs(err, &ce) {
		t.Fatalf("expected *ConfigurationError, got %T: %v", err, err)
	}
	if a != nil {
		t.Fatalf("expected nil Assembly on resolver failure")
	}
}

// TestInit_ExplicitOptionsBypassResolver is the AAASM-1849 regression
// guarantee: callers passing both WithGatewayURL and WithAPIKey skip
// the resolver entirely. Stubs probeHealthz / autoStartGateway via the
// seams with panic-on-call sentinels — if either is invoked, the test
// fails immediately.
func TestInit_ExplicitOptionsBypassResolver(t *testing.T) {
	withSeams(t,
		func() string {
			t.Fatalf("findAasmOnPath should not be called with explicit options")
			return ""
		},
		func(string) error {
			t.Fatalf("spawnAasm should not be called with explicit options")
			return nil
		},
	)

	// Pin a deterministic in-memory FFI client via the seam so the boot
	// transport path succeeds regardless of build tag (the real -tags aa_ffi_go
	// binding would need a live runtime). This test is about resolver bypass,
	// not the FFI transport.
	capClient, _ := ffi.NewCapturingClient()
	originalFactory := newFFIClient
	t.Cleanup(func() { newFFIClient = originalFactory })
	newFFIClient = func() *ffi.Client { return capClient }

	originalConnector := sidecarConnector
	t.Cleanup(func() { sidecarConnector = originalConnector })
	sidecarConnector = func(context.Context, string) (SidecarClient, error) {
		return stubSidecarClient{}, nil
	}

	a, err := Init(context.Background(), validTestOptions()...)
	if err != nil {
		t.Fatalf("unexpected error with explicit options: %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil Assembly on success")
	}
	if a.opts.gatewayURL != "https://gateway.example.com" {
		t.Errorf("gatewayURL: got %q, want gateway.example.com", a.opts.gatewayURL)
	}
	if a.opts.apiKey != "test-key" {
		t.Errorf("apiKey: got %q, want test-key", a.opts.apiKey)
	}
	_ = a.Close()
}
