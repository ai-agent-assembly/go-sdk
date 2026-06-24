//go:build cgo && aa_ffi_go

package ffi

import "testing"

// Exercises the real native binding (links native/aa-ffi-go → aa-sdk-client).
// spawn_ipc_thread returns a handle even when the socket is absent (the
// background thread exits cleanly), so connect → disconnect round-trips without
// a live runtime and proves the FFI boundary links and marshals correctly.
func TestCgoBridgeConnectDisconnect(t *testing.T) {
	if !NativeBindingEnabled() {
		t.Skip("native aa_ffi_go binding not enabled")
	}

	client := NewDefaultClient()

	if err := client.Connect("/tmp/aa-ffi-go-cgo-roundtrip.sock", "", ""); err != nil {
		t.Fatalf("connect over native binding failed: %v", err)
	}

	// The send crosses the FFI boundary; without a live runtime the background
	// thread may have exited, so an enqueue error is acceptable — the point is
	// the call marshals safely and never panics across the boundary.
	if err := client.SendEvent("tool_call", `{"event":"x"}`); err != nil {
		t.Logf("send_event without a live runtime returned (acceptable): %v", err)
	}

	if err := client.Disconnect(); err != nil {
		t.Fatalf("disconnect over native binding failed: %v", err)
	}
}
