package assembly

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestOpIDFromContext_NilReturnsEmpty pins the nil-context guard on the
// unexported op-id accessor so library callers never panic on an unprimed
// context.
func TestOpIDFromContext_NilReturnsEmpty(t *testing.T) {
	t.Parallel()

	if got := opIDFromContext(nil); got != "" { //nolint:staticcheck // explicitly testing nil-ctx guard
		t.Fatalf("expected empty op id for nil context, got %q", got)
	}
}

// TestOpIDFromContext_RoundTrip covers the value-present path of the accessor.
func TestOpIDFromContext_RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := WithOpID(context.Background(), "trace:span")
	if got := opIDFromContext(ctx); got != "trace:span" {
		t.Fatalf("expected op id trace:span, got %q", got)
	}
}

// TestSpanIDFromContext_NilReturnsEmpty covers the nil-context guard.
func TestSpanIDFromContext_NilReturnsEmpty(t *testing.T) {
	t.Parallel()

	if got := spanIDFromContext(nil); got != "" { //nolint:staticcheck // explicitly testing nil-ctx guard
		t.Fatalf("expected empty span id for nil context, got %q", got)
	}
}

// TestSpanIDFromContext_NoSpanReturnsEmpty covers the no-valid-span branch.
func TestSpanIDFromContext_NoSpanReturnsEmpty(t *testing.T) {
	t.Parallel()

	if got := spanIDFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty span id when no span is on the context, got %q", got)
	}
}

// TestWithOpControl_SetsOpController covers the WithOpControl option writing
// the subscriber onto the runtime options.
func TestWithOpControl_SetsOpController(t *testing.T) {
	t.Parallel()

	oc := newFakeOpControl()
	opts := defaultRuntimeOptions()
	WithOpControl(oc)(&opts)

	if opts.opControl != oc {
		t.Fatal("expected WithOpControl to set the opControl field")
	}
}

// TestAssemblyWrapTools_WiresOpControlWhenSet covers the branch in
// Assembly.WrapTools that propagates a live op-control subscriber into the
// wrapped tool path: a terminated op fast-fails the call before the gateway
// check runs.
func TestAssemblyWrapTools_WiresOpControlWhenSet(t *testing.T) {
	t.Parallel()

	oc := newFakeOpControl()
	oc.terminated["op-killed"] = true

	a := &Assembly{
		opts:       runtimeOptions{opControl: oc},
		governance: &checkRecordingClient{decision: Decision{}},
	}
	wrapped := a.WrapTools([]Tool{stubTool{name: "t", result: "ok"}})

	ctx := WithOpID(context.Background(), "op-killed")
	_, err := wrapped[0].Call(ctx, "input")

	var termErr *OpTerminatedError
	if !errors.As(err, &termErr) {
		t.Fatalf("expected *OpTerminatedError from a terminated op, got %v", err)
	}
}

// TestGatewayClient_CheckAppliesConfiguredTimeout covers the branch where a
// positive configured timeout (not the built-in default) is applied because
// the caller's context carries no deadline.
func TestGatewayClient_CheckAppliesConfiguredTimeout(t *testing.T) {
	t.Parallel()

	var hasDeadline bool
	client := NewGatewayClient(
		gatewayTransportStub{check: func(ctx context.Context, _ CheckRequest) (Decision, error) {
			_, hasDeadline = ctx.Deadline()
			return Decision{}, nil
		}},
		WithTimeout(50*time.Millisecond),
	)

	if _, err := client.Check(context.Background(), CheckRequest{ToolName: "calc"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasDeadline {
		t.Fatal("expected Check to apply the configured timeout as a deadline")
	}
}

// TestGatewayClient_CheckReturnsCtxErrWhenCancelled covers the early
// cancellation branch: a context already cancelled before the transport runs
// surfaces ctx.Err() rather than calling the transport.
func TestGatewayClient_CheckReturnsCtxErrWhenCancelled(t *testing.T) {
	t.Parallel()

	transportCalled := false
	client := NewGatewayClient(gatewayTransportStub{check: func(context.Context, CheckRequest) (Decision, error) {
		transportCalled = true
		return Decision{}, nil
	}})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Check(ctx, CheckRequest{ToolName: "calc"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if transportCalled {
		t.Fatal("expected the transport not to run for an already-cancelled context")
	}
}
