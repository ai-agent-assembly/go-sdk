package assembly

import (
	"context"
	"strings"
	"testing"
)

// The accessor functions guard against a nil context so callers in
// library code never panic on an unprimed context. These tests pin the
// nil-context contract for every accessor (AAASM-3178 coverage).

func TestAgentIDFromContext_NilContextReturnsEmpty(t *testing.T) {
	t.Parallel()

	if got := AgentIDFromContext(nil); got != "" { //nolint:staticcheck // explicitly testing nil-ctx guard
		t.Fatalf("expected empty agent id for nil context, got %q", got)
	}
}

func TestTraceIDFromContext_NilContextReturnsEmpty(t *testing.T) {
	t.Parallel()

	if got := TraceIDFromContext(nil); got != "" { //nolint:staticcheck // explicitly testing nil-ctx guard
		t.Fatalf("expected empty trace id for nil context, got %q", got)
	}
}

func TestRunIDFromContext_NilContextGeneratesRunID(t *testing.T) {
	t.Parallel()

	got := RunIDFromContext(nil) //nolint:staticcheck // explicitly testing nil-ctx guard
	if !strings.HasPrefix(got, "run_") {
		t.Fatalf("expected generated run id with run_ prefix for nil context, got %q", got)
	}
}

func TestEnsureRunID_NilContextSeedsBackgroundContext(t *testing.T) {
	t.Parallel()

	ctx, runID := EnsureRunID(nil) //nolint:staticcheck // explicitly testing nil-ctx guard
	if ctx == nil {
		t.Fatal("expected non-nil context from EnsureRunID(nil)")
	}
	if !strings.HasPrefix(runID, "run_") {
		t.Fatalf("expected generated run id, got %q", runID)
	}
	// The returned context must carry the same run id so the tree is stable.
	if got := RunIDFromContext(ctx); got != runID {
		t.Fatalf("expected returned context to carry run id %q, got %q", runID, got)
	}
}

func TestParentAgentIDFromContext_NilContextReturnsEmpty(t *testing.T) {
	t.Parallel()

	if got := ParentAgentIDFromContext(nil); got != "" { //nolint:staticcheck // explicitly testing nil-ctx guard
		t.Fatalf("expected empty parent agent id for nil context, got %q", got)
	}
}

func TestParentAgentIDFromContext_RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := ContextWithParentAgentID(context.Background(), "orchestrator")
	if got := ParentAgentIDFromContext(ctx); got != "orchestrator" {
		t.Fatalf("expected parent agent id orchestrator, got %q", got)
	}
}

func TestSpawnedByToolFromContext_NilContextReturnsEmpty(t *testing.T) {
	t.Parallel()

	if got := SpawnedByToolFromContext(nil); got != "" { //nolint:staticcheck // explicitly testing nil-ctx guard
		t.Fatalf("expected empty spawned-by-tool for nil context, got %q", got)
	}
}

func TestSpawnedByToolFromContext_RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := ContextWithSpawnedByTool(context.Background(), "delegate")
	if got := SpawnedByToolFromContext(ctx); got != "delegate" {
		t.Fatalf("expected spawned-by-tool delegate, got %q", got)
	}
}
