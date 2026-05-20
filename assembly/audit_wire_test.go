package assembly

import (
	"reflect"
	"testing"
)

func TestMarshalUnmarshalAuditEvent_ThreeLevelCallStack(t *testing.T) {
	original := &AuditEvent{
		EventID:      "evt-1",
		AgentID:      "support-agent",
		ActionType:   "llm_call",
		Decision:     "allow",
		TraceID:      "trace-1",
		SpanID:       "span-1",
		ParentSpanID: "",
		Labels:       map[string]string{"env": "prod", "team": "support"},
		CallStack: []*CallStackNode{
			{
				ID:        "n0",
				Kind:      CallStackNodeKindLLM,
				Label:     "gpt-4o",
				LatencyMs: 300,
				Children: []*CallStackNode{
					{
						ID:        "n1",
						Kind:      CallStackNodeKindTool,
						Label:     "gmail.send",
						LatencyMs: 120,
						Children: []*CallStackNode{
							{ID: "n2", Kind: CallStackNodeKindResult, Label: "200 OK"},
						},
					},
				},
			},
		},
	}

	wire, err := MarshalAuditEvent(original)
	if err != nil {
		t.Fatalf("MarshalAuditEvent: %v", err)
	}
	if len(wire) == 0 {
		t.Fatal("MarshalAuditEvent returned empty bytes")
	}

	decoded, err := UnmarshalAuditEvent(wire)
	if err != nil {
		t.Fatalf("UnmarshalAuditEvent: %v", err)
	}
	if decoded == nil {
		t.Fatal("UnmarshalAuditEvent returned nil *AuditEvent")
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip mismatch\noriginal: %+v\ndecoded:  %+v", original, decoded)
	}
}

func TestUnmarshalAuditEvent_LegacyPayloadNoCallStack(t *testing.T) {
	// Payload predating AAASM-1419: no call_stack field on the wire.
	// Also omits labels to exercise the symmetric nil-map case.
	legacy := []byte(`{
		"event_id":   "evt-legacy",
		"agent_id":   "support-agent",
		"action_type":"llm_call",
		"decision":   "allow"
	}`)

	decoded, err := UnmarshalAuditEvent(legacy)
	if err != nil {
		t.Fatalf("UnmarshalAuditEvent: %v", err)
	}
	if decoded == nil {
		t.Fatal("UnmarshalAuditEvent returned nil *AuditEvent")
	}
	if decoded.CallStack != nil {
		t.Errorf("CallStack: got %v, want nil for legacy payload", decoded.CallStack)
	}
	if decoded.Labels != nil {
		t.Errorf("Labels: got %v, want nil for legacy payload", decoded.Labels)
	}
	if decoded.EventID != "evt-legacy" {
		t.Errorf("EventID: got %q, want %q", decoded.EventID, "evt-legacy")
	}
}
