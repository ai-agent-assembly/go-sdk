package ffi

import (
	"errors"
	"strings"
	"testing"
)

// TestStatusToError_RegisterStatusCodes pins the fail-closed registration
// status codes (AAASM-3404, extended by AAASM-6119) that aa_register can
// surface. They are not covered by TestStatusToError_RemainingBranches, yet
// they are the whole point of this change: GATEWAY_UNREACHABLE, REGISTER_FAILED,
// and IDENTITY_UNAVAILABLE must map to *distinct* sentinels so the SDK can tell
// "could not reach the gateway" apart from "the gateway rejected us" apart from
// "this agent has no identity to register with", and all three must carry the
// operation prefix.
func TestStatusToError_RegisterStatusCodes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int32
		want   error
	}{
		{"gateway unreachable", statusGatewayUnreachable, ErrGatewayUnreachable},
		{"register failed", statusRegisterFailed, ErrRegisterFailed},
		{"identity unavailable", statusIdentityUnavailable, ErrIdentityUnavailable},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := statusToError(tc.status, "register")
			if !errors.Is(err, tc.want) {
				t.Fatalf("statusToError(%d) = %v, want chain to %v", tc.status, err, tc.want)
			}
			if !strings.HasPrefix(err.Error(), "register: ") {
				t.Fatalf("expected operation prefix, got %q", err.Error())
			}
		})
	}
}

// TestStatusToError_RegisterCodesAreDistinct guards that the three registration
// outcomes never collapse onto each other or onto the runtime-unavailable
// sentinel: none of GATEWAY_UNREACHABLE, REGISTER_FAILED, or
// IDENTITY_UNAVAILABLE may look like one of the others, because the boot
// path's log message and any future retry/provisioning policy depend on
// telling them apart (AAASM-6119 specifically exists to keep
// IDENTITY_UNAVAILABLE distinguishable from REGISTER_FAILED, where it used to
// be folded).
func TestStatusToError_RegisterCodesAreDistinct(t *testing.T) {
	t.Parallel()

	unreachable := statusToError(statusGatewayUnreachable, "register")
	rejected := statusToError(statusRegisterFailed, "register")
	noIdentity := statusToError(statusIdentityUnavailable, "register")

	if errors.Is(unreachable, ErrRegisterFailed) || errors.Is(unreachable, ErrIdentityUnavailable) {
		t.Fatal("GATEWAY_UNREACHABLE must not match ErrRegisterFailed or ErrIdentityUnavailable")
	}
	if errors.Is(rejected, ErrGatewayUnreachable) || errors.Is(rejected, ErrIdentityUnavailable) {
		t.Fatal("REGISTER_FAILED must not match ErrGatewayUnreachable or ErrIdentityUnavailable")
	}
	if errors.Is(noIdentity, ErrGatewayUnreachable) || errors.Is(noIdentity, ErrRegisterFailed) {
		t.Fatal("IDENTITY_UNAVAILABLE must not match ErrGatewayUnreachable or ErrRegisterFailed")
	}
	for _, other := range []error{ErrRuntimeUnavailable, ErrNotConnected} {
		if errors.Is(unreachable, other) || errors.Is(rejected, other) || errors.Is(noIdentity, other) {
			t.Fatalf("register status unexpectedly matched %v", other)
		}
	}
}

// TestClientRegisterSurfacesRegisterFailed complements TestClientRegisterSurfacesFailure
// (which exercises GATEWAY_UNREACHABLE): a binding that reports REGISTER_FAILED
// must surface ErrRegisterFailed with no policy id, so the boot path logs the
// rejection and proceeds unregistered.
func TestClientRegisterSurfacesRegisterFailed(t *testing.T) {
	t.Parallel()

	client, _, _ := NewCapturingClientWithRegisterStatus(RegisterFailedStatus)
	if err := client.Connect("127.0.0.1:50051", "", ""); err != nil {
		t.Fatalf("connect: %v", err)
	}

	policyID, err := client.Register("agent-001", "agent-001", "go", "", "", "")
	if !errors.Is(err, ErrRegisterFailed) {
		t.Fatalf("expected ErrRegisterFailed, got %v", err)
	}
	if policyID != "" {
		t.Fatalf("policyID = %q, want empty on a rejected registration", policyID)
	}
}

// TestClientRegisterSurfacesIdentityUnavailable complements
// TestClientRegisterSurfacesRegisterFailed: a binding that reports
// IDENTITY_UNAVAILABLE must surface ErrIdentityUnavailable (not
// ErrRegisterFailed) with no policy id, so a caller can distinguish "this
// agent needs key provisioning" from "the gateway rejected us" (AAASM-6119).
func TestClientRegisterSurfacesIdentityUnavailable(t *testing.T) {
	t.Parallel()

	client, _, _ := NewCapturingClientWithRegisterStatus(IdentityUnavailableStatus)
	if err := client.Connect("127.0.0.1:50051", "", ""); err != nil {
		t.Fatalf("connect: %v", err)
	}

	policyID, err := client.Register("agent-001", "agent-001", "go", "", "", "")
	if !errors.Is(err, ErrIdentityUnavailable) {
		t.Fatalf("expected ErrIdentityUnavailable, got %v", err)
	}
	if errors.Is(err, ErrRegisterFailed) {
		t.Fatal("IDENTITY_UNAVAILABLE must not also match ErrRegisterFailed")
	}
	if policyID != "" {
		t.Fatalf("policyID = %q, want empty on a rejected registration", policyID)
	}
}

// TestClientRegisterRecordsAttemptOnFailure verifies the register attempt is
// still recorded by the binding even when it fails: the boot path must be able
// to observe that aa_register was actually invoked (one attempt, no silent skip)
// before deciding to proceed unregistered.
func TestClientRegisterRecordsAttemptOnFailure(t *testing.T) {
	t.Parallel()

	client, _, regs := NewCapturingClientWithRegisterStatus(GatewayUnreachableStatus)
	if err := client.Connect("127.0.0.1:50051", "", ""); err != nil {
		t.Fatalf("connect: %v", err)
	}

	if _, err := client.Register("agent-007", "agent-007", "go", "ep", "", ""); !errors.Is(err, ErrGatewayUnreachable) {
		t.Fatalf("expected ErrGatewayUnreachable, got %v", err)
	}
	if len(*regs) != 1 {
		t.Fatalf("registrations recorded = %d, want 1 (attempt must be observable)", len(*regs))
	}
	if got := (*regs)[0]; got.AgentID != "agent-007" || got.GatewayEndpoint != "ep" {
		t.Fatalf("recorded attempt = %+v, want agentID/endpoint preserved", got)
	}
}
