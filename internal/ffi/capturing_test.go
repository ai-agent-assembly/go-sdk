package ffi

import (
	"testing"
)

func TestCapturingClientRecordsSendEventPayloads(t *testing.T) {
	t.Parallel()

	client, events := NewCapturingClient()

	if err := client.Connect("unix:///tmp/aa-cap-test.sock", "", ""); err != nil {
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

func TestConnectForwardsAgentIDAndSDKVersion(t *testing.T) {
	t.Parallel()

	// AAASM-3683: Client.Connect must forward the agent id and the Go-module SDK
	// version down to the binding so they are signed into the runtime handshake.
	b := &capturingBinding{}
	client := NewClient(b)

	if err := client.Connect("unix:///tmp/aa-version-test.sock", "agent-7", "go-9.8.7"); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if b.ConnectAgentID != "agent-7" {
		t.Errorf("ConnectAgentID = %q, want %q", b.ConnectAgentID, "agent-7")
	}
	if b.ConnectSDKVersion != "go-9.8.7" {
		t.Errorf("ConnectSDKVersion = %q, want %q", b.ConnectSDKVersion, "go-9.8.7")
	}
}

func TestCapturingClientDisconnect(t *testing.T) {
	t.Parallel()

	client, _ := NewCapturingClient()

	if err := client.Connect("unix:///tmp/aa-cap-test2.sock", "", ""); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if err := client.Disconnect(); err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}
}
