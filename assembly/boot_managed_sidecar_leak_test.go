package assembly

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

// TestBoot_StopsManagedSidecarOnFallthroughError is the AAASM-4789 regression:
// once a managed sidecar subprocess is running, ANY error boot returns after
// that point must stop it before returning. Before the fix, a fallthrough
// failure from the sidecarConnector (the path boot takes once the FFI Connect
// attempt declines) returned early without calling Stop — and because Init
// discards the *Assembly on error, no caller is left holding a handle able to
// kill the subprocess, leaking it.
func TestBoot_StopsManagedSidecarOnFallthroughError(t *testing.T) {
	bin := osTrueBinary(t)
	sc := NewSidecar(bin, "127.0.0.1:0")
	sc.cmd = exec.Command(bin)
	if err := sc.cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	sc.stopTimeout = 2 * time.Second
	proc := sc.cmd.Process

	wantErr := errors.New("boom: no local sidecar")
	a := &Assembly{
		opts: runtimeOptions{
			gatewayURL: "https://gateway.example.com",
		},
		managedSidecar: sc,
		sidecarConnector: func(context.Context, string) (SidecarClient, error) {
			return nil, wantErr
		},
	}

	err := a.boot(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("boot() error = %v, want %v", err, wantErr)
	}
	if a.managedSidecar != nil {
		t.Fatal("expected boot to clear managedSidecar after stopping it on error")
	}

	// A second Wait on the same *os.Process only succeeds if the first Wait
	// (run inside Stop) never reaped it — i.e. boot never actually stopped
	// the subprocess. Erroring here is the proof the process is gone.
	if _, err := proc.Wait(); err == nil {
		t.Fatal("expected the managed sidecar subprocess to already be reaped by boot's Stop() call, but it was still waitable")
	}
}
