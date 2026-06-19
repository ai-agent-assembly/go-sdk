package assembly

import (
	"context"
	"errors"
	"testing"

	"github.com/ai-agent-assembly/go-sdk/internal/ffi"
)

// withCapturingFFIClient installs a factory that hands boot the given capturing
// client and restores the real factory on cleanup, so a test can drive the full
// Init -> aa_register -> WrapTools -> aa_query_policy path without a native
// binding.
func withCapturingFFIClient(t *testing.T, client *ffi.Client) {
	t.Helper()
	orig := newFFIClient
	newFFIClient = func() *ffi.Client { return client }
	t.Cleanup(func() { newFFIClient = orig })
}

// TestBootRegisterSuccessLeavesAgentUsable is the AAASM-3404 success contract end
// to end: a reachable runtime registers the agent on Init AND the returned
// Assembly is fully usable — WrapTools threads the live governance client so an
// allowed tool runs through the native query path after registration.
func TestBootRegisterSuccessLeavesAgentUsable(t *testing.T) {
	capClient, regs := ffi.NewCapturingClientAllowing()
	withCapturingFFIClient(t, capClient)

	a, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
		WithSelfAgentID("agent-007"),
	)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if len(*regs) != 1 {
		t.Fatalf("registrations = %d, want 1 (Init must call aa_register)", len(*regs))
	}

	inner := &countingTool{name: "web_search", result: "ok"}
	wrapped := a.WrapTools([]Tool{inner})
	result, err := wrapped[0].Call(context.Background(), `{"q":"x"}`)
	if err != nil {
		t.Fatalf("expected an allowed tool to run after a registered Init, got %v", err)
	}
	if result != "ok" || inner.calls != 1 {
		t.Fatalf("inner tool result=%q calls=%d, want ok/1 (agent must be usable post-register)", result, inner.calls)
	}
}

// TestBootRegisterDenyBlocksToolEndToEnd is the AAASM-3404 + AAASM-3048 deny
// contract through a real boot: a reachable runtime registers the agent and then
// a DENY from the native query path blocks a wrapped tool — the wrapper returns
// PolicyViolationError and the inner tool never runs.
func TestBootRegisterDenyBlocksToolEndToEnd(t *testing.T) {
	capClient, regs := ffi.NewCapturingClientDenying("blocked by policy")
	withCapturingFFIClient(t, capClient)

	a, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
		WithSelfAgentID("agent-007"),
	)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if len(*regs) != 1 {
		t.Fatalf("registrations = %d, want 1", len(*regs))
	}

	inner := &countingTool{name: "web_search", result: "leaked"}
	wrapped := a.WrapTools([]Tool{inner})
	_, err = wrapped[0].Call(context.Background(), `{"q":"secret"}`)

	var violation *PolicyViolationError
	if !errors.As(err, &violation) {
		t.Fatalf("expected PolicyViolationError on a runtime DENY, got %v", err)
	}
	if violation.Reason != "blocked by policy" {
		t.Fatalf("Reason = %q, want %q", violation.Reason, "blocked by policy")
	}
	if inner.calls != 0 {
		t.Fatalf("inner tool ran %d times, want 0 (DENY must block end to end)", inner.calls)
	}
}

// TestBootRegisterFailedIsAdvisoryAndAgentUsable strengthens the advisory
// contract for the REGISTER_FAILED outcome specifically (the existing advisory
// test uses GATEWAY_UNREACHABLE): a gateway *rejection* is logged, Init still
// succeeds, and the agent remains usable — the runtime governance client is
// wired regardless of the registration result, so an allowed tool still runs.
func TestBootRegisterFailedIsAdvisoryAndAgentUsable(t *testing.T) {
	capClient, _, regs := ffi.NewCapturingClientWithRegisterStatus(ffi.RegisterFailedStatus)
	withCapturingFFIClient(t, capClient)

	a, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
		WithSelfAgentID("agent-009"),
	)
	if err != nil {
		t.Fatalf("Init must succeed despite a REGISTER_FAILED rejection, got %v", err)
	}
	if len(*regs) != 1 {
		t.Fatalf("register attempts = %d, want 1", len(*regs))
	}

	// The capturing binding does not answer policy queries, so QueryPolicy fails
	// open (allow) — the unregistered-but-wired agent must still run tools.
	inner := &countingTool{name: "web_search", result: "ok"}
	wrapped := a.WrapTools([]Tool{inner})
	result, err := wrapped[0].Call(context.Background(), `{"q":"x"}`)
	if err != nil {
		t.Fatalf("unregistered agent must remain usable, got %v", err)
	}
	if result != "ok" || inner.calls != 1 {
		t.Fatalf("inner tool result=%q calls=%d, want ok/1", result, inner.calls)
	}
}

// TestBootRegistersAcrossEnforcementModes verifies registration is attempted on
// Init regardless of enforcement posture (register is a session-establishing
// call, not gated by enforce/observe/disabled): the gateway issues the
// credential token even in observe/disabled so any later mode switch is already
// authenticated.
func TestBootRegistersAcrossEnforcementModes(t *testing.T) {
	for _, mode := range []EnforcementMode{
		EnforcementModeEnforce,
		EnforcementModeObserve,
		EnforcementModeDisabled,
		"", // gateway default
	} {
		mode := mode
		t.Run(string(mode), func(t *testing.T) {
			capClient, regs := ffi.NewCapturingClientAllowing()
			withCapturingFFIClient(t, capClient)

			_, err := Init(context.Background(),
				WithGatewayURL("https://gateway.example.com"),
				WithAPIKey("test-key"),
				withSidecarAddress("127.0.0.1:50051"),
				WithSelfAgentID("agent-007"),
				WithEnforcementMode(mode),
			)
			if err != nil {
				t.Fatalf("Init(mode=%q): %v", mode, err)
			}
			if len(*regs) != 1 {
				t.Fatalf("mode=%q registrations = %d, want 1 (register is not mode-gated)", mode, len(*regs))
			}
		})
	}
}

// TestEnforcementDenyHonoredUnderEnforce asserts that under the enforce posture a
// concrete runtime DENY blocks the tool end to end through a real boot. (The
// observe/disabled postures only change how check *errors* are treated — a
// positive DENY decision is always honoured — and that error path is covered by
// the fail-closed governance suite.)
func TestEnforcementDenyHonoredUnderEnforce(t *testing.T) {
	capClient, _ := ffi.NewCapturingClientDenying("blocked")
	withCapturingFFIClient(t, capClient)

	a, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
		WithSelfAgentID("agent-007"),
		WithEnforcementMode(EnforcementModeEnforce),
	)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	inner := &countingTool{name: "web_search", result: "leaked"}
	wrapped := a.WrapTools([]Tool{inner})
	_, err = wrapped[0].Call(context.Background(), `{"q":"x"}`)
	var violation *PolicyViolationError
	if !errors.As(err, &violation) {
		t.Fatalf("expected PolicyViolationError under enforce, got %v", err)
	}
	if inner.calls != 0 {
		t.Fatalf("inner tool ran %d times under enforce DENY, want 0", inner.calls)
	}
}
