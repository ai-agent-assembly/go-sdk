package assembly

import (
	"errors"
	"time"
)

// Option mutates runtime options during Assembly construction.
type Option func(*runtimeOptions)

type runtimeOptions struct {
	gatewayURL       string
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
	errs             []error
}

// WithGatewayURL sets the governance gateway URL. This option is required;
// [Init] returns [ErrInvalidGateway] if it is not set.
func WithGatewayURL(gatewayURL string) Option {
	return func(opts *runtimeOptions) {
		opts.gatewayURL = gatewayURL
	}
}

// WithAPIKey sets the governance API key. This option is required;
// [Init] returns [ErrInvalidAPIKey] if it is not set.
func WithAPIKey(apiKey string) Option {
	return func(opts *runtimeOptions) {
		opts.apiKey = apiKey
	}
}

// WithFailClosed toggles gateway failure behavior. When true, a governance
// check failure causes the tool call to be rejected. When false (the default),
// the tool call proceeds even if the governance check fails.
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

func withSidecarAddress(sidecarAddress string) Option {
	return func(opts *runtimeOptions) {
		opts.sidecarAddress = sidecarAddress
	}
}
