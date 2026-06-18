package assembly

import (
	"os/exec"
	"testing"
	"time"
)

// TestAssemblyWrapTools_DelegatesWithAssemblyPosture covers the
// (a *Assembly) WrapTools method, which threads the Assembly's own
// governance client and fail-closed posture into the free WrapTools.
func TestAssemblyWrapTools_DelegatesWithAssemblyPosture(t *testing.T) {
	t.Parallel()

	a := &Assembly{
		opts: runtimeOptions{
			failClosed:      true,
			enforcementMode: EnforcementModeEnforce,
		},
	}

	tools := []Tool{&countingTool{name: "calc"}, &countingTool{name: "search"}}
	wrapped := a.WrapTools(tools)
	if len(wrapped) != len(tools) {
		t.Fatalf("expected %d wrapped tools, got %d", len(tools), len(wrapped))
	}
	for i, w := range wrapped {
		if w.Name() != tools[i].(*countingTool).name {
			t.Fatalf("wrapped tool %d name = %q, want %q", i, w.Name(), tools[i].(*countingTool).name)
		}
	}
}

// TestAssemblyClose_NoManagedSidecarIsNil covers the no-managed-sidecar
// branch of Close — it simply clears state and returns nil.
func TestAssemblyClose_NoManagedSidecarIsNil(t *testing.T) {
	t.Parallel()

	a := &Assembly{}
	if err := a.Close(); err != nil {
		t.Fatalf("Close on a bare Assembly should be nil, got %v", err)
	}
}

// TestAssemblyClose_StopsManagedSidecar covers the managed-sidecar branch of
// Close: it calls Stop on the managed sidecar and clears the handle.
func TestAssemblyClose_StopsManagedSidecar(t *testing.T) {
	t.Parallel()

	bin := osTrueBinary(t)
	sc := NewSidecar(bin, "127.0.0.1:0")
	sc.cmd = exec.Command(bin)
	if err := sc.cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	sc.stopTimeout = 2 * time.Second

	a := &Assembly{managedSidecar: sc}
	if err := a.Close(); err != nil {
		t.Fatalf("Close should stop the managed sidecar cleanly, got %v", err)
	}
	if a.managedSidecar != nil {
		t.Fatal("expected managedSidecar to be cleared after Close")
	}
}

// TestSidecarStop_SignalErrorOnReapedProcess covers Stop's signal-error
// branch: a process that has already exited and been waited on cannot be
// signalled, so Stop must surface a wrapped error rather than hang.
func TestSidecarStop_SignalErrorOnReapedProcess(t *testing.T) {
	t.Parallel()

	bin := osTrueBinary(t)
	sc := NewSidecar(bin, "127.0.0.1:0")
	sc.cmd = exec.Command(bin)
	if err := sc.cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	// Reap the process so its handle is finished; signalling it then fails.
	_ = sc.cmd.Wait()

	err := sc.Stop()
	if err == nil {
		t.Fatal("expected Stop to surface a signal error for a reaped process")
	}
}

func TestDefaultFindAasmOnPath_ReturnsEmptyWhenAbsent(t *testing.T) {
	// Constrain PATH to an empty temp dir so the lookup of "aasm" fails and
	// the "" not-found branch is exercised deterministically.
	t.Setenv("PATH", t.TempDir())
	if got := defaultFindAasmOnPath(); got != "" {
		t.Fatalf("expected empty path when aasm is absent, got %q", got)
	}
}

func TestDefaultFindAasmOnPath_ResolvesWhenPresent(t *testing.T) {
	// Drop an executable named "aasm" on a temp PATH and confirm the lookup
	// resolves it (the found branch).
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	makeFakeAasm(t, dir)
	got := defaultFindAasmOnPath()
	if got == "" {
		t.Fatal("expected defaultFindAasmOnPath to resolve a fake aasm on PATH")
	}
}
