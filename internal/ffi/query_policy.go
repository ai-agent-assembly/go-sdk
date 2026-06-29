package ffi

import "unsafe"

// AaDecision values mirror the aa-ffi-go C ABI (AaDecision) returned by
// aa_query_policy. UNSPECIFIED is folded onto ALLOW by the native shim, so it is
// not represented here.
const (
	// DecisionAllow permits the action. It is also the placeholder decision
	// returned alongside an error: an unreachable / slow / closed runtime
	// surfaces a non-OK status (fail-closed, AAASM-3920) rather than a silent
	// allow, so callers must consult the error, not just the decision.
	DecisionAllow int32 = 0
	// DecisionDeny blocks the action.
	DecisionDeny int32 = 1
	// DecisionPending holds the action for out-of-band human approval.
	DecisionPending int32 = 2
	// DecisionRedact permits the action but requires sensitive fields redacted.
	DecisionRedact int32 = 3
)

// policyQuerier is an optional capability a binding may implement to answer
// synchronous policy queries via the native aa_query_policy entry point. Only
// the real transports (cgoBridge, fallbackUDSBridge) implement it; in-memory
// test bindings do not, so Client.QueryPolicy fails open for them.
type policyQuerier interface {
	queryPolicy(handle unsafe.Pointer, agentID, actionType, toolName, argsJSON string) (decision int32, reason string, status int32)
}

// QueryPolicy synchronously asks the runtime for a policy decision on an action.
//
// It delegates to the native aa_query_policy primitive (AAASM-3048). The native
// shim no longer folds an unreachable / slow / closed runtime onto an allow:
// such a failure surfaces as a non-OK status (AAASM-3920), which QueryPolicy
// returns as an error so the tool wrapper can apply its fail-open / fail-closed
// posture (deny by default under enforce; allow only when WithFailClosed is
// disabled or under observe / disabled). The decision is always DecisionAllow on
// error so a caller that ignores the error still gets an advisory non-blocking
// value. QueryPolicy still fails open when no native binding is compiled in or
// the binding cannot answer policy queries (in-memory test transports).
//
// actionType is a snake_case proto action name (e.g. "tool_call"); toolName and
// argsJSON may be empty.
func (c *Client) QueryPolicy(agentID, actionType, toolName, argsJSON string) (decision int32, reason string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	querier, ok := c.binding.(policyQuerier)
	if !ok {
		// No native policy-query transport (in-memory test binding): fail open.
		return DecisionAllow, "", nil
	}

	if c.handle == nil {
		return DecisionAllow, "", statusToError(statusNotConnected, "query_policy")
	}

	decision, reason, status := querier.queryPolicy(c.handle, agentID, actionType, toolName, argsJSON)
	if err := statusToError(status, "query_policy"); err != nil {
		return DecisionAllow, "", err
	}
	return decision, reason, nil
}
