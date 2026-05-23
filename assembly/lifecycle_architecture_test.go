package assembly

import (
	"context"
	"testing"
)

func TestAssemblyLifecycle(t *testing.T) {
	originalConnector := sidecarConnector
	t.Cleanup(func() {
		sidecarConnector = originalConnector
	})

	connectorCalled := false
	sidecarConnector = func(ctx context.Context, address string) (SidecarClient, error) {
		connectorCalled = true
		if ctx == nil {
			t.Fatal("expected context to be set")
		}
		if address != "" {
			t.Fatalf("expected default sidecar address to be empty, got %q", address)
		}
		return stubSidecarClient{}, nil
	}

	a, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
	)
	if err != nil {
		t.Fatalf("expected no init error, got %v", err)
	}
	if !connectorCalled {
		t.Fatal("expected sidecar connector to be called")
	}
	if a.sidecar == nil {
		t.Fatal("expected sidecar to be set after init")
	}

	if err := a.Close(); err != nil {
		t.Fatalf("expected no close error, got %v", err)
	}
	if a.sidecar != nil {
		t.Fatal("expected sidecar to be released after close")
	}
}

func TestAssemblyInitValidation(t *testing.T) {
	// Cannot run in parallel: swaps gatewayResolverSeams. Forces the
	// resolver's auto-start path to fail by stubbing findAasmOnPath to
	// "" — the post-AAASM-1849 equivalent of the old "no gateway URL"
	// validation error is *ConfigurationError from the resolver chain.
	withSeams(t, func() string { return "" }, func(string) error { return nil })
	t.Setenv(envGatewayURL, "")
	a, err := Init(context.Background(), WithAPIKey("test-key"))
	if err == nil {
		t.Fatalf("expected ConfigurationError, got nil")
	}
	var ce *ConfigurationError
	if !errorsAs(err, &ce) {
		t.Fatalf("expected *ConfigurationError, got %T: %v", err, err)
	}
	if a != nil {
		t.Fatal("expected nil Assembly on validation error")
	}
}

type stubSidecarClient struct{}

func (stubSidecarClient) Ping(context.Context) error {
	return nil
}
