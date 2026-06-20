package assembly

// WrapTools wraps all tools with the assembly's own governance client so that a
// reachable runtime's DENY blocks the tool call via the native aa_query_policy
// primitive. When no runtime was reachable at Init, the governance client is nil:
// under the fail-closed enforce posture (the default) the wrapped tools deny the
// call rather than run unchecked (AAASM-3109); under the observe/disabled postures
// they pass through so the runtime / proxy / eBPF layers remain authoritative.
//
// The assembly's fail-closed posture and enforcement mode are propagated to the
// wrapper, so a governance check error (or a missing client) blocks the call
// under the fail-closed enforce posture and allows it otherwise.
func (a *Assembly) WrapTools(toolList []Tool) []Tool {
	options := []Option{
		WithFailClosed(a.opts.failClosed),
		withEnforcementMode(a.opts.enforcementMode),
	}
	// Propagate a live op-control subscriber (AAASM-3491 / AAASM-3501) when one
	// was wired at Init via WithOpControl, so the gateway's kill switch reaches
	// the wrapped tool path; the native runtime consumer is unaffected.
	if a.opts.opControl != nil {
		options = append(options, WithOpControl(a.opts.opControl))
	}
	return WrapTools(toolList, a.governance, options...)
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
