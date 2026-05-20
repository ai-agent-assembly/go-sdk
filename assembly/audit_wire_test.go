package assembly

import (
	"encoding/json"
	"reflect"
	"strings"
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

func TestMarshalAuditEvent_WireKeysAreSnakeCase(t *testing.T) {
	event := &AuditEvent{
		EventID:      "evt-1",
		AgentID:      "support-agent",
		ActionType:   "llm_call",
		Decision:     "allow",
		TraceID:      "trace-1",
		SpanID:       "span-1",
		ParentSpanID: "span-0",
		Labels:       map[string]string{"env": "prod"},
		CallStack: []*CallStackNode{
			{ID: "n0", Kind: CallStackNodeKindLLM, Label: "gpt-4o", LatencyMs: 300},
		},
	}

	wire, err := MarshalAuditEvent(event)
	if err != nil {
		t.Fatalf("MarshalAuditEvent: %v", err)
	}
	wireStr := string(wire)

	// snake_case keys must appear on the wire; PascalCase Go field
	// names must not.
	wantSnakeCase := []string{
		`"event_id"`,
		`"agent_id"`,
		`"action_type"`,
		`"decision"`,
		`"trace_id"`,
		`"span_id"`,
		`"parent_span_id"`,
		`"labels"`,
		`"call_stack"`,
		`"latency_ms"`,
	}
	for _, key := range wantSnakeCase {
		if !strings.Contains(wireStr, key) {
			t.Errorf("missing snake_case key %s in wire JSON: %s", key, wireStr)
		}
	}
	notWantPascalCase := []string{
		`"EventID"`,
		`"AgentID"`,
		`"ActionType"`,
		`"TraceID"`,
		`"SpanID"`,
		`"ParentSpanID"`,
		`"CallStack"`,
		`"LatencyMs"`,
	}
	for _, key := range notWantPascalCase {
		if strings.Contains(wireStr, key) {
			t.Errorf("leaked PascalCase key %s into wire JSON: %s", key, wireStr)
		}
	}

	// Decoder accepts the snake_case wire keys into the PascalCase Go
	// fields — i.e. the json tag is what drives translation, not the
	// field name.
	decoded := &AuditEvent{}
	if err := json.Unmarshal(wire, decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded.EventID != "evt-1" {
		t.Errorf("EventID after snake_case decode: got %q, want %q", decoded.EventID, "evt-1")
	}
	if len(decoded.CallStack) != 1 {
		t.Fatalf("CallStack len after snake_case decode: got %d, want 1", len(decoded.CallStack))
	}
	if decoded.CallStack[0].LatencyMs != 300 {
		t.Errorf("CallStack[0].LatencyMs after snake_case decode: got %d, want 300",
			decoded.CallStack[0].LatencyMs)
	}
}
