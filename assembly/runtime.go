package assembly

import (
	"context"
	"log"

	"github.com/ai-agent-assembly/go-sdk/internal/ffi"
)

// frameworkGo identifies this SDK's framework to the gateway at registration,
// mirroring the descriptive `framework` metadata the Python and Node SDKs send.
const frameworkGo = "go"

// Assembly is the runtime entrypoint for governance-enabled execution.
type Assembly struct {
	opts             runtimeOptions
	sidecar          SidecarClient
	sidecarConnector func(context.Context, string) (SidecarClient, error)
	ffiClient        *ffi.Client
	governance       GovernanceClient
	managedSidecar   *Sidecar
}

var newFFIClient = ffi.NewDefaultClient

// newAssembly builds an Assembly runtime from functional options.
func newAssembly(options ...Option) *Assembly {
	opts := defaultRuntimeOptions()
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}

	return &Assembly{
		opts:             opts,
		sidecarConnector: sidecarConnector,
		ffiClient:        newFFIClient(),
	}
}

// boot boots the runtime and prepares governance integrations.
func (a *Assembly) boot(ctx context.Context) error {
	resolvedURL, err := resolveGatewayURL(ctx, a.opts.gatewayURL)
	if err != nil {
		return err
	}
	a.opts.gatewayURL = resolvedURL
	a.opts.apiKey = resolveAPIKey(a.opts.apiKey)

	if err := validateRuntimeOptions(a.opts); err != nil {
		return err
	}

	if a.opts.sidecarBinary != "" {
		sc := NewSidecar(a.opts.sidecarBinary, a.opts.sidecarAddress)
		if err := sc.Start(ctx); err != nil {
			return err
		}
		if err := sc.Healthy(ctx); err != nil {
			_ = sc.Stop()
			return err
		}
		a.managedSidecar = sc
	}

	if a.opts.sidecarAddress != "" && a.ffiClient != nil {
		if err := a.ffiClient.Connect(a.opts.sidecarAddress); err == nil {
			// The runtime is reachable: route governance checks through the
			// native aa_query_policy primitive so a DENY blocks a tool call.
			a.governance = newFFIGovernanceClient(a.ffiClient)
			a.registerAgent()
			return a.ffiClient.SendEvent("register", buildRegistrationEvent(a.opts))
		}
	}

	sidecar, err := a.sidecarConnector(ctx, a.opts.sidecarAddress)
	if err != nil {
		return err
	}

	a.sidecar = sidecar
	return nil
}

// registerAgent registers this agent with the governance gateway over the native
// aa_register primitive (AAASM-3401/3404) so the gateway issues a credential
// token — stored on the shared FFI session — that authenticates every later
// aa_query_policy check (ADR 0004). It is the SDK's only direct gateway gRPC
// call. The agent's team/topology lineage (teamID / parentAgentID) now rides
// the native register itself (AAASM-3415), matching the Python and Node SDKs:
// teamID drives team-budget attribution and parentAgentID the topology graph.
// The remaining lineage detail (delegation reason, spawned-by tool) still flows
// separately as the SendEvent("register", ...) audit event.
//
// Registration is advisory at the SDK layer: although aa_register is fail-closed
// at the native boundary, a failure here is logged and boot proceeds
// unregistered, matching the Python and Node SDKs (AAASM-3402/3403). An
// unreachable or rejecting gateway must not abort the agent — the runtime /
// proxy / eBPF layers remain authoritative.
func (a *Assembly) registerAgent() {
	if a.ffiClient == nil {
		return
	}
	if _, err := a.ffiClient.Register(
		a.opts.agentID,
		a.opts.agentID,
		frameworkGo,
		a.opts.gatewayURL,
		a.opts.teamID,
		a.opts.parentAgentID,
	); err != nil {
		log.Printf("assembly: agent registration failed; proceeding unregistered: %v", err)
	}
}

// Close shuts down runtime resources.
func (a *Assembly) Close() error {
	if a.managedSidecar != nil {
		if err := a.managedSidecar.Stop(); err != nil {
			return err
		}
		a.managedSidecar = nil
	}
	a.sidecar = nil
	return nil
}
