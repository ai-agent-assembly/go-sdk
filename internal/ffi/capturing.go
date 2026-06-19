package ffi

import "unsafe"

// Registration records the arguments of a single capturingBinding.register call
// so boot tests can assert the agent was registered via the native aa_register
// path (AAASM-3404) rather than only via a SendEvent("register", ...) audit
// event.
type Registration struct {
	AgentID         string
	Name            string
	Framework       string
	GatewayEndpoint string
}

type capturingBinding struct {
	handle        *byte
	Events        []string
	Registrations []Registration
}

func (b *capturingBinding) connect(_ string) (unsafe.Pointer, int32) {
	b.handle = new(byte)
	return unsafe.Pointer(b.handle), statusOK
}

func (b *capturingBinding) sendEvent(_ unsafe.Pointer, _, details string) int32 {
	b.Events = append(b.Events, details)
	return statusOK
}

func (b *capturingBinding) disconnect(_ unsafe.Pointer) int32 {
	return statusOK
}

// register records the registration and returns a deterministic policy id so the
// advisory boot-path register call succeeds in tests.
func (b *capturingBinding) register(_ unsafe.Pointer, agentID, name, framework, gatewayEndpoint string) (string, int32) {
	b.Registrations = append(b.Registrations, Registration{
		AgentID:         agentID,
		Name:            name,
		Framework:       framework,
		GatewayEndpoint: gatewayEndpoint,
	})
	return "policy-" + agentID, statusOK
}

// NewCapturingClient returns an FFI client backed by an in-memory binding that
// records every SendEvent's details payload. The second return value points to
// the captured slice — callers inspect it to assert on emitted events.
func NewCapturingClient() (*Client, *[]string) {
	b := &capturingBinding{}
	return NewClient(b), &b.Events
}

// NewCapturingClientWithRegistrations is like NewCapturingClient but also exposes
// the captured native registrations so boot tests can assert aa_register was
// called on Init (AAASM-3404).
func NewCapturingClientWithRegistrations() (*Client, *[]string, *[]Registration) {
	b := &capturingBinding{}
	return NewClient(b), &b.Events, &b.Registrations
}
