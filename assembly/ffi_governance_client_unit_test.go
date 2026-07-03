package assembly

import (
	"context"
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
