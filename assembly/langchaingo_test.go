package assembly

import (
	"context"
	"errors"
	"testing"
)

func TestWrapChainPropagatesParentAgentIDToContext(t *testing.T) {
	t.Parallel()

	captured := ""
	inner := chainFunc(func(ctx context.Context, _ map[string]any) (map[string]any, error) {
		captured = ParentAgentIDFromContext(ctx)
		return nil, nil
	})

	a := &Assembly{opts: runtimeOptions{agentID: "parent-agent-1"}}
	wrapped := WrapChain(a, inner)

	if _, err := wrapped.Call(context.Background(), map[string]any{"input": "hello"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured != "parent-agent-1" {
		t.Errorf("expected parentAgentID %q in context, got %q", "parent-agent-1", captured)
	}
}

func TestWrapChainSkipsEnrichmentWhenAgentIDEmpty(t *testing.T) {
	t.Parallel()

	captured := "initial"
	inner := chainFunc(func(ctx context.Context, _ map[string]any) (map[string]any, error) {
		captured = ParentAgentIDFromContext(ctx)
		return nil, nil
	})

	a := &Assembly{opts: runtimeOptions{}}
	wrapped := WrapChain(a, inner)

	if _, err := wrapped.Call(context.Background(), map[string]any{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured != "" {
		t.Errorf("expected empty parentAgentID when assembly has no agentID, got %q", captured)
	}
}

func TestWrapChainPassesThroughInnerError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("chain-error")
	inner := chainFunc(func(_ context.Context, _ map[string]any) (map[string]any, error) {
		return nil, sentinel
	})

	a := &Assembly{opts: runtimeOptions{agentID: "agent-2"}}
	wrapped := WrapChain(a, inner)

	if _, err := wrapped.Call(context.Background(), nil); !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error %v, got %v", sentinel, err)
	}
}

func TestWrapChainForwardsInputsUnchanged(t *testing.T) {
	t.Parallel()

	gotInputs := map[string]any{}
	inner := chainFunc(func(_ context.Context, inputs map[string]any) (map[string]any, error) {
		gotInputs = inputs
		return inputs, nil
	})

	a := &Assembly{opts: runtimeOptions{agentID: "agent-3"}}
	wrapped := WrapChain(a, inner)

	want := map[string]any{"key": "value", "num": 42}
	if _, err := wrapped.Call(context.Background(), want); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotInputs) != len(want) {
		t.Errorf("expected inputs %v, got %v", want, gotInputs)
	}
}

func TestWrapChainDoesNotOverwriteExistingParentAgentIDInContext(t *testing.T) {
	t.Parallel()

	captured := ""
	inner := chainFunc(func(ctx context.Context, _ map[string]any) (map[string]any, error) {
		captured = ParentAgentIDFromContext(ctx)
		return nil, nil
	})

	a := &Assembly{opts: runtimeOptions{agentID: "new-parent"}}
	wrapped := WrapChain(a, inner)

	// Pre-seeded context — WrapChain should overwrite with assembly's own ID
	ctx := ContextWithParentAgentID(context.Background(), "original-parent")
	if _, err := wrapped.Call(ctx, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured != "new-parent" {
		t.Errorf("expected assembly agentID %q to win, got %q", "new-parent", captured)
	}
}

func TestChainFuncImplementsChainInterface(t *testing.T) {
	t.Parallel()

	var _ Chain = chainFunc(nil)
}
