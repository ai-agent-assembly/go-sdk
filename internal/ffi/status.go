package ffi

import (
	"errors"
	"fmt"
)

// Status codes 0–9 mirror the aa-ffi-go C ABI (AaStatus) returned by the native
// cgo bridge. statusRuntimeUnavailable is a Go-only sentinel used by the
// fail-closed fallback binding (no native transport compiled in). Codes 8–9 are
// the fail-closed registration outcomes surfaced only by aa_register — unlike a
// policy query, registration never fails open (see aa_ffi_go.h).
const (
	statusOK                 int32 = 0
	statusNullPointer        int32 = 1
	statusInvalidUTF8        int32 = 2
	statusNotConnected       int32 = 3
	statusMutexPoison        int32 = 4
	statusIPCError           int32 = 5
	statusChannelClosed      int32 = 6
	statusPanic              int32 = 7
	statusGatewayUnreachable int32 = 8
	statusRegisterFailed     int32 = 9
	statusRuntimeUnavailable int32 = 100
)

// errWrapFormat is the shared format string for wrapping a sentinel error with
// the originating FFI operation name. Hoisted to a const so the literal is not
// duplicated across every statusToError branch (go:S1192).
const errWrapFormat = "%s: %w"

var (
	// ErrNullPointer reports an FFI null-pointer guard failure.
	ErrNullPointer = errors.New("ffi null pointer")
	// ErrInvalidUTF8 reports invalid UTF-8 payload crossing the FFI boundary.
	ErrInvalidUTF8 = errors.New("ffi invalid utf-8")
	// ErrNotConnected reports calls attempted before connect.
	ErrNotConnected = errors.New("ffi client not connected")
	// ErrMutexPoison reports state lock corruption inside the FFI runtime.
	ErrMutexPoison = errors.New("ffi mutex poisoned")
	// ErrIPC reports the native IPC thread could not be started.
	ErrIPC = errors.New("ffi ipc thread error")
	// ErrChannelClosed reports the event could not be enqueued (channel closed).
	ErrChannelClosed = errors.New("ffi channel closed")
	// ErrPanic reports a panic was caught at the native FFI boundary.
	ErrPanic = errors.New("ffi panic at boundary")
	// ErrGatewayUnreachable reports that aa_register could not reach the gateway
	// gRPC endpoint. Registration is fail-closed at the native boundary; the SDK
	// layer treats it as advisory and proceeds unregistered (the runtime / proxy
	// / eBPF layers remain authoritative).
	ErrGatewayUnreachable = errors.New("ffi gateway unreachable")
	// ErrRegisterFailed reports the gateway rejected the Register call (e.g. an
	// invalid did:key). Like ErrGatewayUnreachable it is advisory at the SDK
	// layer.
	ErrRegisterFailed = errors.New("ffi register rejected by gateway")
	// ErrRuntimeUnavailable reports that no native enforcement transport is
	// available. The SDK fails closed rather than silently allowing traffic;
	// build with `-tags aa_ffi_go` (CGO_ENABLED=1) to enable the native binding.
	ErrRuntimeUnavailable = errors.New("ffi runtime unavailable (build with -tags aa_ffi_go for native enforcement)")
)

func statusToError(status int32, operation string) error {
	switch status {
	case statusOK:
		return nil
	case statusNullPointer:
		return fmt.Errorf(errWrapFormat, operation, ErrNullPointer)
	case statusInvalidUTF8:
		return fmt.Errorf(errWrapFormat, operation, ErrInvalidUTF8)
	case statusNotConnected:
		return fmt.Errorf(errWrapFormat, operation, ErrNotConnected)
	case statusMutexPoison:
		return fmt.Errorf(errWrapFormat, operation, ErrMutexPoison)
	case statusIPCError:
		return fmt.Errorf(errWrapFormat, operation, ErrIPC)
	case statusChannelClosed:
		return fmt.Errorf(errWrapFormat, operation, ErrChannelClosed)
	case statusPanic:
		return fmt.Errorf(errWrapFormat, operation, ErrPanic)
	case statusGatewayUnreachable:
		return fmt.Errorf(errWrapFormat, operation, ErrGatewayUnreachable)
	case statusRegisterFailed:
		return fmt.Errorf(errWrapFormat, operation, ErrRegisterFailed)
	case statusRuntimeUnavailable:
		return fmt.Errorf(errWrapFormat, operation, ErrRuntimeUnavailable)
	default:
		return fmt.Errorf("%s: ffi status %d", operation, status)
	}
}
