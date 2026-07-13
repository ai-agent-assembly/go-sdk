ctx := assembly.WithAgentID(context.Background(), "basic-agent-demo")

// Use the offline mock governance client.
// In production, replace mockClient with a client backed by a real gateway.
fmt.Println("[assembly] using offline mock governance client")
client := &mockClient{}

// Wrap the tool — every Call now goes through the governance client first.
tools := assembly.WrapTools([]assembly.Tool{&echoTool{}}, client)
