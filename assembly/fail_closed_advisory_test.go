// Regression tests for AAASM-4019: the SDK must fail closed rather than fail
// open on two version-skew / kill-switch edge cases —
//   1. an unrecognized policy decision code from the FFI governance client, and
//   2. the op-control stream dying while an op is paused.
// Both must deny the tool call under the fail-closed enforce posture and allow
// it only under observe / disabled / fail-open, mirroring the transport-error
// handling in fail_closed_governance_test.go.

package assembly

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/ai-agent-assembly/go-sdk/internal/proto"
)

// TestUnknownDecisionDeniesUnderEnforce is the item-1 regression: a governance
// client that yields a decision code this SDK cannot interpret (version skew)
// must deny the tool under the fail-closed enforce posture rather than run it.
func TestUnknownDecisionDeniesUnderEnforce(t *testing.T) {
	t.Parallel()

	inner := &failClosedTool{name: "web_search", result: "leaked"}
	// decision 99 is not one of DecisionAllow/Deny/Pending/Redact, so Check
	// surfaces an error, which the enforce posture turns into a denial.
	client := newFFIGovernanceClient(&fakeQuerier{decision: 99})
	opts := defaultRuntimeOptions()
	opts.enforcementMode = EnforcementModeEnforce
	wrapped := newAssemblyTool(inner, client, opts)

	_, err := wrapped.Call(context.Background(), "query")
	if err == nil {
		t.Fatal("expected an unrecognized decision to deny under fail-closed enforce")
	}
	if inner.calls != 0 {
		t.Fatalf("inner tool ran %d times, want 0 (unknown decision must block under enforce)", inner.calls)
	}
}

// TestUnknownDecisionAllowsUnderObserve verifies the observe posture still lets
// the call through on an unrecognized decision, so the gateway can shadow-audit
// without the SDK short-circuiting the call.
func TestUnknownDecisionAllowsUnderObserve(t *testing.T) {
	t.Parallel()

	inner := &failClosedTool{name: "web_search", result: "ok"}
	client := newFFIGovernanceClient(&fakeQuerier{decision: 99})
	opts := defaultRuntimeOptions()
	opts.enforcementMode = EnforcementModeObserve
	wrapped := newAssemblyTool(inner, client, opts)

	result, err := wrapped.Call(context.Background(), "query")
	if err != nil {
		t.Fatalf("expected observe posture to allow on unknown decision, got %v", err)
	}
	if result != "ok" || inner.calls != 1 {
		t.Fatalf("expected inner tool to run once with result ok, got result=%q calls=%d", result, inner.calls)
	}
}

// TestWaitForOp_PausedStreamDeathFailsClosed is the item-3 regression at the
// subscriber level: a paused op whose control stream dies while a caller is
// blocked must return ErrOpControlUnavailable, not nil — the pause can no longer
// be lifted, so the op stays blocked rather than silently resuming.
func TestWaitForOp_PausedStreamDeathFailsClosed(t *testing.T) {
	sub, stream, _ := newSubscriber(t)
	stream.push(msg("op-death", pb.OpControlSignal_OP_CONTROL_SIGNAL_PAUSE, 0))
	waitFor(t, func() bool { return sub.IsPaused("op-death") }, time.Second)

	done := make(chan error, 1)
	go func() {
		done <- sub.WaitForOp(context.Background(), "op-death")
	}()

	// The op is paused, so WaitForOp must be blocked.
	select {
	case <-done:
		t.Fatal("WaitForOp returned while op was paused")
	case <-time.After(50 * time.Millisecond):
	}

	stream.end()

	select {
	case err := <-done:
		if !errors.Is(err, ErrOpControlUnavailable) {
			t.Fatalf("WaitForOp returned %v; want ErrOpControlUnavailable on stream death while paused", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForOp did not wake on stream death")
	}
}

// TestWaitForOp_AlreadyDeadWhilePausedFailsClosed covers the early-return branch:
// when the stream is already dead and the op is paused before WaitForOp is
// called, it fails closed immediately.
func TestWaitForOp_AlreadyDeadWhilePausedFailsClosed(t *testing.T) {
	sub, stream, _ := newSubscriber(t)
	stream.push(msg("op-early", pb.OpControlSignal_OP_CONTROL_SIGNAL_PAUSE, 0))
	waitFor(t, func() bool { return sub.IsPaused("op-early") }, time.Second)

	stream.end()
	waitFor(t, func() bool { return !sub.StreamAlive() }, time.Second)

	if err := sub.WaitForOp(context.Background(), "op-early"); !errors.Is(err, ErrOpControlUnavailable) {
		t.Fatalf("WaitForOp returned %v; want ErrOpControlUnavailable when stream already dead and op paused", err)
	}
}

// errOpControl is a minimal OpController that always returns a fixed error,
// standing in for a subscriber whose stream died while an op was paused.
type errOpControl struct{ err error }

func (e errOpControl) WaitForOp(context.Context, string) error { return e.err }

// TestOpControlGate_PausedStreamDeathDeniesUnderEnforce is the item-3 regression
// at the tool-wrapper level: when the op-control stream dies while an op is
// paused (WaitForOp yields ErrOpControlUnavailable), the tool call is denied
// under the fail-closed enforce posture.
func TestOpControlGate_PausedStreamDeathDeniesUnderEnforce(t *testing.T) {
	t.Parallel()

	inner := &failClosedTool{name: "web_search", result: "leaked"}
	opts := defaultRuntimeOptions()
	opts.enforcementMode = EnforcementModeEnforce
	opts.opControl = errOpControl{err: ErrOpControlUnavailable}
	wrapped := newAssemblyTool(inner, &coverageGovernanceClient{}, opts)

	ctx := WithOpID(context.Background(), "op-x")
	_, err := wrapped.Call(ctx, "query")
	if !errors.Is(err, ErrOpControlUnavailable) {
		t.Fatalf("expected op-control stream death to deny under enforce, got %v", err)
	}
	if inner.calls != 0 {
		t.Fatalf("inner tool ran %d times, want 0 (paused-stream-death must block under enforce)", inner.calls)
	}
}

// TestOpControlGate_PausedStreamDeathAllowsUnderObserve verifies the observe
// posture lets the call proceed to the governance gate even when the op-control
// stream died while paused.
func TestOpControlGate_PausedStreamDeathAllowsUnderObserve(t *testing.T) {
	t.Parallel()

	inner := &failClosedTool{name: "web_search", result: "ok"}
	opts := defaultRuntimeOptions()
	opts.enforcementMode = EnforcementModeObserve
	opts.opControl = errOpControl{err: ErrOpControlUnavailable}
	wrapped := newAssemblyTool(inner, &coverageGovernanceClient{}, opts)

	ctx := WithOpID(context.Background(), "op-x")
	result, err := wrapped.Call(ctx, "query")
	if err != nil {
		t.Fatalf("expected observe posture to allow on op-control stream death, got %v", err)
	}
	if result != "ok" || inner.calls != 1 {
		t.Fatalf("expected inner tool to run once with result ok, got result=%q calls=%d", result, inner.calls)
	}
}
