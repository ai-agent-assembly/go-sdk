package ffi

import (
	"errors"
	"strings"
	"testing"
)

// TestStatusToError_RemainingBranches covers the status codes not already
// exercised by TestStatusToError so every mapped sentinel and the default
// "unknown status" path is pinned (AAASM-3178 coverage).
func TestStatusToError_RemainingBranches(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int32
		want   error
	}{
		{"ipc", statusIPCError, ErrIPC},
		{"channel closed", statusChannelClosed, ErrChannelClosed},
		{"panic", statusPanic, ErrPanic},
		{"runtime unavailable", statusRuntimeUnavailable, ErrRuntimeUnavailable},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := statusToError(tc.status, "op")
			if !errors.Is(err, tc.want) {
				t.Fatalf("statusToError(%d) = %v, want chain to %v", tc.status, err, tc.want)
			}
			if !strings.HasPrefix(err.Error(), "op: ") {
				t.Fatalf("expected operation prefix, got %q", err.Error())
			}
		})
	}
}

func TestStatusToError_UnknownStatusFallsThrough(t *testing.T) {
	t.Parallel()

	err := statusToError(4242, "do_thing")
	if err == nil {
		t.Fatal("expected an error for an unmapped status code")
	}
	if !strings.Contains(err.Error(), "ffi status 4242") {
		t.Fatalf("expected unmapped status in message, got %q", err.Error())
	}
	// Unknown statuses must not masquerade as any known sentinel.
	for _, sentinel := range []error{ErrNullPointer, ErrNotConnected, ErrPanic, ErrRuntimeUnavailable} {
		if errors.Is(err, sentinel) {
			t.Fatalf("unmapped status unexpectedly matched sentinel %v", sentinel)
		}
	}
}
