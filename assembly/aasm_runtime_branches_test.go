package assembly

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStartRuntime_SpawnsDetachedProcess covers the happy path of
// startRuntime: it opens the runtime log in the working directory, spawns
// the binary, and returns a live *os.Process handle without waiting on it.
func TestStartRuntime_SpawnsDetachedProcess(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	bin := makeFakeAasm(t, dir)

	proc, err := startRuntime(bin, DefaultPort)
	if err != nil {
		t.Fatalf("startRuntime returned error: %v", err)
	}
	if proc == nil {
		t.Fatal("startRuntime returned a nil process handle")
	}
	// Reap the short-lived child so it does not linger as a zombie.
	_, _ = proc.Wait()

	// The runtime log must have been created in the working directory.
	if _, statErr := os.Stat(filepath.Join(dir, RuntimeLogFilename)); statErr != nil {
		t.Fatalf("expected runtime log %q to exist: %v", RuntimeLogFilename, statErr)
	}
}

// TestStartRuntime_ErrorWhenLogPathUnwritable covers the log-open failure
// branch: when the log path cannot be opened for writing, startRuntime
// surfaces the error and never spawns the binary.
func TestStartRuntime_ErrorWhenLogPathUnwritable(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	// Make the log path a directory so O_WRONLY|O_CREATE on it fails.
	if err := os.Mkdir(filepath.Join(dir, RuntimeLogFilename), 0o755); err != nil {
		t.Fatalf("mkdir log placeholder: %v", err)
	}

	proc, err := startRuntime("/bin/true", DefaultPort)
	if err == nil {
		if proc != nil {
			_, _ = proc.Wait()
		}
		t.Fatal("expected startRuntime to fail when the log path is a directory")
	}
	if proc != nil {
		t.Fatalf("expected nil process on log-open failure, got %+v", proc)
	}
}

// TestInitAssembly_StartsSidecarWhenFoundAndNotRunning covers the
// InitAssembly orchestration path where no sidecar is listening, the binary
// resolves from ~/.local/bin, and startRuntime succeeds — returning nil.
func TestInitAssembly_StartsSidecarWhenFoundAndNotRunning(t *testing.T) {
	if isRunning(DefaultPort) {
		t.Skipf("skipping: something is already listening on %s:%d", DefaultRuntimeHost, DefaultPort)
	}
	dir := t.TempDir()
	t.Chdir(dir)
	localBin := filepath.Join(dir, UserLocalBin)
	if err := os.MkdirAll(localBin, 0o755); err != nil {
		t.Fatalf("mkdir ~/.local/bin: %v", err)
	}
	makeFakeAasm(t, localBin)
	t.Setenv("PATH", filepath.Join(dir, "no-such-path"))
	t.Setenv("HOME", dir)

	if err := InitAssembly("agent-init"); err != nil {
		t.Fatalf("InitAssembly returned error: %v", err)
	}
}
