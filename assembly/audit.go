package assembly

// AuditEvent + CallStackNode mirror the wire-protocol shape of
// `assembly.audit.v1.AuditEvent` and `assembly.audit.v1.CallStackNode`
// as of agent-assembly commit
// `ed4aa11a8c1d1ce1e6f96b08cf2179fd772099b2` (AAASM-1419 / PR #467).
//
// The Go SDK is a pure-Go module (no protoc, no buf) and exchanges
// events with the Rust runtime over a JSON wire format via the FFI
// shim in `internal/ffi/`. These structs are handwritten to track the
// canonical proto shape; field naming uses Go PascalCase to match
// `policy_model.go`'s existing convention.

// CallStackNodeKind identifies the category of a CallStackNode.
//
// String-typed (not an enum) to keep this open-ended on the wire — the
// dashboard renders three known values today, exposed as named
// constants below.
type CallStackNodeKind string

const (
	// CallStackNodeKindLLM denotes an LLM-call step in the stack.
	CallStackNodeKindLLM CallStackNodeKind = "llm"
	// CallStackNodeKindTool denotes a tool / MCP-call step in the stack.
	CallStackNodeKindTool CallStackNodeKind = "tool"
	// CallStackNodeKindResult denotes a result / completion step in the stack.
	CallStackNodeKindResult CallStackNodeKind = "result"
)

// CallStackNode is one node in the hierarchical call stack attached to
// an AuditEvent. Renders inline beneath an expanded Live Ops row in
// the dashboard.
//
// JSON field naming is snake_case to match the proto/JSON wire shape
// (`assembly.audit.v1.CallStackNode`); see MarshalAuditEvent /
// UnmarshalAuditEvent in audit_wire.go for the serialization helpers.
type CallStackNode struct {
	// ID is a stable identifier for this node within the call stack.
	ID string `json:"id"`
	// Kind is the node category — typically one of CallStackNodeKindLLM,
	// CallStackNodeKindTool, or CallStackNodeKindResult.
	Kind CallStackNodeKind `json:"kind"`
	// Label is the human-readable label rendered by downstream UI.
	Label string `json:"label"`
	// LatencyMs is the step-local latency in milliseconds. `0` means
	// the producer did not record a duration (proto3 default semantics).
	LatencyMs int64 `json:"latency_ms"`
	// Children is the recursive descent — nested calls produced by
	// this step. Nil or empty slice when the node has no children.
	Children []*CallStackNode `json:"children,omitempty"`
}

// AuditEvent records a governance-relevant occurrence in the gateway
// audit trail.
//
// Focused subset of the proto `assembly.audit.v1.AuditEvent` message —
// exposes the scalar identifying fields, labels, and the new
// CallStack field added in AAASM-1419. The proto's `detail` oneof
// (LLM / tool / file-op / network / process / violation / approval
// variants) and the full lineage block are intentionally out of scope;
// they'll be filed as separate follow-up Tasks if a Go consumer needs
// them.
type AuditEvent struct {
	// EventID is the unique identifier for this audit record (UUID v7).
	EventID string `json:"event_id"`
	// AgentID is the identity string of the agent that produced the event.
	AgentID string `json:"agent_id"`
	// ActionType is the high-level action category — e.g. "llm_call",
	// "tool_call", "file_op". Open-ended on the wire.
	ActionType string `json:"action_type"`
	// Decision is the policy engine verdict — e.g. "allow", "deny",
	// "redact". Open-ended on the wire.
	Decision string `json:"decision"`
	// TraceID is the distributed tracing run-level identifier. Empty
	// when unset.
	TraceID string `json:"trace_id,omitempty"`
	// SpanID is the distributed tracing action-level identifier. Empty
	// when unset.
	SpanID string `json:"span_id,omitempty"`
	// ParentSpanID is the distributed tracing parent span identifier.
	// Empty when this is a root span.
	ParentSpanID string `json:"parent_span_id,omitempty"`
	// Labels are arbitrary key/value labels attached at event creation.
	Labels map[string]string `json:"labels,omitempty"`
	// CallStack is the hierarchical record of LLM / tool / result
	// steps that led to this event. Nil or empty slice when the
	// producer did not record a stack.
	CallStack []*CallStackNode `json:"call_stack,omitempty"`
}
