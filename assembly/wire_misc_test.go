package assembly

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalAuditEvent_ReturnsErrorOnInvalidJSON(t *testing.T) {
	t.Parallel()

	ev, err := UnmarshalAuditEvent([]byte("{not valid json"))
	if err == nil {
		t.Fatal("expected a decode error for malformed JSON")
	}
	if ev != nil {
		t.Fatalf("expected nil event on decode error, got %+v", ev)
	}
}

func TestBuildRegistrationEvent_OmitsEmptyTopologyFields(t *testing.T) {
	t.Parallel()

	got := buildRegistrationEvent(runtimeOptions{})

	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("registration event is not valid JSON: %v", err)
	}
	if decoded["event_type"] != "register" {
		t.Fatalf("expected event_type register, got %v", decoded["event_type"])
	}
	for _, omitted := range []string{"parent_agent_id", "team_id", "delegation_reason", "spawned_by_tool", "enforcement_mode"} {
		if _, present := decoded[omitted]; present {
			t.Fatalf("expected %q to be omitted when empty, but it was present", omitted)
		}
	}
}

func TestBuildRegistrationEvent_IncludesTopologyFields(t *testing.T) {
	t.Parallel()

	got := buildRegistrationEvent(runtimeOptions{
		parentAgentID:    "parent-1",
		teamID:           "team-a",
		delegationReason: "research subtask",
		spawnedByTool:    "spawn_agent",
		enforcementMode:  EnforcementModeObserve,
	})

	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("registration event is not valid JSON: %v", err)
	}
	want := map[string]string{
		"parent_agent_id":   "parent-1",
		"team_id":           "team-a",
		"delegation_reason": "research subtask",
		"spawned_by_tool":   "spawn_agent",
		"enforcement_mode":  "observe",
	}
	for k, v := range want {
		if decoded[k] != v {
			t.Fatalf("registration event %q = %v, want %q", k, decoded[k], v)
		}
	}
}
