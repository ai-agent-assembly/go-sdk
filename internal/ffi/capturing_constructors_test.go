package ffi

import "testing"

// TestCapturingClientWithConnectArgs_RecordsHandshakeArgs drives a client
// built from NewCapturingClientWithConnectArgs through Connect and asserts the
// agent id and Go-module SDK version are forwarded into the handshake.
func TestCapturingClientWithConnectArgs_RecordsHandshakeArgs(t *testing.T) {
	t.Parallel()

	client, args := NewCapturingClientWithConnectArgs()
	if err := client.Connect("unix:///tmp/aa.sock", "agent-42", "9.9.9"); err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}
	if *args.AgentID != "agent-42" {
		t.Fatalf("captured agent id = %q, want agent-42", *args.AgentID)
	}
	if *args.SDKVersion != "9.9.9" {
		t.Fatalf("captured sdk version = %q, want 9.9.9", *args.SDKVersion)
	}
}

// TestCapturingClientDenying_BlocksWithReason drives the deny-capable binding
// end to end: Connect, Register (recorded), then QueryPolicy returns DENY with
// the configured reason.
func TestCapturingClientDenying_BlocksWithReason(t *testing.T) {
	t.Parallel()

	client, registrations := NewCapturingClientDenying("blocked by policy")
	if err := client.Connect("ep", "agent-1", "1.0.0"); err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}
	if _, err := client.Register("agent-1", "agent-1", "go", "ep", "team", ""); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if len(*registrations) != 1 {
		t.Fatalf("expected exactly one captured registration, got %d", len(*registrations))
	}

	decision, reason, err := client.QueryPolicy("agent-1", "tool_call", "danger", "{}")
	if err != nil {
		t.Fatalf("QueryPolicy returned error: %v", err)
	}
	if decision != DecisionDeny {
		t.Fatalf("decision = %d, want DecisionDeny", decision)
	}
	if reason != "blocked by policy" {
		t.Fatalf("reason = %q, want %q", reason, "blocked by policy")
	}
}

// TestCapturingClientAllowing_PermitsAction drives the allow-capable binding:
// after Connect, QueryPolicy returns ALLOW.
func TestCapturingClientAllowing_PermitsAction(t *testing.T) {
	t.Parallel()

	client, registrations := NewCapturingClientAllowing()
	if err := client.Connect("ep", "agent-2", "1.0.0"); err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}
	if _, err := client.Register("agent-2", "agent-2", "go", "ep", "team", ""); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if len(*registrations) != 1 {
		t.Fatalf("expected exactly one captured registration, got %d", len(*registrations))
	}

	decision, _, err := client.QueryPolicy("agent-2", "tool_call", "safe", "{}")
	if err != nil {
		t.Fatalf("QueryPolicy returned error: %v", err)
	}
	if decision != DecisionAllow {
		t.Fatalf("decision = %d, want DecisionAllow", decision)
	}
}
