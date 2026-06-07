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

	type evt struct{ eventType, details string }
	sent := []evt{
		{"register", `{"parent":"a"}`},
		{"tool_call", `{"tool":"calc"}`},
	}
	for _, e := range sent {
		if err := client.SendEvent(e.eventType, e.details); err != nil {
			t.Fatalf("SendEvent(%q,%q) failed: %v", e.eventType, e.details, err)
		}
	}

	if len(*events) != len(sent) {
		t.Fatalf("expected %d events captured, got %d", len(sent), len(*events))
	}
	for i, e := range sent {
		if (*events)[i] != e.details {
			t.Errorf("events[%d] = %q, want %q", i, (*events)[i], e.details)
		}
	}
}

func TestCapturingClientDisconnect(t *testing.T) {
	t.Parallel()

	client, _ := NewCapturingClient()

	if err := client.Connect("unix:///tmp/aa-cap-test2.sock"); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if err := client.Disconnect(); err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}
}
