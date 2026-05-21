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
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
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
