package ffi

import (
	"errors"
	"testing"
	"unsafe"
)

// A binding that does not implement policyQuerier must make QueryPolicy fail
// open: an unreachable / unsupported policy-query transport never blocks.
func TestQueryPolicyFailsOpenWithoutPolicyQuerier(t *testing.T) {
	t.Parallel()

	client := NewClient(&mockBinding{})
	if err := client.Connect("unix:///tmp/aa.sock", "", ""); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	decision, reason, err := client.QueryPolicy("agent-1", "tool_call", "web_search", `{"q":"x"}`)
	if err != nil {
		t.Fatalf("expected fail-open nil error, got %v", err)
	}
	if decision != DecisionAllow {
		t.Fatalf("expected DecisionAllow, got %d", decision)
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got %q", reason)
	}
}

// When a binding does implement policyQuerier, QueryPolicy returns its decision
// and reason verbatim and surfaces a non-OK status as an error.
func TestQueryPolicyDelegatesToPolicyQuerier(t *testing.T) {
	t.Parallel()

	binding := &queryingBinding{decision: DecisionDeny, reason: "blocked by policy", status: statusOK}
	client := NewClient(binding)
	if err := client.Connect("unix:///tmp/aa.sock", "", ""); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	decision, reason, err := client.QueryPolicy("agent-1", "tool_call", "web_search", `{"q":"x"}`)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if decision != DecisionDeny {
		t.Fatalf("expected DecisionDeny, got %d", decision)
	}
	if reason != "blocked by policy" {
		t.Fatalf("expected reason %q, got %q", "blocked by policy", reason)
	}
	if binding.agentID != "agent-1" || binding.actionType != "tool_call" || binding.toolName != "web_search" {
		t.Fatalf("unexpected forwarded args: %+v", binding)
	}
}

// A non-OK status from the querier is surfaced as an error with a fail-open
// decision, so the caller can apply its own fail-open / fail-closed policy.
func TestQueryPolicySurfacesNonOKStatus(t *testing.T) {
	t.Parallel()

	binding := &queryingBinding{decision: DecisionAllow, status: statusRuntimeUnavailable}
	client := NewClient(binding)
	if err := client.Connect("unix:///tmp/aa.sock", "", ""); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	decision, _, err := client.QueryPolicy("agent-1", "tool_call", "", "")
	if !errors.Is(err, ErrRuntimeUnavailable) {
		t.Fatalf("expected ErrRuntimeUnavailable, got %v", err)
	}
	if decision != DecisionAllow {
		t.Fatalf("expected DecisionAllow on error, got %d", decision)
	}
}

// QueryPolicy before Connect reports not-connected (fail-open decision).
func TestQueryPolicyNotConnected(t *testing.T) {
	t.Parallel()

	client := NewClient(&queryingBinding{decision: DecisionDeny, status: statusOK})

	decision, _, err := client.QueryPolicy("agent-1", "tool_call", "", "")
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
	if decision != DecisionAllow {
		t.Fatalf("expected DecisionAllow when not connected, got %d", decision)
	}
}

// queryingBinding implements both binding and policyQuerier, recording the
// forwarded arguments for assertion.
type queryingBinding struct {
	decision   int32
	reason     string
	status     int32
	agentID    string
	actionType string
	toolName   string
	argsJSON   string
}

func (q *queryingBinding) connect(string, string, string) (unsafe.Pointer, int32) {
	handle := new(byte)
	return unsafe.Pointer(handle), statusOK
}

func (q *queryingBinding) sendEvent(unsafe.Pointer, string, string) int32 {
	return statusOK
}

func (q *queryingBinding) disconnect(unsafe.Pointer) int32 {
	return statusOK
}

func (q *queryingBinding) queryPolicy(_ unsafe.Pointer, agentID, actionType, toolName, argsJSON string) (int32, string, int32) {
	q.agentID = agentID
	q.actionType = actionType
	q.toolName = toolName
	q.argsJSON = argsJSON
	return q.decision, q.reason, q.status
}
