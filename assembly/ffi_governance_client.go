package assembly

import (
	"context"

	"github.com/ai-agent-assembly/go-sdk/internal/ffi"
)

// actionTypeToolCall is the snake_case proto action name used when querying the
// runtime for a tool-call policy decision. It matches the shared enforcement
// contract used by the Python and Node SDKs.
const actionTypeToolCall = "tool_call"

// policyQuerier is the subset of *ffi.Client used to query the runtime for a
// policy decision. It is an interface so tests can substitute a fake without a
// live native binding.
type policyQuerier interface {
	QueryPolicy(agentID, actionType, toolName, argsJSON string) (decision int32, reason string, err error)
}

// ffiGovernanceClient is the production GovernanceClient. Its Check delegates to
// the native aa_query_policy primitive (AAASM-3048) through the FFI client and
// maps the returned decision onto a Decision per the shared enforcement
// contract:
//
//	deny                          -> Decision{Denied: true, Reason}
//	allow / redact / unspecified  -> Decision{}
//	pending                       -> Decision{Pending: true} (wrapper waits)
//
// The native query is fail-open: an unreachable, slow, or closed runtime
// returns allow with a nil error, so Check never blocks the agent on a missing
// runtime. A genuine hard error (e.g. no native binding compiled in) is returned
// to the caller; the tool wrapper then honours WithFailClosed.
type ffiGovernanceClient struct {
	querier policyQuerier
}

// newFFIGovernanceClient builds a GovernanceClient backed by the FFI client.
func newFFIGovernanceClient(querier policyQuerier) *ffiGovernanceClient {
	return &ffiGovernanceClient{querier: querier}
}

// Check queries the runtime for a policy decision on a tool call.
func (c *ffiGovernanceClient) Check(_ context.Context, request CheckRequest) (Decision, error) {
	if c.querier == nil {
		return Decision{}, nil
	}

	decision, reason, err := c.querier.QueryPolicy(
		request.AgentID,
		actionTypeToolCall,
		request.ToolName,
		request.Args,
	)
	if err != nil {
		return Decision{}, err
	}

	switch decision {
	case ffi.DecisionDeny:
		return Decision{Denied: true, Reason: reason}, nil
	case ffi.DecisionPending:
		return Decision{Pending: true, Reason: reason}, nil
	default:
		// Allow, redact, and any unspecified/garbled decision proceed. The
		// native shim already folds unspecified onto allow.
		return Decision{}, nil
	}
}

// WaitForApproval has no native approval-polling primitive yet, so it fails open
// and proceeds. The runtime / proxy / eBPF layers remain authoritative.
func (c *ffiGovernanceClient) WaitForApproval(_ context.Context, _ ApprovalRequest) (Decision, error) {
	return Decision{}, nil
}

// RecordResult is a no-op: tool results are reported to the runtime through the
// FFI event channel by the Assembly runtime, not through this client.
func (c *ffiGovernanceClient) RecordResult(_ context.Context, _ RecordRequest) error {
	return nil
}

// Close releases no resources; the underlying FFI client lifecycle is owned by
// the Assembly runtime.
func (c *ffiGovernanceClient) Close() error {
	return nil
}

var _ GovernanceClient = (*ffiGovernanceClient)(nil)
