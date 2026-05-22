package assembly

// ConfigurationError signals that the SDK could not resolve the gateway
// configuration — for example, the local gateway is absent and the
// “aasm“ binary is missing from “PATH“.
//
// Mirrors “ConfigurationError“ in the Python and Node SDKs so the
// cross-SDK error contract stays aligned per Epic 17 S-G.
type ConfigurationError struct {
	Message string
}

func (e *ConfigurationError) Error() string {
	return "assembly: " + e.Message
}

// GatewayError signals that the SDK has a gateway URL but cannot talk
// to it — for example, “aasm“ was spawned but “/healthz“ did not
// become ready within the auto-start timeout window.
type GatewayError struct {
	Message string
}

func (e *GatewayError) Error() string {
	return "assembly: " + e.Message
}
