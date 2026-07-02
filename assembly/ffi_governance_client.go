package assembly

import (
	"context"
	"fmt"

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
//	pending                       -> Decision{Pending: true}; the wrapper then
//	                                 calls WaitForApproval, which denies (no
//	                                 approval channel exists — AAASM-3920)
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
	case ffi.DecisionAllow, ffi.DecisionRedact:
		// Allow and redact proceed. DecisionAllow (0) also covers the
		// UNSPECIFIED verdict, which the native shim folds onto allow.
		return Decision{}, nil
	default:
		// A decision code this SDK does not recognise — an out-of-range value
		// or a variant a newer gateway added after this SDK was built (version
		// skew). Do not silently allow it: surface an error so the tool wrapper
		// applies its fail-open / fail-closed posture (deny under the default
		// enforce; allow only when fail-closed is disabled or under observe /
		// disabled), exactly as it does for a transport error (AAASM-4019). The
		// native shim already folds an unrecognised proto verdict onto deny;
		// this is the Go-side defence-in-depth for any other querier.
		return Decision{}, fmt.Errorf("assembly: unrecognized policy decision code %d", decision)
	}
}

// WaitForApproval resolves a pending (requires-approval) decision. There is no
// native approval-polling primitive yet, so there is no channel through which an
// operator could ever approve the held call. Returning allow here (the previous
// behaviour) silently downgraded every requires-approval verdict to allow,
// defeating the gateway's hold (AAASM-3920). With no way to obtain approval the
// only safe resolution is to deny, so a pending decision blocks the tool rather
// than running it unapproved. The runtime / proxy / eBPF layers remain
// authoritative; this is the SDK's defence-in-depth fail-closed posture.
func (c *ffiGovernanceClient) WaitForApproval(_ context.Context, _ ApprovalRequest) (Decision, error) {
	return Decision{
		Denied: true,
		Reason: "tool call requires approval but no approval channel is available; denying (fail-closed)",
	}, nil
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
