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

	policyID, err := client.Register("agent-001", "agent-001", "go", "http://127.0.0.1:50051", "", "")
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
	// No lineage supplied ⇒ both fields empty (team-unscoped / root).
	if got.TeamID != "" || got.ParentAgentID != "" {
		t.Fatalf("expected empty lineage, got TeamID=%q ParentAgentID=%q", got.TeamID, got.ParentAgentID)
	}
}

// TestClientRegisterForwardsLineage verifies Register marshals teamID and
// parentAgentID through to the binding's native register call (AAASM-3444),
// asserting on the exact values forwarded for the team-only, parent-only, both,
// and neither scenarios (mirrors the Python/Node lineage tests, AAASM-3415).
func TestClientRegisterForwardsLineage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		teamID     string
		parentID   string
		wantTeam   string
		wantParent string
	}{
		{name: "neither", teamID: "", parentID: "", wantTeam: "", wantParent: ""},
		{name: "team_only", teamID: "team-platform", parentID: "", wantTeam: "team-platform", wantParent: ""},
		{name: "parent_only", teamID: "", parentID: "agent-orchestrator", wantTeam: "", wantParent: "agent-orchestrator"},
		{name: "both", teamID: "team-platform", parentID: "agent-orchestrator", wantTeam: "team-platform", wantParent: "agent-orchestrator"},
		// Round-trip unicode + long ids: forwarded verbatim, no truncation.
		{
			name:       "unicode_and_long",
			teamID:     "团队-αβγ-" + longID(),
			parentID:   "parent-日本語-" + longID(),
			wantTeam:   "团队-αβγ-" + longID(),
			wantParent: "parent-日本語-" + longID(),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, _, regs := NewCapturingClientWithRegistrations()
			if err := client.Connect("127.0.0.1:50051"); err != nil {
				t.Fatalf("connect: %v", err)
			}

			if _, err := client.Register("agent-001", "agent-001", "go", "", tc.teamID, tc.parentID); err != nil {
				t.Fatalf("register: %v", err)
			}

			if len(*regs) != 1 {
				t.Fatalf("registrations = %d, want 1", len(*regs))
			}
			got := (*regs)[0]
			if got.TeamID != tc.wantTeam {
				t.Fatalf("TeamID = %q, want %q", got.TeamID, tc.wantTeam)
			}
			if got.ParentAgentID != tc.wantParent {
				t.Fatalf("ParentAgentID = %q, want %q", got.ParentAgentID, tc.wantParent)
			}
		})
	}
}

// longID returns a long (>256 char) id segment to exercise the no-truncation
// round-trip of lineage values across Register.
func longID() string {
	b := make([]byte, 300)
	for i := range b {
		b[i] = byte('a' + (i % 26))
	}
	return string(b)
}

// TestClientRegisterNotConnected fails before connect with ErrNotConnected.
func TestClientRegisterNotConnected(t *testing.T) {
	t.Parallel()

	client, _, _ := NewCapturingClientWithRegistrations()

	_, err := client.Register("agent-001", "agent-001", "go", "", "", "")
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}

// TestClientRegisterNoRegistererBinding verifies a binding without the registerer
// capability reports the runtime as unavailable so the boot path proceeds
// unregistered.
func TestClientRegisterNoRegistererBinding(t *testing.T) {
	t.Parallel()

	client := NewClient(&mockBinding{})

	_, err := client.Register("agent-001", "agent-001", "go", "", "", "")
	if !errors.Is(err, ErrRuntimeUnavailable) {
		t.Fatalf("expected ErrRuntimeUnavailable, got %v", err)
	}
}

// TestClientRegisterSurfacesFailure verifies a gateway-unreachable status from
// the binding surfaces as ErrGatewayUnreachable (Register itself does not fail
// open; the boot path decides to proceed advisorily).
func TestClientRegisterSurfacesFailure(t *testing.T) {
	t.Parallel()

	client, _, _ := NewCapturingClientFailingRegister()
	if err := client.Connect("127.0.0.1:50051"); err != nil {
		t.Fatalf("connect: %v", err)
	}

	policyID, err := client.Register("agent-001", "agent-001", "go", "", "", "")
	if !errors.Is(err, ErrGatewayUnreachable) {
		t.Fatalf("expected ErrGatewayUnreachable, got %v", err)
	}
	if policyID != "" {
		t.Fatalf("policyID = %q, want empty on failure", policyID)
	}
}
