//go:build !cgo || !aa_ffi_go

package ffi

import (
	"errors"
	"testing"
)

// TestFallbackBindingRegisterUnavailable verifies the no-cgo fallback reports the
// runtime as unavailable so the SDK boot path proceeds unregistered (registration
// is advisory at the SDK layer).
func TestFallbackBindingRegisterUnavailable(t *testing.T) {
	t.Parallel()

	// The fallback connect fails closed, so drive the binding directly to assert
	// the register capability's unavailable status independent of connect.
	_, status := fallbackUDSBridge{}.register(nil, "agent-001", "agent-001", "go", "", "", "")
	if err := statusToError(status, "register"); !errors.Is(err, ErrRuntimeUnavailable) {
		t.Fatalf("expected ErrRuntimeUnavailable, got %v", err)
	}
}
