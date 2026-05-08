package ffi

import (
	"testing"
)

func TestCapturingClientRecordsSendEventPayloads(t *testing.T) {
	t.Parallel()

	client, events := NewCapturingClient()

	if err := client.Connect("unix:///tmp/aa-cap-test.sock"); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	payloads := []string{
		`{"event_type":"register"}`,
		`{"event_type":"tool_call","tool":"calc"}`,
	}
	for _, p := range payloads {
		if err := client.SendEvent(p); err != nil {
			t.Fatalf("SendEvent(%q) failed: %v", p, err)
		}
	}

	if len(*events) != len(payloads) {
		t.Fatalf("expected %d events captured, got %d", len(payloads), len(*events))
	}
	for i, want := range payloads {
		if (*events)[i] != want {
			t.Errorf("events[%d] = %q, want %q", i, (*events)[i], want)
		}
	}
}

func TestCapturingClientQueryPolicyAndDisconnect(t *testing.T) {
	t.Parallel()

	client, _ := NewCapturingClient()

	if err := client.Connect("unix:///tmp/aa-cap-test2.sock"); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	resp, err := client.QueryPolicy(`{"tool":"calculator"}`)
	if err != nil {
		t.Fatalf("QueryPolicy failed: %v", err)
	}
	if resp == "" {
		t.Fatal("expected non-empty QueryPolicy response")
	}

	if err := client.Disconnect(); err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}
}
