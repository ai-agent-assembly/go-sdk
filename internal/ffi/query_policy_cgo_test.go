//go:build cgo && aa_ffi_go

package ffi

import "testing"

// Exercises the real native aa_query_policy through the cgo binding. Without a
// live runtime the query cannot obtain a decision, so the native shim surfaces a
// non-OK status which QueryPolicy returns as an error (AAASM-3920) — proving the
// boundary marshals safely and fails closed rather than silently allowing. The
// caller's tool wrapper then applies its fail-open / fail-closed posture.
func TestCgoQueryPolicyFailsClosedWithoutRuntime(t *testing.T) {
	if !NativeBindingEnabled() {
		t.Skip("native aa_ffi_go binding not enabled")
	}

	client := NewDefaultClient()
	if err := client.Connect("/tmp/aa-ffi-go-query-policy.sock", "", ""); err != nil {
		t.Fatalf("connect over native binding failed: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	decision, _, err := client.QueryPolicy("agent-1", "tool_call", "web_search", `{"q":"x"}`)
	if err == nil {
		t.Fatal("expected an error from native query without a runtime (fail-closed), got nil")
	}
	if decision != DecisionAllow {
		t.Fatalf("expected DecisionAllow placeholder on error, got %d", decision)
	}
}
