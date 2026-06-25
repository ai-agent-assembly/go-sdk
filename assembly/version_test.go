package assembly

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVersionMatchesVERSIONFile guards against the version markers drifting
// apart again (AAASM-3731): the user-surfaced Version constant — which boot
// signs into the runtime handshake — must agree with the repo-root VERSION
// file that release tooling reads.
func TestVersionMatchesVERSIONFile(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "VERSION"))
	if err != nil {
		t.Fatalf("read VERSION file: %v", err)
	}
	want := strings.TrimSpace(string(raw))
	if Version != want {
		t.Errorf("Version = %q, want %q (VERSION file); version markers must stay in sync", Version, want)
	}
}
