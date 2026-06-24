package assembly

import (
	"context"
	"testing"

	"github.com/ai-agent-assembly/go-sdk/internal/ffi"
)

// TestBootForwardsAgentIDAndSDKVersionToHandshake is the AAASM-3683 integration
// contract: a reachable runtime makes boot forward the configured agent id AND
// the published Go-module SDK version (the package Version constant) into the FFI
// connect call, so the installed package version — not the shared aa-sdk-client
// crate version — is signed into the runtime handshake for downgrade detection.
func TestBootForwardsAgentIDAndSDKVersionToHandshake(t *testing.T) {
	capClient, connectArgs := ffi.NewCapturingClientWithConnectArgs()
	withCapturingFFIClient(t, capClient)

	_, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
		WithSelfAgentID("agent-007"),
	)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	if got := *connectArgs.AgentID; got != "agent-007" {
		t.Errorf("connect agentID = %q, want %q (boot must forward WithSelfAgentID)", got, "agent-007")
	}
	// boot forwards the package Version constant, not a hard-coded string, so the
	// signed version always tracks the installed Go SDK release.
	if got := *connectArgs.SDKVersion; got != Version {
		t.Errorf("connect sdkVersion = %q, want %q (boot must forward the package Version constant)", got, Version)
	}
}

// TestBootForwardsEmptyAgentIDWhenUnset confirms that when no self agent id is
// configured boot still forwards the SDK Version but leaves the agent id empty,
// matching the connect contract where an unset agent id is absent from the
// handshake while the SDK version is always present (AAASM-3683).
func TestBootForwardsEmptyAgentIDWhenUnset(t *testing.T) {
	capClient, connectArgs := ffi.NewCapturingClientWithConnectArgs()
	withCapturingFFIClient(t, capClient)

	_, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
	)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	if got := *connectArgs.AgentID; got != "" {
		t.Errorf("connect agentID = %q, want empty when WithSelfAgentID is unset", got)
	}
	if got := *connectArgs.SDKVersion; got != Version {
		t.Errorf("connect sdkVersion = %q, want %q even when agent id is unset", got, Version)
	}
}
