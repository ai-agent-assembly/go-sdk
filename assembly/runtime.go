package assembly

import (
	"context"
	"errors"
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
	// ffiConnected is set once boot successfully opens the native FFI session so
	// Close knows to tear it down. It gates the Disconnect call: the default
	// pure-Go build never connects (it falls through to the sidecar connector),
	// and Disconnect on a never-connected client returns a not-connected error —
	// so a guarded Close stays a no-op there rather than surfacing a spurious
	// error (AAASM-4832).
	ffiConnected bool
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
	// Surface malformed-option errors (a bad WithEnforcementMode / over-long
	// WithDelegationReason) BEFORE any side-effecting resolution. resolveGatewayURL
	// can spawn an aasm subprocess and probe the network; a caller who passed a
	// malformed option must fail fast with no such side effects (AAASM-4811). The
	// gateway-URL presence check stays after resolution below, since resolution is
	// what fills an empty URL from env/config/local-default.
	if err := validateOptionErrors(a.opts); err != nil {
		return err
	}

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

	// stopManagedSidecar releases the subprocess started above (if any) on a
	// boot failure past this point. Without it, an error returned here
	// propagates out of Init, which discards the *Assembly — and with it the
	// only handle able to Stop() the managed process — leaking it (AAASM-4789).
	stopManagedSidecar := func() {
		if a.managedSidecar != nil {
			_ = a.managedSidecar.Stop()
			a.managedSidecar = nil
		}
	}

	if a.opts.sidecarAddress != "" && a.ffiClient != nil {
		// Forward the agent id (signed into the handshake, AAASM-3587) and the
		// Go-module SDK version (Version) so the installed package version — not
		// the aa-sdk-client crate version — is signed into the handshake for
		// accurate downgrade detection (AAASM-3683).
		if err := a.ffiClient.Connect(a.opts.sidecarAddress, a.opts.agentID, Version); err == nil {
			// The runtime is reachable: route governance checks through the
			// native aa_query_policy primitive so a DENY blocks a tool call.
			a.ffiConnected = true
			a.governance = newFFIGovernanceClient(a.ffiClient)
			a.registerAgent()
			if err := a.ffiClient.SendEvent("register", buildRegistrationEvent(a.opts)); err != nil {
				stopManagedSidecar()
				return err
			}
			return nil
		}
	}

	sidecar, err := a.sidecarConnector(ctx, a.opts.sidecarAddress)
	if err != nil {
		stopManagedSidecar()
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

// Close shuts down runtime resources: it tears down the native FFI session (when
// boot opened one) and stops the managed sidecar subprocess (when one was
// launched). Both teardowns run even if the first errors; the combined error is
// returned. Disconnecting the FFI session releases the native session, its IPC
// reader thread, and the registered credential — without it those leak for the
// process lifetime (AAASM-4832). The default pure-Go build never connected, so
// the ffiConnected guard keeps Close a no-op there.
func (a *Assembly) Close() error {
	var errs []error
	if a.ffiConnected && a.ffiClient != nil {
		if err := a.ffiClient.Disconnect(); err != nil {
			errs = append(errs, err)
		}
		a.ffiConnected = false
	}
	if a.managedSidecar != nil {
		if err := a.managedSidecar.Stop(); err != nil {
			errs = append(errs, err)
		}
		a.managedSidecar = nil
	}
	a.sidecar = nil
	return errors.Join(errs...)
}
