package assembly

import "encoding/json"

// buildRegistrationEvent marshals topology fields from opts into the JSON
// payload sent to the sidecar as the first event after connect. The sidecar
// forwards this as a RegisterRequest to the gateway.
func buildRegistrationEvent(opts runtimeOptions) string {
	type event struct {
		EventType        string          `json:"event_type"`
		ParentAgentID    string          `json:"parent_agent_id,omitempty"`
		TeamID           string          `json:"team_id,omitempty"`
		DelegationReason string          `json:"delegation_reason,omitempty"`
		SpawnedByTool    string          `json:"spawned_by_tool,omitempty"`
		EnforcementMode  EnforcementMode `json:"enforcement_mode,omitempty"`
	}

	payload, err := json.Marshal(event{
		EventType:        "register",
		ParentAgentID:    opts.parentAgentID,
		TeamID:           opts.teamID,
		DelegationReason: opts.delegationReason,
		SpawnedByTool:    opts.spawnedByTool,
		EnforcementMode:  opts.enforcementMode,
	})
	if err != nil {
		return `{"event_type":"register"}`
	}
	return string(payload)
}
