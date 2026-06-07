//go:build !cgo

package ffi

import (
	"errors"
	"testing"
)

func TestDefaultBindingUsesFallbackWhenCGODisabled(t *testing.T) {
	t.Parallel()

	if _, ok := defaultBinding().(fallbackUDSBridge); !ok {
		t.Fatalf("expected fallbackUDSBridge, got %T", defaultBinding())
	}
	if NativeBindingEnabled() {
		t.Fatal("NativeBindingEnabled should be false without -tags aa_ffi_go")
	}
}

// Without the native binding the fallback fails closed: there is no runtime to
// enforce, so Connect reports ErrRuntimeUnavailable rather than a silent allow.
func TestFallbackFailsClosedWithoutCGO(t *testing.T) {
	t.Parallel()

	client := NewDefaultClient()

	err := client.Connect("unix:///tmp/aa.sock")
	if !errors.Is(err, ErrRuntimeUnavailable) {
		t.Fatalf("expected ErrRuntimeUnavailable from fail-closed fallback, got %v", err)
	}

	// The handle was never set, so a subsequent send also fails (not connected).
	if err := client.SendEvent("tool_call", `{"event":"x"}`); err == nil {
		t.Fatal("expected SendEvent to fail on a fail-closed fallback")
	}
}

// The fallback binding itself reports unavailable on every operation.
func TestFallbackBindingReportsUnavailable(t *testing.T) {
	t.Parallel()

	b := fallbackUDSBridge{}
	if _, status := b.connect("x"); status != statusRuntimeUnavailable {
		t.Fatalf("connect status = %d, want statusRuntimeUnavailable", status)
	}
	if status := b.sendEvent(nil, "t", "d"); status != statusRuntimeUnavailable {
		t.Fatalf("sendEvent status = %d, want statusRuntimeUnavailable", status)
	}
	if status := b.disconnect(nil); status != statusRuntimeUnavailable {
		t.Fatalf("disconnect status = %d, want statusRuntimeUnavailable", status)
	}
}
