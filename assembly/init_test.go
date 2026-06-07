package assembly

import (
	"context"
	"errors"
	"testing"

	"github.com/ai-agent-assembly/go-sdk/internal/ffi"
)

func TestInit(t *testing.T) {
	t.Run("connector success", func(t *testing.T) {
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
	})

	t.Run("connector failure", func(t *testing.T) {
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
	})

	t.Run("missing gateway and absent aasm surfaces ConfigurationError", func(t *testing.T) {
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
	})

	t.Run("fallback ffi fails closed and falls through to sidecar connector", func(t *testing.T) {
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
	})
}
