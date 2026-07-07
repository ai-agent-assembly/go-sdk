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
//	allow / redact                -> Decision{}
//	unspecified                   -> error (fail-closed under enforce): the
//	                                 proto3 zero value is not an authoritative
//	                                 allow (AAASM-4166)
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

// Check queries the runtime for a policy decision on a tool call. It fails
// fast when the context is already cancelled (AAASM-4194), avoiding a blocking
// FFI call that cannot be interrupted once started.
func (c *ffiGovernanceClient) Check(ctx context.Context, request CheckRequest) (Decision, error) {
	// Check context cancellation before the FFI call: the native aa_query_policy
	// primitive is synchronous and cannot be interrupted once started, so an
	// already-cancelled context must short-circuit here (AAASM-4194).
	if err := ctx.Err(); err != nil {
		return Decision{}, fmt.Errorf("assembly: context cancelled before policy check: %w", err)
	}

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
		// Allow and redact proceed.
		return Decision{}, nil
	case ffi.DecisionUnspecified:
		// The proto3 zero value UNSPECIFIED means "no decision rendered" — a
		// non-authoritative verdict. It must NOT proceed as a silent allow: surface
		// an error so the tool wrapper applies its posture (deny under the default
		// enforce; allow only when fail-closed is disabled or under observe /
		// disabled), matching the Node SDK's fail-closed handling (AAASM-4166).
		return Decision{}, fmt.Errorf("assembly: non-authoritative UNSPECIFIED policy verdict")
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
//
// Context cancellation is checked first (AAASM-4194): a cancelled context should
// abort the wait rather than proceeding to a denial verdict.
func (c *ffiGovernanceClient) WaitForApproval(ctx context.Context, _ ApprovalRequest) (Decision, error) {
	if err := ctx.Err(); err != nil {
		return Decision{}, fmt.Errorf("assembly: context cancelled while waiting for approval: %w", err)
	}

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
