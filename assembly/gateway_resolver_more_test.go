package assembly

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// restoreResolverSeams snapshots the package-level resolver seams and
// restores them when the test finishes so seam overrides never leak.
func restoreResolverSeams(t *testing.T) {
	t.Helper()
	original := gatewayResolverSeams
	t.Cleanup(func() { gatewayResolverSeams = original })
}

// TestWaitForHealthz_FalseWhenNeverReady covers the timeout path: every
// probe fails, the poll loop exhausts its deadline, and the final probe
// also fails — yielding false.
func TestWaitForHealthz_FalseWhenNeverReady(t *testing.T) {
	t.Parallel()

	// 127.0.0.1:1 is never listened on in CI, so every probe fails fast.
	if waitForHealthz(context.Background(), "http://127.0.0.1:1", 60*time.Millisecond, 10*time.Millisecond) {
		t.Fatal("expected waitForHealthz to return false when the gateway never becomes ready")
	}
}

// TestExpandHome_TildeUserUnchanged covers the "~user" branch: a tilde that
// is not a bare "~" or "~/" prefix is returned unchanged (Go does not expand
// other users' home directories here).
func TestExpandHome_TildeUserUnchanged(t *testing.T) {
	t.Parallel()

	if got := expandHome("~someuser/config.yaml"); got != "~someuser/config.yaml" {
		t.Fatalf("expected ~user path to be returned unchanged, got %q", got)
	}
}

// TestDefaultSpawnAasm_Success covers defaultSpawnAasm spawning a binary and
// releasing the child process handle without error.
func TestDefaultSpawnAasm_Success(t *testing.T) {
	dir := t.TempDir()
	bin := makeFakeAasm(t, dir)

	if err := defaultSpawnAasm(bin); err != nil {
		t.Fatalf("defaultSpawnAasm returned error: %v", err)
	}
}

// TestDefaultSpawnAasm_ErrorWhenBinaryMissing covers the Start failure
// branch: a non-existent binary path makes cmd.Start fail.
func TestDefaultSpawnAasm_ErrorWhenBinaryMissing(t *testing.T) {
	t.Parallel()

	if err := defaultSpawnAasm(filepath.Join(t.TempDir(), "definitely-not-here")); err == nil {
		t.Fatal("expected defaultSpawnAasm to fail for a missing binary")
	}
}

// TestAutoStartGateway_ConfigErrorWhenAasmMissing covers the branch where
// the aasm binary is not on PATH: autoStartGateway returns a
// *ConfigurationError without attempting to spawn.
func TestAutoStartGateway_ConfigErrorWhenAasmMissing(t *testing.T) {
	restoreResolverSeams(t)
	gatewayResolverSeams.findAasmOnPath = func() string { return "" }

	err := autoStartGateway(context.Background(), defaultGatewayURL, 50*time.Millisecond)
	var cfgErr *ConfigurationError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected *ConfigurationError when aasm is missing, got %v", err)
	}
}

// TestAutoStartGateway_ConfigErrorWhenSpawnFails covers the spawn-failure
// branch: a spawn error is wrapped as a *ConfigurationError.
func TestAutoStartGateway_ConfigErrorWhenSpawnFails(t *testing.T) {
	restoreResolverSeams(t)
	gatewayResolverSeams.findAasmOnPath = func() string { return "/usr/bin/aasm" }
	gatewayResolverSeams.spawnAasm = func(string) error { return errors.New("boom") }

	err := autoStartGateway(context.Background(), defaultGatewayURL, 50*time.Millisecond)
	var cfgErr *ConfigurationError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected *ConfigurationError when spawn fails, got %v", err)
	}
}

// TestAutoStartGateway_GatewayErrorWhenNotReady covers the readiness-timeout
// branch: the spawn succeeds but the gateway never answers /healthz, so a
// *GatewayError is returned.
func TestAutoStartGateway_GatewayErrorWhenNotReady(t *testing.T) {
	restoreResolverSeams(t)
	gatewayResolverSeams.findAasmOnPath = func() string { return "/usr/bin/aasm" }
	gatewayResolverSeams.spawnAasm = func(string) error { return nil }

	err := autoStartGateway(context.Background(), "http://127.0.0.1:1", 60*time.Millisecond)
	var gwErr *GatewayError
	if !errors.As(err, &gwErr) {
		t.Fatalf("expected *GatewayError when gateway never becomes ready, got %v", err)
	}
}

// TestAutoStartGateway_SuccessWhenHealthzReady covers the success path: the
// spawn succeeds and the gateway answers /healthz, so autoStartGateway
// returns nil.
func TestAutoStartGateway_SuccessWhenHealthzReady(t *testing.T) {
	restoreResolverSeams(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gatewayResolverSeams.findAasmOnPath = func() string { return "/usr/bin/aasm" }
	gatewayResolverSeams.spawnAasm = func(string) error { return nil }

	if err := autoStartGateway(context.Background(), srv.URL, time.Second); err != nil {
		t.Fatalf("expected autoStartGateway to succeed against a healthy gateway, got %v", err)
	}
}

// TestResolveGatewayURL_PropagatesAutoStartError covers resolveGatewayURL's
// terminal branch: no explicit URL, no env, no config file, the local
// default is unreachable, and auto-start fails — the error propagates.
func TestResolveGatewayURL_PropagatesAutoStartError(t *testing.T) {
	restoreResolverSeams(t)
	t.Setenv(envGatewayURL, "")
	t.Setenv(legacyEnvGatewayURL, "")
	// Point HOME at an empty dir so the config-file step finds nothing.
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := os.Stat(filepath.Join(home, ".aasm")); err == nil {
		t.Fatal("unexpected pre-existing .aasm in temp home")
	}
	gatewayResolverSeams.findAasmOnPath = func() string { return "" }
	if probeHealthz(context.Background(), defaultGatewayURL, defaultProbeTimeout) {
		t.Skipf("skipping: a gateway is already answering at %s", defaultGatewayURL)
	}

	if _, err := resolveGatewayURL(context.Background(), ""); err == nil {
		t.Fatal("expected resolveGatewayURL to propagate the auto-start error")
	}
}
