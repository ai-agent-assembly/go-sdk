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

import "time"

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
