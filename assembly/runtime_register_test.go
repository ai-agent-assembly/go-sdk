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

// TestBootProceedsWhenRegisterFails is the advisory contract: a failed native
// registration is logged and Init still succeeds (the agent proceeds
// unregistered; the runtime / proxy / eBPF layers remain authoritative). The
// topology audit event must still be emitted.
func TestBootProceedsWhenRegisterFails(t *testing.T) {
	capClient, events, regs := ffi.NewCapturingClientFailingRegister()

	origFactory := newFFIClient
	newFFIClient = func() *ffi.Client { return capClient }
	t.Cleanup(func() { newFFIClient = origFactory })

	_, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
		WithSelfAgentID("agent-009"),
	)
	if err != nil {
		t.Fatalf("expected Init to succeed despite register failure, got %v", err)
	}
	if len(*regs) != 1 {
		t.Fatalf("expected register to be attempted once, got %d", len(*regs))
	}
	if len(*events) == 0 {
		t.Fatal("expected the topology register audit event to still be emitted")
	}
}

// TestBootForwardsExplicitLineageToNativeRegister verifies the agent's team and
// parent set via options ride the native aa_register call (AAASM-3444), so the
// gateway gets team-budget attribution and topology lineage at registration.
func TestBootForwardsExplicitLineageToNativeRegister(t *testing.T) {
	capClient, _, regs := ffi.NewCapturingClientWithRegistrations()

	origFactory := newFFIClient
	newFFIClient = func() *ffi.Client { return capClient }
	t.Cleanup(func() { newFFIClient = origFactory })

	_, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
		WithSelfAgentID("agent-007"),
		WithTeamID("team-platform"),
		WithParentAgentID("agent-orchestrator"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(*regs) != 1 {
		t.Fatalf("native registrations = %d, want 1", len(*regs))
	}
	got := (*regs)[0]
	if got.TeamID != "team-platform" {
		t.Errorf("registration TeamID = %q, want team-platform", got.TeamID)
	}
	if got.ParentAgentID != "agent-orchestrator" {
		t.Errorf("registration ParentAgentID = %q, want agent-orchestrator", got.ParentAgentID)
	}
}

// TestBootAutoFillsAmbientParentOnNativeRegister verifies a parent set only in
// the ambient context (no WithParentAgentID) is auto-filled and forwarded over
// native register, while teamID is NOT auto-filled from context — mirroring the
// Python/Node advisory semantics (AAASM-3415): only parent_agent_id auto-fills.
func TestBootAutoFillsAmbientParentOnNativeRegister(t *testing.T) {
	capClient, _, regs := ffi.NewCapturingClientWithRegistrations()

	origFactory := newFFIClient
	newFFIClient = func() *ffi.Client { return capClient }
	t.Cleanup(func() { newFFIClient = origFactory })

	ctx := ContextWithParentAgentID(context.Background(), "ambient-parent")
	_, err := Init(ctx,
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
		WithSelfAgentID("agent-007"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(*regs) != 1 {
		t.Fatalf("native registrations = %d, want 1", len(*regs))
	}
	got := (*regs)[0]
	if got.ParentAgentID != "ambient-parent" {
		t.Errorf("registration ParentAgentID = %q, want ambient-parent (auto-filled)", got.ParentAgentID)
	}
	if got.TeamID != "" {
		t.Errorf("registration TeamID = %q, want empty (team is not auto-filled from context)", got.TeamID)
	}
}

// TestBootExplicitParentOverridesAmbientOnNativeRegister verifies an explicit
// WithParentAgentID wins over an ambient context parent on native register,
// matching the Python/Node precedence (explicit config overrides ambient).
func TestBootExplicitParentOverridesAmbientOnNativeRegister(t *testing.T) {
	capClient, _, regs := ffi.NewCapturingClientWithRegistrations()

	origFactory := newFFIClient
	newFFIClient = func() *ffi.Client { return capClient }
	t.Cleanup(func() { newFFIClient = origFactory })

	ctx := ContextWithParentAgentID(context.Background(), "ambient-parent")
	_, err := Init(ctx,
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
		WithSelfAgentID("agent-007"),
		WithParentAgentID("explicit-parent"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(*regs) != 1 {
		t.Fatalf("native registrations = %d, want 1", len(*regs))
	}
	if got := (*regs)[0]; got.ParentAgentID != "explicit-parent" {
		t.Errorf("registration ParentAgentID = %q, want explicit-parent (explicit overrides ambient)", got.ParentAgentID)
	}
}

// TestRegisterAgentNilFFIClientIsNoOp guards the early return when no FFI client
// is present (defensive; boot only calls registerAgent on the ffi path).
func TestRegisterAgentNilFFIClientIsNoOp(t *testing.T) {
	t.Parallel()

	a := &Assembly{}
	a.registerAgent() // must not panic
}
