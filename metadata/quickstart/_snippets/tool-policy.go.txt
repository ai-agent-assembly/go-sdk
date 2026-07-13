ctx := assembly.WithAgentID(context.Background(), "tool-policy-demo")

fmt.Printf("[policy] client loaded: read-file=ALLOW, delete-file=DENY\n\n")

client := &policyClient{}
tools := assembly.WrapTools(
	[]assembly.Tool{&readFileTool{}, &deleteFileTool{}},
	client,
)
