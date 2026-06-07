//go:build !cgo || !aa_ffi_go

package ffi

import "unsafe"

type fallbackUDSHandle struct {
	endpoint  string
	connected bool
	events    uint64
}

type fallbackUDSBridge struct{}

func (fallbackUDSBridge) connect(endpoint string) (unsafe.Pointer, int32) {
	handle := &fallbackUDSHandle{
		endpoint:  endpoint,
		connected: true,
	}

	return unsafe.Pointer(handle), statusOK
}

func (fallbackUDSBridge) sendEvent(handle unsafe.Pointer, _, _ string) int32 {
	client := (*fallbackUDSHandle)(handle)
	if client == nil {
		return statusNullPointer
	}
	if !client.connected {
		return statusNotConnected
	}

	client.events++
	return statusOK
}

func (fallbackUDSBridge) disconnect(handle unsafe.Pointer) int32 {
	client := (*fallbackUDSHandle)(handle)
	if client == nil {
		return statusNullPointer
	}
	if !client.connected {
		return statusNotConnected
	}

	client.connected = false
	return statusOK
}
