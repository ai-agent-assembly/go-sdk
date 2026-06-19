package ffi

import (
	"errors"
	"strings"
	"testing"
)

// TestStatusToError_RegisterStatusCodes pins the two fail-closed registration
// status codes (AAASM-3404) that aa_register can surface. They are not covered
// by TestStatusToError_RemainingBranches, yet they are the whole point of this
// change: GATEWAY_UNREACHABLE and REGISTER_FAILED must map to *distinct*
// sentinels so the SDK can tell "could not reach the gateway" apart from "the
// gateway rejected us", and both must carry the operation prefix.
func TestStatusToError_RegisterStatusCodes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int32
		want   error
	}{
		{"gateway unreachable", statusGatewayUnreachable, ErrGatewayUnreachable},
		{"register failed", statusRegisterFailed, ErrRegisterFailed},
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

// TestStatusToError_RegisterCodesAreDistinct guards that the two registration
// outcomes never collapse onto each other or onto the runtime-unavailable
// sentinel: a REGISTER_FAILED must not look like a GATEWAY_UNREACHABLE (and vice
// versa), because the boot path's log message and any future retry policy depend
// on telling them apart.
func TestStatusToError_RegisterCodesAreDistinct(t *testing.T) {
	t.Parallel()

	unreachable := statusToError(statusGatewayUnreachable, "register")
	rejected := statusToError(statusRegisterFailed, "register")

	if errors.Is(unreachable, ErrRegisterFailed) {
		t.Fatal("GATEWAY_UNREACHABLE must not match ErrRegisterFailed")
	}
	if errors.Is(rejected, ErrGatewayUnreachable) {
		t.Fatal("REGISTER_FAILED must not match ErrGatewayUnreachable")
	}
	for _, other := range []error{ErrRuntimeUnavailable, ErrNotConnected} {
		if errors.Is(unreachable, other) || errors.Is(rejected, other) {
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
	if err := client.Connect("127.0.0.1:50051"); err != nil {
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

// TestClientRegisterRecordsAttemptOnFailure verifies the register attempt is
// still recorded by the binding even when it fails: the boot path must be able
// to observe that aa_register was actually invoked (one attempt, no silent skip)
// before deciding to proceed unregistered.
func TestClientRegisterRecordsAttemptOnFailure(t *testing.T) {
	t.Parallel()

	client, _, regs := NewCapturingClientWithRegisterStatus(GatewayUnreachableStatus)
	if err := client.Connect("127.0.0.1:50051"); err != nil {
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
