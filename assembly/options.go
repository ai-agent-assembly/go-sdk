package assembly

import (
	"errors"
	"time"
)

// Option mutates runtime options during Assembly construction.
type Option func(*runtimeOptions)

type runtimeOptions struct {
	gatewayURL       string
	controlPlaneURL  string
	apiKey           string
	failClosed       bool
	timeout          time.Duration
	sidecarAddress   string
	sidecarBinary    string
	agentID          string
	parentAgentID    string
	teamID           string
	delegationReason string
	spawnedByTool    string
	enforcementMode  EnforcementMode
	errs             []error
}

// WithGatewayURL sets the governance gateway URL. This option is required;
// [Init] returns [ErrInvalidGateway] if it is not set.
func WithGatewayURL(gatewayURL string) Option {
	return func(opts *runtimeOptions) {
		opts.gatewayURL = gatewayURL
	}
}

// WithControlPlaneURL sets the HTTP control-plane URL. The value is stored on
// the runtime options for future HTTP control-plane consumers; the Go SDK has
// no HTTP control-plane caller today (lifecycle is delegated to the aasm
// runtime). The field is filed now so the config shape stays consistent with
// the Python and Node SDKs and is ready for the first HTTP caller.
//
// When this option is not set, the URL falls back to the AA_CONTROL_PLANE_URL
// environment variable at resolution time.
func WithControlPlaneURL(controlPlaneURL string) Option {
	return func(opts *runtimeOptions) {
		opts.controlPlaneURL = controlPlaneURL
	}
}

// WithAPIKey sets the governance API key. This option is required;
// [Init] returns [ErrInvalidAPIKey] if it is not set.
func WithAPIKey(apiKey string) Option {
	return func(opts *runtimeOptions) {
		opts.apiKey = apiKey
	}
}

// WithFailClosed toggles gateway failure behavior. When true (the default,
// AAASM-3108), a governance check failure — a transport error or timeout —
// denies the tool call so an unreachable gateway cannot silently allow an
// unchecked action. Pass false to opt into fail-open, allowing the call to
// proceed when the check fails. The fail-open path is honored only in the
// observe and disabled enforcement postures; in enforce mode a check error
// always denies.
func WithFailClosed(failClosed bool) Option {
	return func(opts *runtimeOptions) {
		opts.failClosed = failClosed
	}
}

// WithTimeout sets the gateway check timeout. If not set, the default
// timeout is 500ms. The timeout is applied only when the caller's context
// does not already carry a deadline.
func WithTimeout(timeout time.Duration) Option {
	return func(opts *runtimeOptions) {
		opts.timeout = timeout
	}
}

// WithSidecarBinary sets the path to the sidecar binary for managed lifecycle.
// When set, [Init] launches the sidecar as a subprocess and waits for it to
// become healthy before returning. If not set, the SDK connects to an
// already-running sidecar.
func WithSidecarBinary(path string) Option {
	return func(opts *runtimeOptions) {
		opts.sidecarBinary = path
	}
}

// WithSelfAgentID records this agent's own ID for lineage tracking.
// When WrapChain is used, this ID is propagated to child agents via context
// so they can auto-register their parentAgentID without manual threading.
func WithSelfAgentID(agentID string) Option {
	return func(opts *runtimeOptions) {
		opts.agentID = agentID
	}
}

// WithParentAgentID sets the parent agent ID for topology tracking.
// When provided, the gateway records this agent as a child of the specified parent.
func WithParentAgentID(parentAgentID string) Option {
	return func(opts *runtimeOptions) {
		opts.parentAgentID = parentAgentID
	}
}

// WithTeamID sets the team ID this agent belongs to for budget and policy scoping.
func WithTeamID(teamID string) Option {
	return func(opts *runtimeOptions) {
		opts.teamID = teamID
	}
}

// WithDelegationReason provides a human-readable explanation for why this agent
// was delegated to by its parent. The reason must be 256 characters or fewer;
// longer values are rejected via the option's error-collecting field.
func WithDelegationReason(reason string) Option {
	return func(opts *runtimeOptions) {
		if len(reason) > 256 {
			opts.errs = append(opts.errs, errors.New("assembly: delegationReason must be <= 256 characters"))
			return
		}
		opts.delegationReason = reason
	}
}

// WithSpawnedByTool records the name of the tool that spawned this agent.
func WithSpawnedByTool(tool string) Option {
	return func(opts *runtimeOptions) {
		opts.spawnedByTool = tool
	}
}

// WithEnforcementMode sets the per-agent governance posture sent to the
// gateway at registration. Pass [EnforcementModeObserve] to register this
// agent in dry-run / sandbox mode (every action proceeds; the gateway
// records would-be violations as shadow audit events surfaced by
// `aa audit list --dry-run-only`).
//
// When this option is not called, the field is omitted from the registration
// body and the gateway applies its server-side default of live enforcement —
// the pre-feature wire shape is preserved.
//
// Unknown values are rejected via the option's error-collecting field; the
// resulting error surfaces from [Init].
func WithEnforcementMode(mode EnforcementMode) Option {
	return func(opts *runtimeOptions) {
		if !mode.Valid() {
			opts.errs = append(opts.errs, errors.New("assembly: enforcement mode must be one of: enforce, observe, disabled"))
			return
		}
		opts.enforcementMode = mode
	}
}

func withSidecarAddress(sidecarAddress string) Option {
	return func(opts *runtimeOptions) {
		opts.sidecarAddress = sidecarAddress
	}
}

// withEnforcementMode propagates the Assembly's resolved enforcement posture
// into a tool wrapper so the fail-closed gate (AAASM-3108) can allow on check
// error under the observe/disabled postures while denying under enforce.
func withEnforcementMode(mode EnforcementMode) Option {
	return func(opts *runtimeOptions) {
		opts.enforcementMode = mode
	}
}
