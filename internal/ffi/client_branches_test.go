package ffi

import (
	"errors"
	"testing"
)

func TestClient_SendEventBeforeConnectReportsNotConnected(t *testing.T) {
	t.Parallel()

	client := NewClient(&mockBinding{})
	err := client.SendEvent("tool_call", `{"event":"x"}`)
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected before connect, got %v", err)
	}
}

func TestClient_DisconnectBeforeConnectReportsNotConnected(t *testing.T) {
	t.Parallel()

	client := NewClient(&mockBinding{})
	err := client.Disconnect()
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected before connect, got %v", err)
	}
}

func TestClient_SendEventBindingUnavailable(t *testing.T) {
	t.Parallel()

	client := NewClient(nil)
	if err := client.SendEvent("t", "d"); !errors.Is(err, ErrBindingUnavailable) {
		t.Fatalf("expected ErrBindingUnavailable, got %v", err)
	}
}

func TestClient_DisconnectBindingUnavailable(t *testing.T) {
	t.Parallel()

	client := NewClient(nil)
	if err := client.Disconnect(); !errors.Is(err, ErrBindingUnavailable) {
		t.Fatalf("expected ErrBindingUnavailable, got %v", err)
	}
}

// TestNewDefaultClient_HasBinding confirms the build-selected constructor wires
// a non-nil binding so the public entrypoint works in both build modes.
func TestNewDefaultClient_HasBinding(t *testing.T) {
	t.Parallel()

	client := NewDefaultClient()
	if client == nil {
		t.Fatal("expected non-nil client from NewDefaultClient")
	}
	if client.binding == nil {
		t.Fatal("expected NewDefaultClient to install a build-selected binding")
	}
}
