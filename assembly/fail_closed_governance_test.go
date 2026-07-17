package assembly

import (
	"context"
	"errors"
	"testing"
)

// failClosedTool is a minimal Tool whose Call records whether it ran so the
// fail-closed regression tests can assert the inner tool was never executed.
type failClosedTool struct {
	name   string
	result string
	calls  int
}

func (t *failClosedTool) Name() string      { return t.name }
func (*failClosedTool) Description() string { return "desc" }
func (t *failClosedTool) Call(context.Context, string) (string, error) {
	t.calls++
	return t.result, nil
}

// TestNilGovernanceClientDeniesUnderFailClosedEnforce is the AAASM-3109
// regression: a wrapped tool with no governance client (no runtime reachable at
// Init) must deny rather than silently run unchecked under the fail-closed
// enforce default.
func TestNilGovernanceClientDeniesUnderFailClosedEnforce(t *testing.T) {
	t.Parallel()

	inner := &failClosedTool{name: "web_search", result: "leaked"}
	wrapped := newAssemblyTool(inner, nil, defaultRuntimeOptions())

	_, err := wrapped.Call(context.Background(), "query")
	if !errors.Is(err, ErrGovernanceUnavailable) {
		t.Fatalf("expected ErrGovernanceUnavailable, got %v", err)
	}
	if inner.calls != 0 {
		t.Fatalf("inner tool ran %d times, want 0 (nil client must block under fail-closed enforce)", inner.calls)
	}
}

// TestNilGovernanceClientPassesThroughUnderDisabled verifies the disabled
// posture skips governance entirely even with the fail-closed default, so a nil
// client does not block the tool.
func TestNilGovernanceClientPassesThroughUnderDisabled(t *testing.T) {
	t.Parallel()

	inner := &failClosedTool{name: "web_search", result: "ok"}
	opts := defaultRuntimeOptions()
	opts.enforcementMode = EnforcementModeDisabled
	wrapped := newAssemblyTool(inner, nil, opts)

	result, err := wrapped.Call(context.Background(), "query")
	if err != nil {
		t.Fatalf("expected pass-through under disabled posture, got %v", err)
	}
	if result != "ok" || inner.calls != 1 {
		t.Fatalf("expected inner tool to run once with result ok, got result=%q calls=%d", result, inner.calls)
	}
}

// TestNilGovernanceClientPassesThroughWhenFailOpen verifies that opting out of
// fail-closed lets a nil client pass through even in an enforcing posture.
func TestNilGovernanceClientPassesThroughWhenFailOpen(t *testing.T) {
	t.Parallel()

	inner := &failClosedTool{name: "web_search", result: "ok"}
	opts := defaultRuntimeOptions()
	WithFailClosed(false)(&opts)
	wrapped := newAssemblyTool(inner, nil, opts)

	result, err := wrapped.Call(context.Background(), "query")
	if err != nil {
		t.Fatalf("expected pass-through under fail-open, got %v", err)
	}
	if result != "ok" || inner.calls != 1 {
		t.Fatalf("expected inner tool to run once with result ok, got result=%q calls=%d", result, inner.calls)
	}
}

// TestCheckErrorDeniesUnderExplicitEnforceMode is the AAASM-3108 regression for
// the explicit enforce posture: a governance check transport error denies under
// the fail-closed default.
func TestCheckErrorDeniesUnderExplicitEnforceMode(t *testing.T) {
	t.Parallel()

	client := &coverageGovernanceClient{checkErr: errors.New("gateway down")}
	opts := defaultRuntimeOptions()
	opts.enforcementMode = EnforcementModeEnforce
	wrapped := newAssemblyTool(&failClosedTool{name: "web_search", result: "leaked"}, client, opts)

	_, err := wrapped.Call(context.Background(), "query")
	if err == nil {
		t.Fatal("expected check error to deny under fail-closed enforce posture")
	}
}

// TestCheckErrorAllowsUnderObserveMode verifies that the observe posture allows
// on a check error even under the fail-closed default, so the gateway can
// shadow-audit without the SDK short-circuiting the call.
func TestCheckErrorAllowsUnderObserveMode(t *testing.T) {
	t.Parallel()

	client := &coverageGovernanceClient{checkErr: errors.New("gateway down")}
	inner := &failClosedTool{name: "web_search", result: "ok"}
	opts := defaultRuntimeOptions()
	opts.enforcementMode = EnforcementModeObserve
	wrapped := newAssemblyTool(inner, client, opts)

	result, err := wrapped.Call(context.Background(), "query")
	if err != nil {
		t.Fatalf("expected observe posture to allow on check error, got %v", err)
	}
	if result != "ok" || inner.calls != 1 {
		t.Fatalf("expected inner tool to run once with result ok, got result=%q calls=%d", result, inner.calls)
	}
}
