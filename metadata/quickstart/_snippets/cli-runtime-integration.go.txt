ctx := assembly.WithAgentID(context.Background(), "cli-runtime-demo")

// Attempt to start the aasm sidecar. This shows the CLI runtime integration
// path — the sidecar runs alongside the Go process and intercepts governance calls.
sidecarRunning := startSidecar()

// Build the governance client. When the sidecar is running, production apps
// would point a real transport at the sidecar endpoint. This example uses
// the offline mock to remain runnable in CI without a live gateway.
client := buildGovernanceClient(sidecarRunning)
defer client.Close()

tools := assembly.WrapTools([]assembly.Tool{&echoTool{}}, client)
