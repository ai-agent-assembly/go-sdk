package assembly

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// writeHomeConfig points HOME at a temp dir and drops a ~/.aasm/config.yaml
// there so the resolver's config-file precedence step (step 3) can be
// exercised without touching the developer's real home directory.
func writeHomeConfig(t *testing.T, contents string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".aasm")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir .aasm: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestResolveGatewayURL_UsesConfigFileWhenNoExplicitOrEnv(t *testing.T) {
	t.Setenv(envGatewayURL, "")
	t.Setenv(legacyEnvGatewayURL, "")
	writeHomeConfig(t, "agent:\n  gateway_url: \"http://config.internal:7391\"\n")

	got, err := resolveGatewayURL(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://config.internal:7391" {
		t.Fatalf("expected gateway from config file, got %q", got)
	}
}

func TestResolveAPIKey_UsesConfigFileWhenNoExplicitOrEnv(t *testing.T) {
	t.Setenv(envAPIKey, "")
	t.Setenv(legacyEnvAPIKey, "")
	writeHomeConfig(t, "agent:\n  api_key: \"k-from-config\"\n")

	if got := resolveAPIKey(""); got != "k-from-config" {
		t.Fatalf("expected api key from config file, got %q", got)
	}
}

func TestResolveEnvWithLegacyFallback_UsesLegacyWhenCanonicalUnset(t *testing.T) {
	t.Setenv(envGatewayURL, "")
	t.Setenv(legacyEnvGatewayURL, "http://legacy:7391")

	once := &sync.Once{}
	got := resolveEnvWithLegacyFallback(envGatewayURL, legacyEnvGatewayURL, once)
	if got != "http://legacy:7391" {
		t.Fatalf("expected legacy env value, got %q", got)
	}
}

func TestResolveEnvWithLegacyFallback_CanonicalWins(t *testing.T) {
	t.Setenv(envGatewayURL, "http://canonical:7391")
	t.Setenv(legacyEnvGatewayURL, "http://legacy:7391")

	once := &sync.Once{}
	got := resolveEnvWithLegacyFallback(envGatewayURL, legacyEnvGatewayURL, once)
	if got != "http://canonical:7391" {
		t.Fatalf("expected canonical env value to win, got %q", got)
	}
}

func TestResolveEnvWithLegacyFallback_EmptyWhenNeitherSet(t *testing.T) {
	t.Setenv(envGatewayURL, "")
	t.Setenv(legacyEnvGatewayURL, "")

	once := &sync.Once{}
	if got := resolveEnvWithLegacyFallback(envGatewayURL, legacyEnvGatewayURL, once); got != "" {
		t.Fatalf("expected empty when neither var is set, got %q", got)
	}
}

func TestExpandHome_PassesThroughNonTilde(t *testing.T) {
	t.Parallel()

	if got := expandHome("/etc/aasm/config.yaml"); got != "/etc/aasm/config.yaml" {
		t.Fatalf("expected non-~ path unchanged, got %q", got)
	}
}

func TestExpandHome_ExpandsBareTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := expandHome("~"); got != home {
		t.Fatalf("expected bare ~ to expand to %q, got %q", home, got)
	}
}

func TestExpandHome_ExpandsTildeSlash(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got, want := expandHome("~/.aasm/config.yaml"), filepath.Join(home, ".aasm/config.yaml"); got != want {
		t.Fatalf("expected ~/ to expand to %q, got %q", want, got)
	}
}
