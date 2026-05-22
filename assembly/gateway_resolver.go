// Package-internal gateway resolver. Implements the zero-config
// developer-experience contract from Epic 17 S-G: assembly.Init(ctx)
// with no options and no environment variables should discover a local
// gateway at http://localhost:7391 — probing it, and auto-starting
// "aasm start --mode local --foreground" when not running.
//
// Resolution precedence (highest first):
//
//  1. Explicit option (WithGatewayURL / WithAPIKey)
//  2. Environment variable (AAASM_GATEWAY_URL / AAASM_API_KEY)
//  3. Config file (~/.aasm/config.yaml, gopkg.in/yaml.v3)
//  4. Local default: probe http://localhost:7391, auto-start if absent

package assembly

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultGatewayURL            = "http://localhost:7391"
	defaultHealthzPath           = "/healthz"
	defaultProbeTimeout          = 500 * time.Millisecond
	defaultAutoStartTimeout      = 5 * time.Second
	defaultAutoStartPollInterval = 100 * time.Millisecond
	defaultConfigFilePath        = "~/.aasm/config.yaml"

	envGatewayURL = "AAASM_GATEWAY_URL"
	envAPIKey     = "AAASM_API_KEY"
)

// aasmAutoStartArgs is the argv tail passed to the aasm binary when
// the resolver auto-starts a local control plane.
var aasmAutoStartArgs = []string{"start", "--mode", "local", "--foreground"}

// probeHealthz returns true when the gateway at baseUrl responds with
// a 2xx status to a GET on /healthz inside the timeout window. Any
// transport or HTTP error is swallowed and surfaces as false — the
// resolver treats unreachable as "absent" rather than fatal.
func probeHealthz(ctx context.Context, baseURL string, timeout time.Duration) bool {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := strings.TrimRight(baseURL, "/") + defaultHealthzPath
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// waitForHealthz polls the gateway healthz endpoint until success or
// the timeout elapses. Returns true on the first successful probe,
// false when no probe succeeds within timeout. The poll interval is
// short (default 100ms) so the auto-start path feels instant when
// the local CP comes up quickly.
func waitForHealthz(ctx context.Context, baseURL string, timeout, pollInterval time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if probeHealthz(ctx, baseURL, defaultProbeTimeout) {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(pollInterval):
		}
	}
	return probeHealthz(ctx, baseURL, defaultProbeTimeout)
}

func expandHome(p string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

// loadConfigFile reads ~/.aasm/config.yaml when present. Missing file,
// read errors, parse errors, and non-mapping payloads all collapse to
// an empty map — the resolver chain treats step 3 as purely advisory
// and never propagates a config-file failure.
func loadConfigFile(path string) map[string]any {
	if path == "" {
		path = defaultConfigFilePath
	}
	expanded := expandHome(path)

	data, err := os.ReadFile(expanded)
	if err != nil {
		return map[string]any{}
	}

	var parsed any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return map[string]any{}
	}
	mapped, ok := parsed.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return mapped
}
