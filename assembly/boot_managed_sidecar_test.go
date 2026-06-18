//go:build !aa_ffi_go

// Gated to the pure-Go fallback build: this test relies on the FFI client failing
// closed (no native binding) so boot falls through to the stubbed sidecarConnector.
// Under -tags aa_ffi_go the native client instead connects to the stub listener and
// boot routes registration through SendEvent, so this fallback scenario never runs.
// Line coverage is measured in the non-native build, so no credit is lost.

package assembly

import (
	"context"
	"net"
	"testing"
)

// TestInit_ManagedSidecarLifecycle drives boot's managed-sidecar branch:
// WithSidecarBinary launches a subprocess, boot then waits for the sidecar
// address to become healthy before wiring the connection. A standalone TCP
// listener stands in for the sidecar's health port so Healthy succeeds
// without a real aasm binary.
func TestInit_ManagedSidecarLifecycle(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	bin := osTrueBinary(t)

	// The fallback FFI client fails closed (no native binding), so boot falls
	// through to sidecarConnector — stub it to a sentinel client so boot
	// completes its managed-sidecar + connector path without a live runtime.
	originalConnector := sidecarConnector
	t.Cleanup(func() { sidecarConnector = originalConnector })
	connected := false
	sidecarConnector = func(context.Context, string) (SidecarClient, error) {
		connected = true
		return stubSidecarClient{}, nil
	}

	a, err := Init(context.Background(),
		WithGatewayURL("https://gw.example.com"),
		WithAPIKey("k"),
		WithSidecarBinary(bin),
		withSidecarAddress(ln.Addr().String()),
	)
	if err != nil {
		t.Fatalf("Init with managed sidecar: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })

	if a.managedSidecar == nil {
		t.Fatal("expected boot to record a managed sidecar")
	}
	if !connected {
		t.Fatal("expected boot to reach the sidecar connector after the managed sidecar became healthy")
	}
}
