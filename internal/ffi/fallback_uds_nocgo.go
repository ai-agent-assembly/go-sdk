//go:build !cgo || !aa_ffi_go

package ffi

import "unsafe"

// fallbackUDSBridge is the binding selected when the native cgo shim is not
// compiled in (no `-tags aa_ffi_go`, or CGO disabled).
//
// It performs NO transport. Agent Assembly is a security/governance product, so
// rather than silently allowing traffic when there is no runtime to enforce it,
// the fallback **fails closed**: every operation reports the runtime as
// unavailable (statusRuntimeUnavailable → ErrRuntimeUnavailable). Build with
// `-tags aa_ffi_go` and CGO_ENABLED=1 to link the native aa-ffi-go binding and
// route events through the authoritative runtime.
type fallbackUDSBridge struct{}

func (fallbackUDSBridge) connect(string) (unsafe.Pointer, int32) {
	return nil, statusRuntimeUnavailable
}

func (fallbackUDSBridge) sendEvent(unsafe.Pointer, string, string) int32 {
	return statusRuntimeUnavailable
}

func (fallbackUDSBridge) disconnect(unsafe.Pointer) int32 {
	return statusRuntimeUnavailable
}
