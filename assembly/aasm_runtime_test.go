// Unit tests for assembly/aasm_runtime.go (AAASM-1229 / F115).
//
// Covers the four scenarios from the AAASM-1230 AC checklist:
//   - binary-in-PATH (TestFindAasmBinaryHitsPath)
//   - binary-bundled (TestFindAasmBinaryHitsUserLocalBin — Go SDK has no
//     language-bundled binary; ~/.local/bin is the curl-installer-bundled
//     equivalent that maps onto the per-Story install matrix)
//   - binary-not-found (TestInitAssemblyReturnsErrBinaryNotFoundWhenMissing)
//   - already-running (TestIsRunningDetectsActiveListener — the orchestrator
//     idempotency is a structural early-return that this test exercises at
//     the primitive level)

package assembly

import (
	"os"
	"path/filepath"
	"testing"
)

// makeFakeAasm writes an executable `aasm` shim into dir and returns its path.
func makeFakeAasm(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, BinaryName)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake aasm: %v", err)
	}
	return path
}

// TestFindAasmBinaryHitsPath covers the binary-in-PATH scenario from the
// AAASM-1230 AC. exec.LookPath must return the shim first, ahead of every
// fallback location.
func TestFindAasmBinaryHitsPath(t *testing.T) {
	dir := t.TempDir()
	fake := makeFakeAasm(t, dir)
	t.Setenv("PATH", dir)
	t.Setenv("HOME", filepath.Join(dir, "no-such-home"))

	resolved, err := findAasmBinary()
	if err != nil {
		t.Fatalf("findAasmBinary returned error: %v", err)
	}
	if resolved != fake {
		t.Fatalf("findAasmBinary returned %q, want %q", resolved, fake)
	}
}

// TestFindAasmBinaryHitsUserLocalBin covers the bundled / curl-installer
// fallback location. The Go SDK has no language-specific bundled binary
// path (unlike Python's wheel or Node's optional dep), so ~/.local/bin is
// the equivalent "binary-bundled" slot in the AAASM-1230 AC matrix.
func TestFindAasmBinaryHitsUserLocalBin(t *testing.T) {
	homeDir := t.TempDir()
	localBin := filepath.Join(homeDir, UserLocalBin)
	if err := os.MkdirAll(localBin, 0o755); err != nil {
		t.Fatalf("mkdir ~/.local/bin: %v", err)
	}
	fake := makeFakeAasm(t, localBin)
	t.Setenv("PATH", filepath.Join(homeDir, "no-such-path"))
	t.Setenv("HOME", homeDir)

	resolved, err := findAasmBinary()
	if err != nil {
		t.Fatalf("findAasmBinary returned error: %v", err)
	}
	if resolved != fake {
		t.Fatalf("findAasmBinary returned %q, want %q", resolved, fake)
	}
}
