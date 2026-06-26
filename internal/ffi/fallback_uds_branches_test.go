//go:build !cgo || !aa_ffi_go

// These tests exercise the pure-Go UDS fallback binding in the default,
// non-native build (no -tags aa_ffi_go). The existing fallback assertions in
// fallback_nocgo_test.go are gated `!cgo` only, so they are skipped whenever
// CGO is enabled — which is the default coverage build (CGO on, aa_ffi_go
// off). Matching the source file's `!cgo || !aa_ffi_go` tag here ensures the
// fallback binding methods are covered in that build too.

package ffi

import "testing"

// TestFallbackBindingsFailClosedNonNative asserts every fallback binding
// method reports the runtime as unavailable so the SDK fails closed rather
// than silently allowing ungoverned traffic.
func TestFallbackBindingsFailClosedNonNative(t *testing.T) {
	t.Parallel()

	b := fallbackUDSBridge{}

	if handle, status := b.connect("unix:///tmp/aa.sock", "agent", "1.2.3"); handle != nil || status != statusRuntimeUnavailable {
		t.Fatalf("connect = (%v, %d), want (nil, statusRuntimeUnavailable)", handle, status)
	}
	if status := b.sendEvent(nil, "tool_call", `{"k":"v"}`); status != statusRuntimeUnavailable {
		t.Fatalf("sendEvent status = %d, want statusRuntimeUnavailable", status)
	}
	if status := b.disconnect(nil); status != statusRuntimeUnavailable {
		t.Fatalf("disconnect status = %d, want statusRuntimeUnavailable", status)
	}
}

// TestFallbackQueryPolicyReportsUnavailable covers the fallback queryPolicy
// surface: there is no transport, so it reports the runtime as unavailable
// (the caller then applies its configured fail-open / fail-closed policy).
func TestFallbackQueryPolicyReportsUnavailable(t *testing.T) {
	t.Parallel()

	decision, reason, status := fallbackUDSBridge{}.queryPolicy(nil, "agent", "tool", "args", "trace")
	if status != statusRuntimeUnavailable {
		t.Fatalf("queryPolicy status = %d, want statusRuntimeUnavailable", status)
	}
	if decision != DecisionAllow {
		t.Fatalf("queryPolicy decision = %d, want DecisionAllow placeholder", decision)
	}
	if reason != "" {
		t.Fatalf("queryPolicy reason = %q, want empty", reason)
	}
}

// TestNativeBindingDisabledInNonNativeBuild covers the fallback
// NativeBindingEnabled stub, which must report false when the native cgo shim
// is not compiled in.
func TestNativeBindingDisabledInNonNativeBuild(t *testing.T) {
	t.Parallel()

	if NativeBindingEnabled() {
		t.Fatal("expected NativeBindingEnabled to be false in the non-native build")
	}
}
