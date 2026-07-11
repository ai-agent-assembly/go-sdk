package assembly

import (
	"context"
	"errors"
	"testing"

	"github.com/ai-agent-assembly/go-sdk/internal/ffi"
)

func assertInitConnectorSuccess(t *testing.T) {
	t.Helper()
	originalConnector := sidecarConnector
	t.Cleanup(func() {
		sidecarConnector = originalConnector
	})

	sidecarConnector = func(ctx context.Context, _ string) (SidecarClient, error) {
		if ctx == nil {
			t.Fatal("expected context to be set")
		}
		return nil, nil
	}

	a, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil Assembly")
	}
}

func assertInitConnectorFailure(t *testing.T) {
	t.Helper()
	originalConnector := sidecarConnector
	t.Cleanup(func() {
		sidecarConnector = originalConnector
	})

	wantErr := errors.New("sidecar unavailable")
	sidecarConnector = func(context.Context, string) (SidecarClient, error) {
		return nil, wantErr
	}

	a, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if a != nil {
		t.Fatal("expected nil Assembly on error")
	}
}

func assertInitMissingGatewaySurfacesConfigError(t *testing.T) {
	t.Helper()
	// AAASM-1849 / E17 S-G: empty gateway URL is no longer an
	// immediate ErrInvalidGateway — the resolver tries env → config
	// → local default → auto-start. With aasm absent from PATH the
	// chain ends in *ConfigurationError.
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

func assertInitFallbackFFIFailsClosed(t *testing.T) {
	t.Helper()
	if ffi.NativeBindingEnabled() {
		t.Skip("native aa_ffi_go binding build does not use fallback transport")
	}

	originalConnector := sidecarConnector
	t.Cleanup(func() {
		sidecarConnector = originalConnector
	})

	// Without the native binding the fallback ffi connect fails closed
	// (ErrRuntimeUnavailable), so boot must fall through to the real
	// sidecar connector rather than silently "succeeding".
	connectorCalled := false
	sidecarConnector = func(context.Context, string) (SidecarClient, error) {
		connectorCalled = true
		return stubSidecarClient{}, nil
	}

	a, err := Init(context.Background(), validTestOptions()...)
	if err != nil {
		t.Fatalf("expected sidecar fallthrough to succeed, got %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil Assembly")
	}
	if !connectorCalled {
		t.Fatal("fallback ffi must fail closed, so sidecarConnector should be reached")
	}
}

func assertInitWithSidecarAddressReachesRegisterBranch(t *testing.T) {
	t.Helper()
	// AAASM-4469 G-A: the exported WithSidecarAddress must populate the field that
	// gates boot's real registration branch. Under the pure-Go fallback the native
	// ffi Connect fails closed, so boot falls through to sidecarConnector carrying
	// the configured address — observing that address proves an external caller
	// (using only exported options) now reaches the register path, which was
	// previously unreachable because the only setter was unexported.
	originalConnector := sidecarConnector
	t.Cleanup(func() {
		sidecarConnector = originalConnector
	})

	var gotAddress string
	sidecarConnector = func(_ context.Context, address string) (SidecarClient, error) {
		gotAddress = address
		return stubSidecarClient{}, nil
	}

	a, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		WithSidecarAddress("127.0.0.1:50051"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil Assembly")
	}
	if gotAddress != "127.0.0.1:50051" {
		t.Fatalf("boot did not carry the configured sidecar address: got %q, want 127.0.0.1:50051", gotAddress)
	}
}

func TestInit(t *testing.T) {
	t.Run("connector success", assertInitConnectorSuccess)
	t.Run("connector failure", assertInitConnectorFailure)
	t.Run("missing gateway and absent aasm surfaces ConfigurationError", assertInitMissingGatewaySurfacesConfigError)
	t.Run("fallback ffi fails closed and falls through to sidecar connector", assertInitFallbackFFIFailsClosed)
	t.Run("WithSidecarAddress reaches register branch via exported option", assertInitWithSidecarAddressReachesRegisterBranch)
}
