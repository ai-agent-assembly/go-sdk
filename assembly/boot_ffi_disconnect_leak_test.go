package assembly

import (
	"context"
	"testing"

	"github.com/ai-agent-assembly/go-sdk/internal/ffi"
)

// TestBootDisconnectsFFIOnSendEventFailure is the AAASM-4843 regression: when boot
// has opened the native FFI session (Connect succeeded) but the register audit
// SendEvent then fails, boot must Disconnect the session before returning the
// error. Init discards the *Assembly on error, so Close — the only other
// Disconnect caller — never runs; without this teardown the native session, its
// IPC reader thread, and the registered credential leak for the process lifetime.
// This is the connect-succeeded-then-fail sibling of the AAASM-4789 managed-
// sidecar leak and the AAASM-4832 Close-disconnect contract.
func TestBootDisconnectsFFIOnSendEventFailure(t *testing.T) {
	capClient, disconnects := ffi.NewCapturingClientFailingSendEvent()
	withCapturingFFIClient(t, capClient)

	_, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		withSidecarAddress("127.0.0.1:50051"),
		WithSelfAgentID("agent-007"),
	)
	if err == nil {
		t.Fatal("Init must surface the SendEvent failure, got nil")
	}
	if *disconnects != 1 {
		t.Fatalf("Disconnect called %d times on the boot SendEvent-failure path, want 1 (boot must not leak the FFI session)", *disconnects)
	}
}
