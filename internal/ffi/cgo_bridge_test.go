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

// TestCgoBridgeConnectForwardsAgentIDAndSDKVersion exercises the AAASM-3683
// marshalling branch in cgoBridge.connect: a non-empty agent id and SDK version
// must each be turned into a heap C string (and freed) and handed to aa_connect.
// Without a live runtime the background thread may exit, so the call need only
// marshal both non-NULL arguments across the FFI boundary safely and round-trip
// to disconnect — proving the new CString path links and frees correctly.
func TestCgoBridgeConnectForwardsAgentIDAndSDKVersion(t *testing.T) {
	if !NativeBindingEnabled() {
		t.Skip("native aa_ffi_go binding not enabled")
	}

	client := NewDefaultClient()

	if err := client.Connect("/tmp/aa-ffi-go-cgo-version.sock", "agent-cgo-1", "go-1.2.3"); err != nil {
		t.Fatalf("connect with agent id and SDK version over native binding failed: %v", err)
	}

	if err := client.Disconnect(); err != nil {
		t.Fatalf("disconnect over native binding failed: %v", err)
	}
}
