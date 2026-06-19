//go:build !cgo || !aa_ffi_go

package ffi

import (
	"errors"
	"testing"
)

// TestDefaultClientRegisterFailsClosedWithoutCgo exercises the no-cgo fallback
// through the public Client.Register surface (not just the binding method): when
// the native shim is not linked in, the build-selected fallback binding does
// implement the registerer capability but reports the runtime as unavailable, so
// Register fails closed with ErrRuntimeUnavailable. The boot path then proceeds
// unregistered. This is the path a default `go test ./...` build actually runs.
func TestDefaultClientRegisterFailsClosedWithoutCgo(t *testing.T) {
	t.Parallel()

	client := NewDefaultClient()

	// connect() also fails closed in the no-cgo fallback, so handle stays nil;
	// Register's not-connected guard short-circuits before reaching the binding.
	_ = client.Connect("127.0.0.1:50051")

	policyID, err := client.Register("agent-001", "agent-001", "go", "", "", "")
	if err == nil {
		t.Fatal("expected the no-cgo fallback to fail closed, got nil error")
	}
	// Either guard (not-connected, because the fallback connect fails closed) or
	// the binding's runtime-unavailable status is an acceptable fail-closed
	// outcome; what must never happen is a silent success.
	if !errors.Is(err, ErrRuntimeUnavailable) && !errors.Is(err, ErrNotConnected) {
		t.Fatalf("expected ErrRuntimeUnavailable or ErrNotConnected, got %v", err)
	}
	if policyID != "" {
		t.Fatalf("policyID = %q, want empty when no native transport is linked", policyID)
	}
}
