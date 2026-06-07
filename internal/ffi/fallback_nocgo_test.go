//go:build !cgo

package ffi

import (
	"testing"
)

func TestDefaultBindingUsesFallbackWhenCGODisabled(t *testing.T) {
	t.Parallel()

	if _, ok := defaultBinding().(fallbackUDSBridge); !ok {
		t.Fatalf("expected fallbackUDSBridge, got %T", defaultBinding())
	}
}

func TestFallbackClientFlowWithoutCGO(t *testing.T) {
	t.Parallel()

	client := NewDefaultClient()
	if err := client.Connect("unix:///tmp/aa.sock"); err != nil {
		t.Fatalf("expected connect success, got %v", err)
	}

	if err := client.SendEvent("tool_call", `{"event":"x"}`); err != nil {
		t.Fatalf("expected send_event success, got %v", err)
	}

	if err := client.Disconnect(); err != nil {
		t.Fatalf("expected disconnect success, got %v", err)
	}
}
