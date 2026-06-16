package assembly

// WrapTools wraps all tools with the assembly's own governance client so that a
// reachable runtime's DENY blocks the tool call via the native aa_query_policy
// primitive. When no runtime was reachable at Init, the governance client is nil
// and the wrapped tools pass through unchecked (the runtime / proxy / eBPF
// layers remain authoritative).
//
// The assembly's WithFailClosed posture is propagated to the wrapper, so a
// governance check error blocks the call only when fail-closed was opted into.
func (a *Assembly) WrapTools(toolList []Tool) []Tool {
	return WrapTools(toolList, a.governance, WithFailClosed(a.opts.failClosed))
}

// WrapTools wraps all tools with AssemblyTool governance interception.
func WrapTools(toolList []Tool, client GovernanceClient, options ...Option) []Tool {
	runtimeOpts := defaultRuntimeOptions()
	for _, option := range options {
		if option != nil {
			option(&runtimeOpts)
		}
	}

	wrapped := make([]Tool, len(toolList))
	for index, tool := range toolList {
		wrapped[index] = NewAssemblyTool(tool, client, runtimeOpts)
	}

	return wrapped
}
