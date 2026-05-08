package assembly

import "context"

// Chain is a minimal callable interface compatible with
// github.com/tmc/langchaingo/chains.Chain without importing that dependency.
// Any struct implementing Call with this signature satisfies the interface.
type Chain interface {
	Call(ctx context.Context, inputs map[string]any) (map[string]any, error)
}

// chainFunc adapts a bare function to the Chain interface.
type chainFunc func(ctx context.Context, inputs map[string]any) (map[string]any, error)

func (f chainFunc) Call(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	return f(ctx, inputs)
}

// WrapChain wraps a Chain so every Call injects the assembly's own agent ID
// as the parentAgentID in context. Child agents that call Init with this context
// auto-inherit parentAgentID without manual WithParentAgentID threading.
//
// Pass-through safety: if the assembly carries no self agent ID, the context
// is not modified and the inner chain is called unchanged.
func WrapChain(a *Assembly, chain Chain) Chain {
	return chainFunc(func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
		if a.opts.agentID != "" {
			ctx = ContextWithParentAgentID(ctx, a.opts.agentID)
		}
		return chain.Call(ctx, inputs)
	})
}
