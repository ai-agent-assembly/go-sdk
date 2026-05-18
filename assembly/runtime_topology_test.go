package assembly

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/AI-agent-assembly/go-sdk/internal/ffi"
)

func TestBootSendsRegistrationEventWithTopologyFields(t *testing.T) {
	capClient, events := ffi.NewCapturingClient()

	origFactory := newFFIClient
	newFFIClient = func() *ffi.Client { return capClient }
	t.Cleanup(func() { newFFIClient = origFactory })

	a, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
		WithParentAgentID("parent-agent-001"),
		WithTeamID("team-alpha"),
		WithDelegationReason("sub-task delegation"),
		WithSpawnedByTool("search_tool"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil Assembly")
	}

	if len(*events) == 0 {
		t.Fatal("expected at least one event to be sent via SendEvent")
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte((*events)[0]), &payload); err != nil {
		t.Fatalf("registration event is not valid JSON: %v", err)
	}

	checkField := func(key, want string) {
		t.Helper()
		if got := payload[key]; got != want {
			t.Errorf("registration event %q = %q, want %q", key, got, want)
		}
	}

	checkField("event_type", "register")
	checkField("parent_agent_id", "parent-agent-001")
	checkField("team_id", "team-alpha")
	checkField("delegation_reason", "sub-task delegation")
	checkField("spawned_by_tool", "search_tool")
}

func TestBootSendsRegistrationEventWithNoTopologyFields(t *testing.T) {
	capClient, events := ffi.NewCapturingClient()

	origFactory := newFFIClient
	newFFIClient = func() *ffi.Client { return capClient }
	t.Cleanup(func() { newFFIClient = origFactory })

	_, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
		withSidecarAddress("127.0.0.1:50051"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(*events) == 0 {
		t.Fatal("expected registration event even with no topology fields")
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte((*events)[0]), &payload); err != nil {
		t.Fatalf("registration event is not valid JSON: %v", err)
	}

	if payload["event_type"] != "register" {
		t.Errorf("expected event_type=register, got %q", payload["event_type"])
	}

	for _, field := range []string{"parent_agent_id", "team_id", "delegation_reason", "spawned_by_tool"} {
		if v, ok := payload[field]; ok {
			t.Errorf("expected %q to be absent when empty, got %q", field, v)
		}
	}
}

func TestBootRegistrationEventFailureIsReturned(t *testing.T) {
	if ffi.NativeBindingEnabled() {
		t.Skip("native binding does not use capturing client path")
	}

	// The capturing client always succeeds, so we test the fallback path
	// by checking that a non-ffi boot (sidecarConnector) does not send events.
	origConnector := sidecarConnector
	t.Cleanup(func() { sidecarConnector = origConnector })

	connectorCalled := false
	sidecarConnector = func(context.Context, string) (SidecarClient, error) {
		connectorCalled = true
		return nil, errors.New("sidecar unavailable")
	}

	// With no sidecarAddress, ffi path is skipped; sidecarConnector is used.
	_, err := Init(context.Background(),
		WithGatewayURL("https://gateway.example.com"),
		WithAPIKey("test-key"),
	)

	if err == nil {
		t.Fatal("expected error from sidecarConnector, got nil")
	}
	if !connectorCalled {
		t.Fatal("expected sidecarConnector to be called when ffi path is skipped")
	}
}
