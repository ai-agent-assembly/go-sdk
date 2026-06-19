package ffi

import "unsafe"

// registerer is an optional capability a binding may implement to register the
// agent with the governance gateway via the native aa_register entry point. Only
// the real transports (cgoBridge, fallbackUDSBridge) implement it; in-memory
// test bindings may implement it too. When a binding does not, Client.Register
// reports the runtime as unavailable so the SDK layer can treat registration as
// advisory and proceed unregistered.
type registerer interface {
	register(handle unsafe.Pointer, agentID, name, framework, gatewayEndpoint string) (policyID string, status int32)
}

// Register registers the agent with the governance gateway over the native
// aa_register primitive (AAASM-3401) and returns the gateway-assigned policy id.
//
// At the native boundary aa_register is fail-closed: an unreachable or rejecting
// gateway surfaces ErrGatewayUnreachable / ErrRegisterFailed rather than failing
// open. Register preserves that — it returns the error to the caller. The SDK
// boot path (assembly.boot) treats registration as advisory: it logs the failure
// and proceeds unregistered, because the runtime / proxy / eBPF layers remain
// authoritative and an unreachable gateway must not abort the agent.
//
// agentID is the identity to register; name and framework are descriptive
// metadata recorded by the gateway; gatewayEndpoint may be empty to let the
// shared client resolve it from AA_GATEWAY_ENDPOINT or its default.
func (c *Client) Register(agentID, name, framework, gatewayEndpoint string) (policyID string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	reg, ok := c.binding.(registerer)
	if !ok {
		// No native registration transport compiled in: report unavailable so
		// the boot path proceeds unregistered.
		return "", statusToError(statusRuntimeUnavailable, "register")
	}

	if c.handle == nil {
		return "", statusToError(statusNotConnected, "register")
	}

	policyID, status := reg.register(c.handle, agentID, name, framework, gatewayEndpoint)
	if err := statusToError(status, "register"); err != nil {
		return "", err
	}
	return policyID, nil
}
