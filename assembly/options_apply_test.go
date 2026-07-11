package assembly

import (
	"strings"
	"testing"
	"time"
)

// applyOptions runs the functional options against a fresh default option
// set so each WithXxx can be asserted on the resolved runtimeOptions.
func applyOptions(opts ...Option) runtimeOptions {
	cfg := defaultRuntimeOptions()
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	return cfg
}

func TestWithSidecarBinary_SetsPath(t *testing.T) {
	t.Parallel()

	cfg := applyOptions(WithSidecarBinary("/usr/local/bin/aa-sidecar"))
	if cfg.sidecarBinary != "/usr/local/bin/aa-sidecar" {
		t.Fatalf("sidecarBinary = %q, want /usr/local/bin/aa-sidecar", cfg.sidecarBinary)
	}
}

func TestWithSidecarAddress_SetsField(t *testing.T) {
	t.Parallel()

	cfg := applyOptions(WithSidecarAddress("127.0.0.1:50051"))
	if cfg.sidecarAddress != "127.0.0.1:50051" {
		t.Fatalf("sidecarAddress = %q, want 127.0.0.1:50051", cfg.sidecarAddress)
	}
}

func TestWithControlPlaneURL_SetsURL(t *testing.T) {
	t.Parallel()

	cfg := applyOptions(WithControlPlaneURL("http://cp.internal:8080"))
	if cfg.controlPlaneURL != "http://cp.internal:8080" {
		t.Fatalf("controlPlaneURL = %q, want http://cp.internal:8080", cfg.controlPlaneURL)
	}
}

func TestWithTimeout_OverridesDefault(t *testing.T) {
	t.Parallel()

	cfg := applyOptions(WithTimeout(2 * time.Second))
	if cfg.timeout != 2*time.Second {
		t.Fatalf("timeout = %v, want 2s", cfg.timeout)
	}
}

func TestWithEnforcementMode_RejectsUnknownValue(t *testing.T) {
	t.Parallel()

	cfg := applyOptions(WithEnforcementMode(EnforcementMode("bogus")))
	if len(cfg.errs) == 0 {
		t.Fatal("expected an error to be collected for an unknown enforcement mode")
	}
	if !strings.Contains(cfg.errs[0].Error(), "enforcement mode") {
		t.Fatalf("expected enforcement-mode error, got %v", cfg.errs[0])
	}
	// The invalid value must not be stored on the options.
	if cfg.enforcementMode == EnforcementMode("bogus") {
		t.Fatal("expected an invalid enforcement mode to be rejected, not stored")
	}
}

func TestWithEnforcementMode_AcceptsObserve(t *testing.T) {
	t.Parallel()

	cfg := applyOptions(WithEnforcementMode(EnforcementModeObserve))
	if len(cfg.errs) != 0 {
		t.Fatalf("unexpected errors for valid mode: %v", cfg.errs)
	}
	if cfg.enforcementMode != EnforcementModeObserve {
		t.Fatalf("enforcementMode = %q, want observe", cfg.enforcementMode)
	}
}

func TestWithDelegationReason_RejectsOverlongValue(t *testing.T) {
	t.Parallel()

	cfg := applyOptions(WithDelegationReason(strings.Repeat("x", 257)))
	if len(cfg.errs) == 0 {
		t.Fatal("expected an error for a delegation reason over 256 characters")
	}
	if cfg.delegationReason != "" {
		t.Fatalf("expected overlong delegation reason to be rejected, got %q", cfg.delegationReason)
	}
}

func TestWithDelegationReason_AcceptsBoundaryValue(t *testing.T) {
	t.Parallel()

	reason := strings.Repeat("x", 256)
	cfg := applyOptions(WithDelegationReason(reason))
	if len(cfg.errs) != 0 {
		t.Fatalf("unexpected errors for boundary-length reason: %v", cfg.errs)
	}
	if cfg.delegationReason != reason {
		t.Fatal("expected boundary-length delegation reason to be accepted")
	}
}

func TestWithEnforcementMode_Internal_PropagatesWithoutValidation(t *testing.T) {
	t.Parallel()

	// withEnforcementMode is the internal propagation seam used by
	// Assembly.WrapTools — it stores the already-resolved posture verbatim
	// (no re-validation), so it must accept the empty default too.
	cfg := applyOptions(withEnforcementMode(EnforcementModeDisabled))
	if cfg.enforcementMode != EnforcementModeDisabled {
		t.Fatalf("enforcementMode = %q, want disabled", cfg.enforcementMode)
	}
	if len(cfg.errs) != 0 {
		t.Fatalf("internal propagation should not collect errors, got %v", cfg.errs)
	}
}
