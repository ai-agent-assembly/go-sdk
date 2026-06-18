package assembly

import "testing"

// These tests pin the Error() string contract of the SDK's typed errors —
// the cross-SDK error shape callers match against (AAASM-3178 coverage).

func TestConfigurationError_ErrorPrefixesAssembly(t *testing.T) {
	t.Parallel()

	err := &ConfigurationError{Message: "aasm not on PATH"}
	if got, want := err.Error(), "assembly: aasm not on PATH"; got != want {
		t.Fatalf("ConfigurationError.Error() = %q, want %q", got, want)
	}
}

func TestGatewayError_ErrorPrefixesAssembly(t *testing.T) {
	t.Parallel()

	err := &GatewayError{Message: "gateway did not become ready"}
	if got, want := err.Error(), "assembly: gateway did not become ready"; got != want {
		t.Fatalf("GatewayError.Error() = %q, want %q", got, want)
	}
}

func TestOpTerminatedError_NilReceiverReturnsEmpty(t *testing.T) {
	t.Parallel()

	var err *OpTerminatedError
	if got := err.Error(); got != "" {
		t.Fatalf("nil OpTerminatedError.Error() = %q, want empty", got)
	}
}

func TestOpTerminatedError_WithoutReasonUsesGenericMessage(t *testing.T) {
	t.Parallel()

	err := &OpTerminatedError{OpID: "trace:span"}
	if got, want := err.Error(), "op trace:span was terminated by the gateway"; got != want {
		t.Fatalf("OpTerminatedError.Error() = %q, want %q", got, want)
	}
}

func TestOpTerminatedError_WithReasonIncludesIt(t *testing.T) {
	t.Parallel()

	err := &OpTerminatedError{OpID: "trace:span", Reason: "budget exceeded"}
	if got, want := err.Error(), "op trace:span was terminated: budget exceeded"; got != want {
		t.Fatalf("OpTerminatedError.Error() = %q, want %q", got, want)
	}
}
