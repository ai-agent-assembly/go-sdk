package ffi

import "unsafe"

type capturingBinding struct {
	handle *byte
	Events []string
}

func (b *capturingBinding) connect(_ string) (unsafe.Pointer, int32) {
	b.handle = new(byte)
	return unsafe.Pointer(b.handle), statusOK
}

func (b *capturingBinding) sendEvent(_ unsafe.Pointer, eventJSON string) int32 {
	b.Events = append(b.Events, eventJSON)
	return statusOK
}

func (b *capturingBinding) queryPolicy(_ unsafe.Pointer, _ string) (string, int32) {
	return `{"allow":true}`, statusOK
}

func (b *capturingBinding) disconnect(_ unsafe.Pointer) int32 {
	return statusOK
}

// NewCapturingClient returns an FFI client backed by an in-memory binding that
// records every SendEvent payload. The second return value points to the
// captured slice — callers inspect it to assert on emitted events.
func NewCapturingClient() (*Client, *[]string) {
	b := &capturingBinding{}
	return NewClient(b), &b.Events
}
