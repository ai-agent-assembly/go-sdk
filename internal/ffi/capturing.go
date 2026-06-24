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
	TeamID          string
	ParentAgentID   string
}

type capturingBinding struct {
	handle        *byte
	Events        []string
	Registrations []Registration
	// ConnectAgentID / ConnectSDKVersion record the last connect arguments so
	// tests can assert the Go-module version was forwarded into the handshake
	// (AAASM-3683).
	ConnectAgentID    string
	ConnectSDKVersion string
	// registerStatus is the status code register returns; statusOK by default.
	// Set it to a failure code to exercise the advisory register-failure path.
	registerStatus int32
}

func (b *capturingBinding) connect(_, agentID, sdkVersion string) (unsafe.Pointer, int32) {
	b.ConnectAgentID = agentID
	b.ConnectSDKVersion = sdkVersion
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
func (b *capturingBinding) register(_ unsafe.Pointer, agentID, name, framework, gatewayEndpoint, teamID, parentAgentID string) (string, int32) {
	b.Registrations = append(b.Registrations, Registration{
		AgentID:         agentID,
		Name:            name,
		Framework:       framework,
		GatewayEndpoint: gatewayEndpoint,
		TeamID:          teamID,
		ParentAgentID:   parentAgentID,
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

// NewCapturingClientWithRegisterStatus is like NewCapturingClientFailingRegister
// but lets the caller pick the failure status code so the advisory boot path can
// be exercised for both REGISTER_FAILED and GATEWAY_UNREACHABLE (AAASM-3404):
// aa_register is fail-closed at the native boundary, but both outcomes are
// advisory at the SDK layer. Pass statusOK to get a succeeding register.
func NewCapturingClientWithRegisterStatus(status int32) (*Client, *[]string, *[]Registration) {
	b := &capturingBinding{registerStatus: status}
	return NewClient(b), &b.Events, &b.Registrations
}

// RegisterFailedStatus is the native status code aa_register returns when the
// gateway rejects the registration (e.g. an invalid did:key). Exported so the
// assembly boot tests can drive the advisory REGISTER_FAILED branch without
// reaching into the unexported status constants.
const RegisterFailedStatus = statusRegisterFailed

// GatewayUnreachableStatus is the native status code aa_register returns when it
// cannot reach the gateway gRPC endpoint. Exported for the same reason as
// RegisterFailedStatus.
const GatewayUnreachableStatus = statusGatewayUnreachable

// denyingRegisteringBinding records native registrations like capturingBinding
// and additionally answers policy queries (policyQuerier) with a fixed decision.
// It lets a boot test drive the full Init -> aa_register -> WrapTools ->
// aa_query_policy path so a runtime DENY blocks a wrapped tool end to end.
type denyingRegisteringBinding struct {
	capturingBinding
	queryDecision int32
	queryReason   string
}

func (b *denyingRegisteringBinding) queryPolicy(_ unsafe.Pointer, _, _, _, _ string) (int32, string, int32) {
	return b.queryDecision, b.queryReason, statusOK
}

// NewCapturingClientDenying returns a capturing client whose register succeeds
// and whose policy queries return DENY with the given reason, so a boot test can
// assert that a reachable runtime's DENY blocks a wrapped tool after Init. The
// captured registrations are exposed so the test can also confirm aa_register ran.
func NewCapturingClientDenying(reason string) (*Client, *[]Registration) {
	b := &denyingRegisteringBinding{queryDecision: DecisionDeny, queryReason: reason}
	return NewClient(b), &b.Registrations
}

// NewCapturingClientAllowing returns a capturing client whose register succeeds
// and whose policy queries return ALLOW, so a boot test can assert that a
// reachable runtime lets a permitted wrapped tool run after Init.
func NewCapturingClientAllowing() (*Client, *[]Registration) {
	b := &denyingRegisteringBinding{queryDecision: DecisionAllow}
	return NewClient(b), &b.Registrations
}
