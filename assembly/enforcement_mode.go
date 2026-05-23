package assembly

// EnforcementMode is the per-agent governance posture sent to the gateway at
// registration.
//
// Mirrors aa_core::EnforcementMode on the wire; the snake_case tokens are
// what the gateway's REST endpoint expects and what its REST -> gRPC bridge
// maps onto RegisterRequest.enforcement_mode (proto enum).
//
// The type is intentionally a string alias rather than an int with iota.
// That keeps the zero value as "" (unset), so JSON `omitempty` drops the
// field from the registration body when the caller doesn't opt in — the
// gateway then applies its server-side default of live enforce, preserving
// the pre-feature wire shape. Matches the design pattern that landed in
// the Python and Node.js SDKs (AAASM-1560, AAASM-1561).
type EnforcementMode string

const (
	// EnforcementModeEnforce — default; deny blocks the action, redact strips secrets.
	EnforcementModeEnforce EnforcementMode = "enforce"

	// EnforcementModeObserve — dry-run; the gateway records what *would* have
	// happened but lets every action through. Surfaced by
	// `aa audit list --dry-run-only`.
	EnforcementModeObserve EnforcementMode = "observe"

	// EnforcementModeDisabled — policy evaluation skipped entirely.
	// Hermetic test environments only.
	EnforcementModeDisabled EnforcementMode = "disabled"
)

// Valid reports whether m is one of the three known posture strings. The
// empty string ("") is also accepted because it means "use the SDK default
// (omit from wire, gateway applies its own default)" — distinct from an
// unknown / malformed value.
func (m EnforcementMode) Valid() bool {
	switch m {
	case "", EnforcementModeEnforce, EnforcementModeObserve, EnforcementModeDisabled:
		return true
	}
	return false
}
