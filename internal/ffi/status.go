package ffi

import (
	"errors"
	"fmt"
)

// Status codes 0–7 mirror the aa-ffi-go C ABI (AaStatus) returned by the native
// cgo bridge. statusRuntimeUnavailable is a Go-only sentinel used by the
// fail-closed fallback binding (no native transport compiled in).
const (
	statusOK                 int32 = 0
	statusNullPointer        int32 = 1
	statusInvalidUTF8        int32 = 2
	statusNotConnected       int32 = 3
	statusMutexPoison        int32 = 4
	statusIPCError           int32 = 5
	statusChannelClosed      int32 = 6
	statusPanic              int32 = 7
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
	case statusRuntimeUnavailable:
		return fmt.Errorf(errWrapFormat, operation, ErrRuntimeUnavailable)
	default:
		return fmt.Errorf("%s: ffi status %d", operation, status)
	}
}
