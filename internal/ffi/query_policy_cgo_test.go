//go:build cgo && aa_ffi_go

package ffi

import "testing"

// Exercises the real native aa_query_policy through the cgo binding. Without a
// live runtime the query fails open inside the native shim, which returns
// AA_STATUS_OK with AA_DECISION_ALLOW — proving the boundary marshals safely and
// preserves the fail-open contract.
func TestCgoQueryPolicyFailsOpenWithoutRuntime(t *testing.T) {
	if !NativeBindingEnabled() {
		t.Skip("native aa_ffi_go binding not enabled")
	}

	client := NewDefaultClient()
	if err := client.Connect("/tmp/aa-ffi-go-query-policy.sock", "", ""); err != nil {
		t.Fatalf("connect over native binding failed: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	decision, _, err := client.QueryPolicy("agent-1", "tool_call", "web_search", `{"q":"x"}`)
	if err != nil {
		t.Fatalf("expected fail-open nil error from native query, got %v", err)
	}
	if decision != DecisionAllow {
		t.Fatalf("expected DecisionAllow on fail-open, got %d", decision)
	}
}
