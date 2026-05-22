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
	"strings"
	"time"
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
