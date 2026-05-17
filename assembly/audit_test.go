package assembly

import (
	"testing"
)

func TestCallStackNode(t *testing.T) {
	t.Run("required fields with optional zero defaults", func(t *testing.T) {
		node := CallStackNode{
			ID:    "n0",
			Kind:  CallStackNodeKindLLM,
			Label: "gpt-4o",
		}
		if node.ID != "n0" {
			t.Errorf("ID: got %q, want %q", node.ID, "n0")
		}
		if node.Kind != CallStackNodeKindLLM {
			t.Errorf("Kind: got %q, want %q", node.Kind, CallStackNodeKindLLM)
		}
		if node.LatencyMs != 0 {
			t.Errorf("LatencyMs default: got %d, want 0", node.LatencyMs)
		}
		if node.Children != nil {
			t.Errorf("Children default: got %v, want nil", node.Children)
		}
	})

	t.Run("with latency and one-level children", func(t *testing.T) {
		child := &CallStackNode{
			ID:        "n1",
			Kind:      CallStackNodeKindTool,
			Label:     "gmail.send",
			LatencyMs: 120,
		}
		parent := CallStackNode{
			ID:        "n0",
			Kind:      CallStackNodeKindLLM,
			Label:     "gpt-4o",
			LatencyMs: 300,
			Children:  []*CallStackNode{child},
		}
		if len(parent.Children) != 1 {
			t.Fatalf("Children len: got %d, want 1", len(parent.Children))
		}
		if parent.Children[0] != child {
			t.Errorf("Children[0]: got %p, want %p", parent.Children[0], child)
		}
		if parent.Children[0].LatencyMs != 120 {
			t.Errorf("Children[0].LatencyMs: got %d, want 120", parent.Children[0].LatencyMs)
		}
	})

	t.Run("three-level LLM tool result tree", func(t *testing.T) {
		tree := CallStackNode{
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
		}
		leaf := tree.Children[0].Children[0]
		if leaf.Kind != CallStackNodeKindResult {
			t.Errorf("leaf Kind: got %q, want %q", leaf.Kind, CallStackNodeKindResult)
		}
		if leaf.LatencyMs != 0 {
			t.Errorf("leaf LatencyMs: got %d, want 0 (not recorded)", leaf.LatencyMs)
		}
	})

	t.Run("kind constants stringify to wire values", func(t *testing.T) {
		cases := map[CallStackNodeKind]string{
			CallStackNodeKindLLM:    "llm",
			CallStackNodeKindTool:   "tool",
			CallStackNodeKindResult: "result",
		}
		for kind, want := range cases {
			if string(kind) != want {
				t.Errorf("%v: got %q, want %q", kind, string(kind), want)
			}
		}
	})
}

func TestAuditEvent(t *testing.T) {
	t.Run("minimal construction leaves CallStack nil", func(t *testing.T) {
		event := AuditEvent{
			EventID:    "evt-1",
			AgentID:    "support-agent",
			ActionType: "llm_call",
			Decision:   "allow",
		}
		if event.EventID != "evt-1" {
			t.Errorf("EventID: got %q, want %q", event.EventID, "evt-1")
		}
		if event.CallStack != nil {
			t.Errorf("CallStack default: got %v, want nil", event.CallStack)
		}
		if event.Labels != nil {
			t.Errorf("Labels default: got %v, want nil", event.Labels)
		}
		if event.TraceID != "" {
			t.Errorf("TraceID default: got %q, want \"\"", event.TraceID)
		}
	})

	t.Run("populated with tracing and call stack tree", func(t *testing.T) {
		event := AuditEvent{
			EventID:    "evt-1",
			AgentID:    "support-agent",
			ActionType: "llm_call",
			Decision:   "allow",
			TraceID:    "trace-1",
			SpanID:     "span-1",
			Labels:     map[string]string{"env": "prod"},
			CallStack: []*CallStackNode{
				{
					ID:        "n0",
					Kind:      CallStackNodeKindLLM,
					Label:     "gpt-4o",
					LatencyMs: 300,
					Children: []*CallStackNode{
						{ID: "n1", Kind: CallStackNodeKindTool, Label: "gmail.send", LatencyMs: 120},
					},
				},
			},
		}
		if len(event.CallStack) != 1 {
			t.Fatalf("CallStack len: got %d, want 1", len(event.CallStack))
		}
		root := event.CallStack[0]
		if root.Kind != CallStackNodeKindLLM {
			t.Errorf("root Kind: got %q, want %q", root.Kind, CallStackNodeKindLLM)
		}
		if root.Children[0].Label != "gmail.send" {
			t.Errorf("Children[0].Label: got %q, want %q", root.Children[0].Label, "gmail.send")
		}
		if event.Labels["env"] != "prod" {
			t.Errorf("Labels[env]: got %q, want %q", event.Labels["env"], "prod")
		}
	})
}
