package assembly

import (
	"context"
	"errors"
	"testing"
)

// WaitForApproval has no native approval primitive, so a pending decision can
// never be approved; it must deny (fail-closed) rather than silently allow the
// held call (AAASM-3920). Close still releases nothing (AAASM-3178 coverage).

func TestFFIGovernanceClient_WaitForApprovalDenies(t *testing.T) {
	t.Parallel()

	dec, err := newFFIGovernanceClient(&fakeQuerier{}).WaitForApproval(
		context.Background(), ApprovalRequest{ToolName: "wire"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dec.Denied {
		t.Fatalf("expected WaitForApproval to deny when no approval channel exists, got %+v", dec)
	}
}

func TestFFIGovernanceClient_CloseIsNoOp(t *testing.T) {
	t.Parallel()

	if err := newFFIGovernanceClient(&fakeQuerier{}).Close(); err != nil {
		t.Fatalf("expected Close no-op to return nil, got %v", err)
	}
}

func TestFFIGovernanceClient_CheckErrorsOnUnknownDecision(t *testing.T) {
	t.Parallel()

	// A decision code this SDK does not recognise (version skew) must NOT fold
	// onto a silent allow: Check surfaces an error so the tool wrapper applies
	// its fail-open / fail-closed posture (AAASM-4019).
	dec, err := newFFIGovernanceClient(&fakeQuerier{decision: 99}).Check(
		context.Background(), CheckRequest{ToolName: "calc"})
	if err == nil {
		t.Fatalf("expected unrecognized decision to surface an error, got decision %+v", dec)
	}
	if dec.Denied || dec.Pending {
		t.Fatalf("expected the returned decision to be the non-committal zero value, got %+v", dec)
	}
}

// TestFFIGovernanceClient_CheckFailsFastOnCancelledContext verifies that Check
// returns early when the context is already cancelled, avoiding a blocking FFI
// call that cannot be interrupted once started (AAASM-4194).
func TestFFIGovernanceClient_CheckFailsFastOnCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	querier := &fakeQuerier{}
	client := newFFIGovernanceClient(querier)

	dec, err := client.Check(ctx, CheckRequest{ToolName: "web_search", AgentID: "agent-1"})
	if err == nil {
		t.Fatal("expected Check to return error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected error to wrap context.Canceled, got %v", err)
	}
	if dec.Denied || dec.Pending {
		t.Fatalf("expected the returned decision to be the non-committal zero value, got %+v", dec)
	}
	if querier.calls != 0 {
		t.Fatalf("querier.QueryPolicy was called %d times, want 0 (should short-circuit before FFI)", querier.calls)
	}
}

// TestFFIGovernanceClient_WaitForApprovalFailsFastOnCancelledContext verifies
// that WaitForApproval returns early when the context is already cancelled,
// rather than proceeding to a denial verdict (AAASM-4194).
func TestFFIGovernanceClient_WaitForApprovalFailsFastOnCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := newFFIGovernanceClient(&fakeQuerier{})

	dec, err := client.WaitForApproval(ctx, ApprovalRequest{ToolName: "web_search"})
	if err == nil {
		t.Fatal("expected WaitForApproval to return error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected error to wrap context.Canceled, got %v", err)
	}
	if dec.Denied || dec.Pending {
		t.Fatalf("expected the returned decision to be the non-committal zero value, got %+v", dec)
	}
}
