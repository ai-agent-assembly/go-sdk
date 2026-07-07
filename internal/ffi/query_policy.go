package ffi

import "unsafe"

// AaDecision values mirror the aa-ffi-go C ABI (AaDecision) returned by
// aa_query_policy.
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
	// DecisionUnspecified is the proto3 zero-value verdict ("no decision
	// rendered"). The native shim maps it to this distinct sentinel — never onto
	// DecisionAllow — so a non-authoritative verdict cannot alias a real allow;
	// ffiGovernanceClient.Check routes it through the fail-closed path under
	// enforce (AAASM-4166), matching the Node SDK.
	DecisionUnspecified int32 = 4
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
// value. The no-cgo production build (fallbackUDSBridge) still answers policy
// queries — it reports an unreachable runtime as a non-OK status, so it fails
// closed (errors → deny under enforce), never open. QueryPolicy only fails open
// for in-memory test bindings that do not implement policyQuerier at all.
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
