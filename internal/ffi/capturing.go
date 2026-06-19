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
	// registerStatus is the status code register returns; statusOK by default.
	// Set it to a failure code to exercise the advisory register-failure path.
	registerStatus int32
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
// advisory boot-path register call succeeds in tests. When registerStatus is set
// to a failure code it records the attempt but returns that status with no policy
// id, exercising the advisory register-failure path.
func (b *capturingBinding) register(_ unsafe.Pointer, agentID, name, framework, gatewayEndpoint string) (string, int32) {
	b.Registrations = append(b.Registrations, Registration{
		AgentID:         agentID,
		Name:            name,
		Framework:       framework,
		GatewayEndpoint: gatewayEndpoint,
	})
	if b.registerStatus != statusOK {
		return "", b.registerStatus
	}
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

// NewCapturingClientFailingRegister returns a capturing client whose register
// fails with GATEWAY_UNREACHABLE so the boot path's advisory register-failure
// branch (log + proceed unregistered) can be exercised. The captured
// registrations and events are still exposed for assertions.
func NewCapturingClientFailingRegister() (*Client, *[]string, *[]Registration) {
	b := &capturingBinding{registerStatus: statusGatewayUnreachable}
	return NewClient(b), &b.Events, &b.Registrations
}
