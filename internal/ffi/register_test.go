package ffi

import (
	"errors"
	"testing"
)

// TestClientRegisterThroughCapturingBinding verifies Register marshals its
// arguments to the binding and returns the gateway-assigned policy id on success.
func TestClientRegisterThroughCapturingBinding(t *testing.T) {
	t.Parallel()

	client, _, regs := NewCapturingClientWithRegistrations()
	if err := client.Connect("127.0.0.1:50051"); err != nil {
		t.Fatalf("connect: %v", err)
	}

	policyID, err := client.Register("agent-001", "agent-001", "go", "http://127.0.0.1:50051")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if policyID != "policy-agent-001" {
		t.Fatalf("policyID = %q, want policy-agent-001", policyID)
	}

	if len(*regs) != 1 {
		t.Fatalf("registrations = %d, want 1", len(*regs))
	}
	got := (*regs)[0]
	if got.AgentID != "agent-001" || got.Name != "agent-001" || got.Framework != "go" ||
		got.GatewayEndpoint != "http://127.0.0.1:50051" {
		t.Fatalf("unexpected registration %+v", got)
	}
}

// TestClientRegisterNotConnected fails before connect with ErrNotConnected.
func TestClientRegisterNotConnected(t *testing.T) {
	t.Parallel()

	client, _, _ := NewCapturingClientWithRegistrations()

	_, err := client.Register("agent-001", "agent-001", "go", "")
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}
