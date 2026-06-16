package assembly

import (
	"bytes"
	"context"
	"log"
	"strings"
	"sync"
	"testing"
)

// captureLog redirects the standard logger to a buffer for the duration
// of one test and restores the previous output and flags via t.Cleanup.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prevOut := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})
	return &buf
}

// TestResolveGatewayURL_EnvPrefix exercises the AA_*-canonical /
// AAASM_*-deprecated precedence for the gateway URL: AA_* wins when set,
// the legacy AAASM_* still resolves and warns exactly once, AA_* takes
// precedence when both are set (no warning), and neither leaves the
// resolution unchanged so the rest of the chain runs.
func TestResolveGatewayURL_EnvPrefix(t *testing.T) {
	tests := []struct {
		name      string
		canonical string
		legacy    string
		want      string
		wantWarn  bool
	}{
		{name: "canonical AA_ used when set", canonical: "http://aa:7391", legacy: "", want: "http://aa:7391", wantWarn: false},
		{name: "legacy AAASM_ resolves and warns", canonical: "", legacy: "http://aaasm:7391", want: "http://aaasm:7391", wantWarn: true},
		{name: "AA_ wins when both set", canonical: "http://aa:7391", legacy: "http://aaasm:7391", want: "http://aa:7391", wantWarn: false},
		{name: "neither set leaves resolution to next step", canonical: "", legacy: "", want: "", wantWarn: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset the per-variable once guard so each subtest can
			// observe the one-time deprecation notice independently.
			deprecationOnce.gatewayURL = sync.Once{}
			t.Setenv(envGatewayURL, tc.canonical)
			t.Setenv(legacyEnvGatewayURL, tc.legacy)
			buf := captureLog(t)

			got := resolveEnvWithLegacyFallback(envGatewayURL, legacyEnvGatewayURL, &deprecationOnce.gatewayURL)
			if got != tc.want {
				t.Errorf("value: got %q, want %q", got, tc.want)
			}
			gotWarn := strings.Contains(buf.String(), legacyEnvGatewayURL)
			if gotWarn != tc.wantWarn {
				t.Errorf("warn: got %v (%q), want %v", gotWarn, buf.String(), tc.wantWarn)
			}
		})
	}
}

// TestResolveAPIKey_EnvPrefix mirrors TestResolveGatewayURL_EnvPrefix for
// the API key variable.
func TestResolveAPIKey_EnvPrefix(t *testing.T) {
	tests := []struct {
		name      string
		canonical string
		legacy    string
		want      string
		wantWarn  bool
	}{
		{name: "canonical AA_ used when set", canonical: "k-aa", legacy: "", want: "k-aa", wantWarn: false},
		{name: "legacy AAASM_ resolves and warns", canonical: "", legacy: "k-aaasm", want: "k-aaasm", wantWarn: true},
		{name: "AA_ wins when both set", canonical: "k-aa", legacy: "k-aaasm", want: "k-aa", wantWarn: false},
		{name: "neither set returns empty", canonical: "", legacy: "", want: "", wantWarn: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deprecationOnce.apiKey = sync.Once{}
			t.Setenv(envAPIKey, tc.canonical)
			t.Setenv(legacyEnvAPIKey, tc.legacy)
			buf := captureLog(t)

			got := resolveEnvWithLegacyFallback(envAPIKey, legacyEnvAPIKey, &deprecationOnce.apiKey)
			if got != tc.want {
				t.Errorf("value: got %q, want %q", got, tc.want)
			}
			gotWarn := strings.Contains(buf.String(), legacyEnvAPIKey)
			if gotWarn != tc.wantWarn {
				t.Errorf("warn: got %v (%q), want %v", gotWarn, buf.String(), tc.wantWarn)
			}
		})
	}
}

// TestWarnLegacyEnv_FiresOnce confirms the deprecation notice is emitted
// at most once even when the legacy variable is read repeatedly.
func TestWarnLegacyEnv_FiresOnce(t *testing.T) {
	deprecationOnce.gatewayURL = sync.Once{}
	t.Setenv(envGatewayURL, "")
	t.Setenv(legacyEnvGatewayURL, "http://aaasm:7391")
	buf := captureLog(t)

	for i := 0; i < 3; i++ {
		resolveEnvWithLegacyFallback(envGatewayURL, legacyEnvGatewayURL, &deprecationOnce.gatewayURL)
	}

	if n := strings.Count(buf.String(), legacyEnvGatewayURL); n != 1 {
		t.Errorf("expected exactly one deprecation notice, got %d: %q", n, buf.String())
	}
}

// TestResolveAPIKey_LegacyEnvWarnsOnce verifies the legacy fallback is
// wired through the public resolveAPIKey entry point, not just the helper.
func TestResolveAPIKey_LegacyEnvWarnsOnce(t *testing.T) {
	deprecationOnce.apiKey = sync.Once{}
	t.Setenv(envAPIKey, "")
	t.Setenv(legacyEnvAPIKey, "k-legacy")
	buf := captureLog(t)

	if got := resolveAPIKey(""); got != "k-legacy" {
		t.Errorf("resolveAPIKey: got %q, want %q", got, "k-legacy")
	}
	if !strings.Contains(buf.String(), legacyEnvAPIKey) {
		t.Errorf("expected deprecation notice for %s, got %q", legacyEnvAPIKey, buf.String())
	}
}

// TestResolveGatewayURL_LegacyEnvWarnsOnce verifies the legacy fallback is
// wired through the public resolveGatewayURL entry point.
func TestResolveGatewayURL_LegacyEnvWarnsOnce(t *testing.T) {
	deprecationOnce.gatewayURL = sync.Once{}
	t.Setenv(envGatewayURL, "")
	t.Setenv(legacyEnvGatewayURL, "http://aaasm:7391")
	buf := captureLog(t)

	got, err := resolveGatewayURL(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://aaasm:7391" {
		t.Errorf("resolveGatewayURL: got %q, want %q", got, "http://aaasm:7391")
	}
	if !strings.Contains(buf.String(), legacyEnvGatewayURL) {
		t.Errorf("expected deprecation notice for %s, got %q", legacyEnvGatewayURL, buf.String())
	}
}
