package assembly

import "encoding/json"

// MarshalAuditEvent encodes an AuditEvent to its canonical JSON wire
// representation.
//
// The wire shape uses snake_case field names (event_id, agent_id,
// action_type, decision, trace_id, span_id, parent_span_id, labels,
// call_stack; latency_ms / children inside each CallStackNode) — see
// the `json:` struct tags on AuditEvent and CallStackNode in
// audit.go for the exact mapping. Optional/recursive fields use
// `omitempty`, so a nil CallStack, nil Labels, or empty TraceID /
// SpanID / ParentSpanID encode to absent on the wire (proto3-default
// semantics on the canonical proto side).
//
// This is the producer-side complement to UnmarshalAuditEvent and
// matches the FFI shim's `eventJSON string` convention in
// `internal/ffi/`.
func MarshalAuditEvent(ev *AuditEvent) ([]byte, error) {
	return json.Marshal(ev)
}
