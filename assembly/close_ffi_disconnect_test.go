package assembly

import (
	"context"
	"testing"

	"github.com/ai-agent-assembly/go-sdk/internal/ffi"
)

// TestCloseDisconnectsNativeFFISession is the AAASM-4832 teardown contract: when
// boot opened a native FFI session, Close must Disconnect it so the runtime
// session, its IPC reader thread, and the registered credential are released
// rather than leaked for the process lifetime.
func TestCloseDisconnectsNativeFFISession(t *testing.T) {
	capClient, disconnects := ffi.NewCapturingClientRecordingDisconnect()
	withCapturingFFIClient(t, capClient)

	a, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		withSidecarAddress("127.0.0.1:50051"),
		WithSelfAgentID("agent-007"),
	)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if *disconnects != 0 {
		t.Fatalf("Disconnect called %d times before Close, want 0", *disconnects)
	}

	if err := a.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if *disconnects != 1 {
		t.Fatalf("Disconnect called %d times after Close, want 1 (Close must tear down the native FFI session)", *disconnects)
	}
}

// TestClosePureGoFallbackDoesNotDisconnect guards the fail-closed pure-Go path:
// a runtime that never opened an FFI session (the default CGO_ENABLED=0 build
// falls through to the sidecar connector) must not call Disconnect on Close —
// Disconnect on a never-connected client returns a not-connected error, so an
// unguarded Close would surface a spurious error. Close stays a no-op here.
func TestClosePureGoFallbackDoesNotDisconnect(t *testing.T) {
	capClient, disconnects := ffi.NewCapturingClientRecordingDisconnect()
	// ffiConnected is left false: this mirrors a boot that fell through to the
	// sidecar connector without ever connecting the FFI session.
	a := &Assembly{ffiClient: capClient}

	if err := a.Close(); err != nil {
		t.Fatalf("Close on a never-connected runtime must be a no-op, got %v", err)
	}
	if *disconnects != 0 {
		t.Fatalf("Disconnect called %d times on a never-connected client, want 0", *disconnects)
	}
}
