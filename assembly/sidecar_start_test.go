package assembly

import (
	"context"
	"os"
	"testing"
)

// TestSidecarStart_LaunchesRealSubprocess exercises Sidecar.Start's own
// exec.CommandContext path (not a pre-seeded cmd). It points the binary at a
// short-lived OS command that ignores the injected "--listen <addr>" args and
// exits, which is enough to drive the real spawn branch.
func TestSidecarStart_LaunchesRealSubprocess(t *testing.T) {
	t.Parallel()

	bin := osTrueBinary(t)
	sc := NewSidecar(bin, "127.0.0.1:0")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := sc.Start(ctx); err != nil {
		t.Fatalf("Start: unexpected error: %v", err)
	}
	if sc.cmd == nil || sc.cmd.Process == nil {
		t.Fatal("expected a live process after Start")
	}
	// Starting again must report the already-started guard.
	if err := sc.Start(ctx); err == nil {
		t.Fatal("expected error starting an already-started sidecar")
	}
	t.Cleanup(func() {
		if sc.cmd != nil && sc.cmd.Process != nil {
			_ = sc.cmd.Process.Kill()
			_, _ = sc.cmd.Process.Wait()
		}
	})
}

// osTrueBinary returns the path to a harmless, always-available executable
// that accepts and ignores extra arguments. Skips the test if none is found.
func osTrueBinary(t *testing.T) string {
	t.Helper()
	for _, cand := range []string{"/usr/bin/true", "/bin/true", "/usr/bin/sleep", "/bin/sleep"} {
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	t.Skip("no harmless OS binary available to drive the real spawn path")
	return ""
}

// TestSidecarStart_FailsForMissingBinary covers the start-error wrap branch.
func TestSidecarStart_FailsForMissingBinary(t *testing.T) {
	t.Parallel()

	sc := NewSidecar("/nonexistent/aa-sidecar-binary", "127.0.0.1:0")
	err := sc.Start(context.Background())
	if err == nil {
		t.Fatal("expected Start to fail for a missing binary")
	}
}

// TestIgnoreSignalExit_PassesThroughNonExitError covers the non-ExitError
// branch of ignoreSignalExit (a genuine error is returned unchanged).
func TestIgnoreSignalExit_PassesThroughNonExitError(t *testing.T) {
	t.Parallel()

	sentinel := context.Canceled
	if got := ignoreSignalExit(sentinel); got != sentinel {
		t.Fatalf("expected non-ExitError to pass through, got %v", got)
	}
	if got := ignoreSignalExit(nil); got != nil {
		t.Fatalf("expected nil to pass through, got %v", got)
	}
}
