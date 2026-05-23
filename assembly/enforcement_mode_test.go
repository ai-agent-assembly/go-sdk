// AAASM-1562 — verifies the EnforcementMode option and its wire-shape
// contract: accepted values flow through, default omits the field, invalid
// strings are rejected via the Init() error.
package assembly

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestEnforcementMode_Valid_AcceptsKnownTokensAndEmpty(t *testing.T) {
	cases := []EnforcementMode{
		"", // unset / SDK default — caller didn't opt in
		EnforcementModeEnforce,
		EnforcementModeObserve,
		EnforcementModeDisabled,
	}
	for _, m := range cases {
		if !m.Valid() {
			t.Errorf("expected %q to be valid", string(m))
		}
	}
}

func TestEnforcementMode_Valid_RejectsUnknownToken(t *testing.T) {
	if EnforcementMode("obesrve").Valid() {
		t.Errorf("expected typo to be rejected as invalid")
	}
	if EnforcementMode("ENFORCE").Valid() {
		t.Errorf("expected uppercase variant to be rejected (wire uses snake_case lowercase)")
	}
}

func TestWithEnforcementMode_StoresValidValue(t *testing.T) {
	for _, mode := range []EnforcementMode{
		EnforcementModeEnforce,
		EnforcementModeObserve,
		EnforcementModeDisabled,
	} {
		opts := newRuntimeOptionsForTest(WithEnforcementMode(mode))
		if opts.enforcementMode != mode {
			t.Errorf("expected enforcementMode=%q, got %q", mode, opts.enforcementMode)
		}
		if len(opts.errs) != 0 {
			t.Errorf("expected no errors for valid mode %q, got %v", mode, opts.errs)
		}
	}
}

func TestWithEnforcementMode_RejectsUnknownValueViaInitError(t *testing.T) {
	// The option's invalid-value path appends to opts.errs, which surfaces
	// when Init() validates. Catches typos like "obesrve" at boot time
	// instead of silently registering under live enforcement.
	_, err := Init(context.Background(),
		WithGatewayURL("http://localhost:8080"),
		WithAPIKey("test-key"),
		WithEnforcementMode("obesrve"),
	)
	if err == nil {
		t.Fatal("expected Init to return an error for unknown enforcement mode")
	}
	if !errorMessageMentions(err, "enforcement mode") {
		t.Errorf("expected error mentioning 'enforcement mode', got: %v", err)
	}
}

func TestBuildRegistrationEvent_EmitsEnforcementModeWhenSet(t *testing.T) {
	// Operator opts in via WithEnforcementMode(EnforcementModeObserve) —
	// the snake_case wire token must land in the registration body so the
	// gateway's REST -> gRPC bridge can map it to RegisterRequest.enforcement_mode.
	opts := newRuntimeOptionsForTest(WithEnforcementMode(EnforcementModeObserve))
	body := buildRegistrationEvent(opts)

	var got map[string]any
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("registration event must be valid JSON, got %q: %v", body, err)
	}
	if got["enforcement_mode"] != "observe" {
		t.Errorf("expected enforcement_mode=\"observe\" in body, got: %v (full body: %s)", got["enforcement_mode"], body)
	}
}

func TestBuildRegistrationEvent_OmitsEnforcementModeByDefault(t *testing.T) {
	// Caller didn't call WithEnforcementMode — the field must be absent from
	// the JSON body so a pre-feature SDK call produces a pre-feature wire
	// shape (the gateway will then apply its server-side default of enforce).
	opts := newRuntimeOptionsForTest()
	body := buildRegistrationEvent(opts)

	var got map[string]any
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("registration event must be valid JSON, got %q: %v", body, err)
	}
	if _, present := got["enforcement_mode"]; present {
		t.Errorf("default registration body must not include enforcement_mode key, got: %s", body)
	}
}

// newRuntimeOptionsForTest applies options and returns the resulting struct.
// Mirrors the pattern used in options_defaults_architecture_test.go.
func newRuntimeOptionsForTest(options ...Option) runtimeOptions {
	var opts runtimeOptions
	for _, opt := range options {
		opt(&opts)
	}
	return opts
}

func errorMessageMentions(err error, fragment string) bool {
	if err == nil {
		return false
	}
	for {
		if msg := err.Error(); contains(msg, fragment) {
			return true
		}
		next := errors.Unwrap(err)
		if next == nil {
			return false
		}
		err = next
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
