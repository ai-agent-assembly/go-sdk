package ffi

import "unsafe"

// AaDecision values mirror the aa-ffi-go C ABI (AaDecision) returned by
// aa_query_policy. UNSPECIFIED is folded onto ALLOW by the native shim, so it is
// not represented here.
const (
	// DecisionAllow permits the action. The native shim also returns this when
	// the query fails open (unreachable / slow / closed runtime).
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
// It delegates to the native aa_query_policy primitive (AAASM-3048), which is
// advisory and fail-open: an unreachable, slow, or closed runtime returns
// AA_STATUS_OK with DecisionAllow rather than an error, because the runtime /
// proxy / eBPF layers enforce authoritatively. QueryPolicy preserves that
// contract — and additionally fails open when no native binding is compiled in
// or the binding cannot answer policy queries, returning DecisionAllow with a
// nil error so the SDK never blocks on a missing transport.
//
// actionType is a snake_case proto action name (e.g. "tool_call"); toolName and
// argsJSON may be empty.
func (c *Client) QueryPolicy(agentID, actionType, toolName, argsJSON string) (decision int32, reason string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	querier, ok := c.binding.(policyQuerier)
	if !ok {
		// No native policy-query transport: fail open, matching the native
		// shim's behaviour for an unreachable runtime.
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
