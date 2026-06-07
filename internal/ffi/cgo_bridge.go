//go:build cgo && aa_ffi_go

package ffi

/*
#cgo CFLAGS: -I${SRCDIR}/../../native/aa-ffi-go/include
#cgo LDFLAGS: -L${SRCDIR}/../../native/aa-ffi-go/target/debug -laa_ffi_go -Wl,-rpath,${SRCDIR}/../../native/aa-ffi-go/target/debug
#include <stdlib.h>
#include "aa_ffi_go.h"
*/
import "C"

import "unsafe"

// cgoBridge links the vendored native/aa-ffi-go library (a thin C-ABI over the
// SHA-pinned aa-sdk-client) and routes events through the authoritative runtime.
// Compiled only with `-tags aa_ffi_go` and CGO_ENABLED=1; the Makefile builds the
// crate first so the header (CFLAGS) and library (LDFLAGS) resolve.
type cgoBridge struct{}

func (cgoBridge) connect(endpoint string) (unsafe.Pointer, int32) {
	cEndpoint := C.CString(endpoint)
	defer C.free(unsafe.Pointer(cEndpoint))

	var handle *C.aa_client_handle
	status := C.aa_connect(cEndpoint, &handle)
	return unsafe.Pointer(handle), int32(status)
}

func (cgoBridge) sendEvent(handle unsafe.Pointer, eventType, details string) int32 {
	cType := C.CString(eventType)
	defer C.free(unsafe.Pointer(cType))
	cDetails := C.CString(details)
	defer C.free(unsafe.Pointer(cDetails))

	status := C.aa_send_event((*C.aa_client_handle)(handle), cType, cDetails)
	return int32(status)
}

func (cgoBridge) disconnect(handle unsafe.Pointer) int32 {
	return int32(C.aa_disconnect((*C.aa_client_handle)(handle)))
}
