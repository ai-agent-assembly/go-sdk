// Wrap the LangChainGo tools with Agent Assembly governance. The wrapped
// values still satisfy langchaingo's tools.Tool, so they can be handed
// straight to a LangChainGo agent/executor.
governed := assembly.WrapTools(
	[]assembly.Tool{&searchTool{}, &sendEmailTool{}},
	&policyClient{},
)
