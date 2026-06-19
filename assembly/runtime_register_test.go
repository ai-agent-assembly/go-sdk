package assembly

import (
	"context"
	"testing"

	"github.com/ai-agent-assembly/go-sdk/internal/ffi"
)

// TestBootRegistersAgentViaNativeRegisterOnInit is the AAASM-3404 regression: a
// reachable runtime at Init must register the agent over the native aa_register
// primitive (not only via the SendEvent("register") audit event), so the gateway
// issues the credential token that authenticates later policy queries.
func TestBootRegistersAgentViaNativeRegisterOnInit(t *testing.T) {
	capClient, _, regs := ffi.NewCapturingClientWithRegistrations()

	origFactory := newFFIClient
	newFFIClient = func() *ffi.Client { return capClient }
	t.Cleanup(func() { newFFIClient = origFactory })

	_, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
		WithSelfAgentID("agent-007"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(*regs) != 1 {
		t.Fatalf("native registrations = %d, want 1 (Init must call aa_register)", len(*regs))
	}
	got := (*regs)[0]
	if got.AgentID != "agent-007" {
		t.Errorf("registration AgentID = %q, want agent-007", got.AgentID)
	}
	if got.Framework != frameworkGo {
		t.Errorf("registration Framework = %q, want %q", got.Framework, frameworkGo)
	}
	if got.GatewayEndpoint != "https://gateway.example.com" {
		t.Errorf("registration GatewayEndpoint = %q, want the resolved gateway URL", got.GatewayEndpoint)
	}
}
