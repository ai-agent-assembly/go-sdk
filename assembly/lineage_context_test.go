package assembly

import (
	"context"
	"errors"
	"testing"
)

func TestParentAgentIDRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := ContextWithParentAgentID(context.Background(), "parent-abc")
	if got := ParentAgentIDFromContext(ctx); got != "parent-abc" {
		t.Fatalf("expected parentAgentID %q, got %q", "parent-abc", got)
	}
}

func TestParentAgentIDFromContextReturnsEmptyWhenAbsent(t *testing.T) {
	t.Parallel()

	if got := ParentAgentIDFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty parentAgentID, got %q", got)
	}
}

func TestSpawnedByToolRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := ContextWithSpawnedByTool(context.Background(), "search_tool")
	if got := SpawnedByToolFromContext(ctx); got != "search_tool" {
		t.Fatalf("expected spawnedByTool %q, got %q", "search_tool", got)
	}
}

func TestSpawnedByToolFromContextReturnsEmptyWhenAbsent(t *testing.T) {
	t.Parallel()

	if got := SpawnedByToolFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty spawnedByTool, got %q", got)
	}
}

func TestParentAgentIDAndAgentIDAreIndependent(t *testing.T) {
	t.Parallel()

	ctx := WithAgentID(context.Background(), "current-agent")
	ctx = ContextWithParentAgentID(ctx, "parent-agent")

	if got := AgentIDFromContext(ctx); got != "current-agent" {
		t.Errorf("current agentID expected %q, got %q", "current-agent", got)
	}
	if got := ParentAgentIDFromContext(ctx); got != "parent-agent" {
		t.Errorf("parentAgentID expected %q, got %q", "parent-agent", got)
	}
}

func TestInitAutoInheritsParentAgentIDFromContext(t *testing.T) {
	originalConnector := sidecarConnector
	t.Cleanup(func() { sidecarConnector = originalConnector })
	sidecarConnector = func(_ context.Context, _ string) (SidecarClient, error) {
		return nil, nil
	}

	ctx := ContextWithParentAgentID(context.Background(), "ctx-parent-agent")

	a, err := Init(ctx,
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.opts.parentAgentID != "ctx-parent-agent" {
		t.Errorf("expected parentAgentID %q from context, got %q", "ctx-parent-agent", a.opts.parentAgentID)
	}
}

func TestInitExplicitParentAgentIDOverridesContext(t *testing.T) {
	originalConnector := sidecarConnector
	t.Cleanup(func() { sidecarConnector = originalConnector })
	sidecarConnector = func(_ context.Context, _ string) (SidecarClient, error) {
		return nil, nil
	}

	ctx := ContextWithParentAgentID(context.Background(), "ctx-parent")

	a, err := Init(ctx,
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		WithParentAgentID("explicit-parent"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.opts.parentAgentID != "explicit-parent" {
		t.Errorf("expected explicit parentAgentID to win, got %q", a.opts.parentAgentID)
	}
}

func TestInitDoesNotSetParentAgentIDWhenContextEmpty(t *testing.T) {
	originalConnector := sidecarConnector
	t.Cleanup(func() { sidecarConnector = originalConnector })
	sidecarConnector = func(_ context.Context, _ string) (SidecarClient, error) {
		return nil, nil
	}

	a, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.opts.parentAgentID != "" {
		t.Errorf("expected empty parentAgentID, got %q", a.opts.parentAgentID)
	}
}

func TestWithSelfAgentIDOption(t *testing.T) {
	t.Parallel()

	opts := defaultRuntimeOptions()
	WithSelfAgentID("agent-self-1")(&opts)

	if opts.agentID != "agent-self-1" {
		t.Errorf("expected agentID %q, got %q", "agent-self-1", opts.agentID)
	}
}

func TestInitWithBootError_DoesNotLeakParentAgentID(t *testing.T) {
	originalConnector := sidecarConnector
	t.Cleanup(func() { sidecarConnector = originalConnector })
	wantErr := errors.New("boot-failure")
	sidecarConnector = func(context.Context, string) (SidecarClient, error) {
		return nil, wantErr
	}

	ctx := ContextWithParentAgentID(context.Background(), "ctx-parent")
	a, err := Init(ctx,
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected boot error %v, got %v", wantErr, err)
	}
	if a != nil {
		t.Fatal("expected nil Assembly on error")
	}
}
