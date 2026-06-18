package assembly

import (
	"context"
	"testing"
)

// WaitForApproval and Close have no native primitive yet, so they fail open /
// release nothing. These tests pin that contract (AAASM-3178 coverage).

func TestFFIGovernanceClient_WaitForApprovalFailsOpen(t *testing.T) {
	t.Parallel()

	dec, err := newFFIGovernanceClient(&fakeQuerier{}).WaitForApproval(
		context.Background(), ApprovalRequest{ToolName: "wire"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Denied || dec.Pending {
		t.Fatalf("expected WaitForApproval to fail open with an allow decision, got %+v", dec)
	}
}

func TestFFIGovernanceClient_CloseIsNoOp(t *testing.T) {
	t.Parallel()

	if err := newFFIGovernanceClient(&fakeQuerier{}).Close(); err != nil {
		t.Fatalf("expected Close no-op to return nil, got %v", err)
	}
}

func TestFFIGovernanceClient_CheckFoldsUnknownDecisionToAllow(t *testing.T) {
	t.Parallel()

	// A garbled decision code that is neither deny nor pending must fold onto
	// an allow Decision (the native shim normalises unspecified to allow).
	dec, err := newFFIGovernanceClient(&fakeQuerier{decision: 99}).Check(
		context.Background(), CheckRequest{ToolName: "calc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Denied || dec.Pending {
		t.Fatalf("expected unknown decision to fold to allow, got %+v", dec)
	}
}
