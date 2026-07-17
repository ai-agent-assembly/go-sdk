package assembly

import (
	"context"
	"errors"
	"testing"

	"github.com/ai-agent-assembly/go-sdk/internal/ffi"
)

// fakeQuerier is a deterministic policyQuerier that records the forwarded
// arguments and returns a fixed decision/reason/error.
type fakeQuerier struct {
	decision int32
	reason   string
	err      error

	agentID    string
	actionType string
	toolName   string
	argsJSON   string
	calls      int
}

func (f *fakeQuerier) QueryPolicy(agentID, actionType, toolName, argsJSON string) (int32, string, error) {
	f.calls++
	f.agentID = agentID
	f.actionType = actionType
	f.toolName = toolName
	f.argsJSON = argsJSON
	return f.decision, f.reason, f.err
}

func TestFFIGovernanceClientCheckMapsDecisions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		decision    int32
		reason      string
		wantDenied  bool
		wantPending bool
		wantReason  string
	}{
		{name: "deny", decision: ffi.DecisionDeny, reason: "blocked", wantDenied: true, wantReason: "blocked"},
		{name: "allow", decision: ffi.DecisionAllow},
		{name: "pending", decision: ffi.DecisionPending, reason: "needs approval", wantPending: true, wantReason: "needs approval"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := newFFIGovernanceClient(&fakeQuerier{decision: tc.decision, reason: tc.reason})
			decision, err := client.Check(context.Background(), CheckRequest{
				ToolName: "web_search",
				Args:     `{"q":"x"}`,
				AgentID:  "agent-1",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if decision.Denied != tc.wantDenied {
				t.Fatalf("Denied = %v, want %v", decision.Denied, tc.wantDenied)
			}
			if decision.Pending != tc.wantPending {
				t.Fatalf("Pending = %v, want %v", decision.Pending, tc.wantPending)
			}
			if decision.Reason != tc.wantReason {
				t.Fatalf("Reason = %q, want %q", decision.Reason, tc.wantReason)
			}
		})
	}
}

// UNSPECIFIED (the proto3 zero value) is non-authoritative: Check must surface an
// error rather than proceed as a silent allow, so the wrapper fails it closed
// under enforce instead of aliasing a real ALLOW (AAASM-4166).
func TestFFIGovernanceClientCheckErrorsOnUnspecifiedDecision(t *testing.T) {
	t.Parallel()

	dec, err := newFFIGovernanceClient(&fakeQuerier{decision: ffi.DecisionUnspecified}).Check(
		context.Background(), CheckRequest{ToolName: "web_search", AgentID: "agent-1"})
	if err == nil {
		t.Fatalf("expected UNSPECIFIED verdict to surface an error, got decision %+v", dec)
	}
	if dec.Denied || dec.Pending {
		t.Fatalf("expected the returned decision to be the non-committal zero value, got %+v", dec)
	}
}

// UNSPECIFIED round-trip: a runtime that returns the proto3 zero-value verdict
// blocks the tool under the fail-closed enforce default — the inner tool never
// runs. Previously UNSPECIFIED folded onto allow and the tool ran unchecked.
func TestWrappedToolUnspecifiedFailsClosedByDefault(t *testing.T) {
	t.Parallel()

	inner := &countingTool{name: "web_search", result: "leaked"}
	client := newFFIGovernanceClient(&fakeQuerier{decision: ffi.DecisionUnspecified})
	wrapped := newAssemblyTool(inner, client, defaultRuntimeOptions())

	_, err := wrapped.Call(context.Background(), `{"q":"secret"}`)
	if err == nil {
		t.Fatal("expected fail-closed default to deny the tool on an UNSPECIFIED verdict")
	}
	if inner.calls != 0 {
		t.Fatalf("inner tool was called %d times, want 0 (UNSPECIFIED must block execution)", inner.calls)
	}
}

func TestFFIGovernanceClientCheckForwardsToolCallContract(t *testing.T) {
	t.Parallel()

	querier := &fakeQuerier{decision: ffi.DecisionAllow}
	client := newFFIGovernanceClient(querier)

	_, err := client.Check(context.Background(), CheckRequest{
		ToolName: "web_search",
		Args:     `{"q":"x"}`,
		AgentID:  "agent-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if querier.agentID != "agent-1" {
		t.Fatalf("agentID = %q, want agent-1", querier.agentID)
	}
	if querier.actionType != "tool_call" {
		t.Fatalf("actionType = %q, want tool_call", querier.actionType)
	}
	if querier.toolName != "web_search" {
		t.Fatalf("toolName = %q, want web_search", querier.toolName)
	}
	if querier.argsJSON != `{"q":"x"}` {
		t.Fatalf("argsJSON = %q, want %q", querier.argsJSON, `{"q":"x"}`)
	}
}

func TestFFIGovernanceClientCheckPropagatesQueryError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("native query failed")
	client := newFFIGovernanceClient(&fakeQuerier{err: sentinel})

	decision, err := client.Check(context.Background(), CheckRequest{ToolName: "web_search", AgentID: "agent-1"})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if decision.Denied {
		t.Fatal("expected non-denied decision on error so the wrapper applies its fail-open policy")
	}
}

func TestFFIGovernanceClientNilQuerierProceeds(t *testing.T) {
	t.Parallel()

	client := newFFIGovernanceClient(nil)
	decision, err := client.Check(context.Background(), CheckRequest{ToolName: "web_search"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Denied || decision.Pending {
		t.Fatalf("expected empty decision, got %+v", decision)
	}
}

// PENDING round-trip: a runtime that returns pending blocks the tool — the
// wrapper routes through WaitForApproval, which denies, so the inner tool never
// runs (AAASM-3920). Previously pending silently fell through to allow.
func TestWrappedToolPendingBlocksInnerCall(t *testing.T) {
	t.Parallel()

	inner := &countingTool{name: "web_search", result: "leaked"}
	client := newFFIGovernanceClient(&fakeQuerier{decision: ffi.DecisionPending, reason: "needs approval"})
	wrapped := newAssemblyTool(inner, client, defaultRuntimeOptions())

	_, err := wrapped.Call(context.Background(), `{"q":"secret"}`)

	var violation *PolicyViolationError
	if !errors.As(err, &violation) {
		t.Fatalf("expected PolicyViolationError, got %v", err)
	}
	if inner.calls != 0 {
		t.Fatalf("inner tool was called %d times, want 0 (pending must block execution)", inner.calls)
	}
}

// DENY round-trip: a runtime that returns deny blocks the tool — the wrapper
// returns PolicyViolationError and the inner tool never runs.
func TestWrappedToolDenyBlocksInnerCall(t *testing.T) {
	t.Parallel()

	inner := &countingTool{name: "web_search", result: "leaked"}
	client := newFFIGovernanceClient(&fakeQuerier{decision: ffi.DecisionDeny, reason: "blocked by policy"})
	wrapped := newAssemblyTool(inner, client, defaultRuntimeOptions())

	_, err := wrapped.Call(context.Background(), `{"q":"secret"}`)

	var violation *PolicyViolationError
	if !errors.As(err, &violation) {
		t.Fatalf("expected PolicyViolationError, got %v", err)
	}
	if violation.ToolName != "web_search" {
		t.Fatalf("ToolName = %q, want web_search", violation.ToolName)
	}
	if violation.Reason != "blocked by policy" {
		t.Fatalf("Reason = %q, want %q", violation.Reason, "blocked by policy")
	}
	if inner.calls != 0 {
		t.Fatalf("inner tool was called %d times, want 0 (deny must block execution)", inner.calls)
	}
}

// No reachable runtime: a query error denies the tool under the fail-closed
// enforce default (AAASM-3108) — the unreachable runtime cannot silently allow
// an unchecked action.
func TestWrappedToolFailsClosedByDefaultWhenRuntimeUnreachable(t *testing.T) {
	t.Parallel()

	inner := &countingTool{name: "web_search", result: "ok"}
	client := newFFIGovernanceClient(&fakeQuerier{err: ffi.ErrRuntimeUnavailable})
	wrapped := newAssemblyTool(inner, client, defaultRuntimeOptions())

	_, err := wrapped.Call(context.Background(), `{"q":"x"}`)
	if err == nil {
		t.Fatal("expected fail-closed default to deny the tool on a query error")
	}
	if inner.calls != 0 {
		t.Fatalf("inner tool was called %d times, want 0 (fail-closed must block execution)", inner.calls)
	}
}

// Opting into fail-open: a query error proceeds and runs the inner tool when
// WithFailClosed(false) is set in an enforcing posture.
func TestWrappedToolFailsOpenWhenOptedOut(t *testing.T) {
	t.Parallel()

	inner := &countingTool{name: "web_search", result: "ok"}
	opts := defaultRuntimeOptions()
	WithFailClosed(false)(&opts)
	client := newFFIGovernanceClient(&fakeQuerier{err: ffi.ErrRuntimeUnavailable})
	wrapped := newAssemblyTool(inner, client, opts)

	result, err := wrapped.Call(context.Background(), `{"q":"x"}`)
	if err != nil {
		t.Fatalf("expected fail-open to proceed, got error %v", err)
	}
	if result != "ok" {
		t.Fatalf("result = %q, want ok", result)
	}
	if inner.calls != 1 {
		t.Fatalf("inner tool was called %d times, want 1 (fail-open must run the tool)", inner.calls)
	}
}

// With WithFailClosed, a query error blocks the tool instead of failing open.
func TestWrappedToolFailsClosedWhenOptedIn(t *testing.T) {
	t.Parallel()

	inner := &countingTool{name: "web_search", result: "ok"}
	opts := defaultRuntimeOptions()
	WithFailClosed(true)(&opts)
	client := newFFIGovernanceClient(&fakeQuerier{err: ffi.ErrRuntimeUnavailable})
	wrapped := newAssemblyTool(inner, client, opts)

	_, err := wrapped.Call(context.Background(), `{"q":"x"}`)
	if err == nil {
		t.Fatal("expected fail-closed to block the tool on a query error")
	}
	if inner.calls != 0 {
		t.Fatalf("inner tool was called %d times, want 0 (fail-closed must block)", inner.calls)
	}
}

// countingTool records how many times Call runs so tests can assert the inner
// tool did or did not execute.
type countingTool struct {
	name   string
	result string
	calls  int
}

func (t *countingTool) Name() string        { return t.name }
func (t *countingTool) Description() string { return "" }
func (t *countingTool) Call(context.Context, string) (string, error) {
	t.calls++
	return t.result, nil
}
