package assembly

import (
	"testing"

	"google.golang.org/grpc/credentials/insecure"
)

// TestGatewayHostOf checks host extraction across the target forms the
// op-control channel may be handed: bare host, host:port, URL form, IPv6 (bare
// and bracketed), userinfo, and mixed case.
func TestGatewayHostOf(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"localhost":                    "localhost",
		"localhost:7391":               "localhost",
		"127.0.0.1:7391":               "127.0.0.1",
		"http://localhost:7391/path":   "localhost",
		"grpc://Gateway.Example:50051": "gateway.example",
		"gateway.example.com":          "gateway.example.com",
		"gateway.example.com:50051":    "gateway.example.com",
		"[::1]:7391":                   "[::1]",
		"[::1]":                        "[::1]",
		"::1":                          "::1",
		"user:pass@gateway.example:1":  "gateway.example",
		"  localhost:7391  ":           "localhost",
	}
	for in, want := range cases {
		if got := gatewayHostOf(in); got != want {
			t.Errorf("gatewayHostOf(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestIsLoopbackTarget asserts only loopback hosts are treated as loopback.
func TestIsLoopbackTarget(t *testing.T) {
	t.Parallel()

	loopback := []string{
		"localhost",
		"localhost:7391",
		"127.0.0.1:7391",
		"http://localhost:7391",
		"[::1]:7391",
		"::1",
	}
	for _, target := range loopback {
		if !isLoopbackTarget(target) {
			t.Errorf("isLoopbackTarget(%q) = false, want true", target)
		}
	}

	remote := []string{
		"gateway.example.com:50051",
		"10.0.0.5:7391",
		"https://gateway.example.com",
		"127.0.0.2:7391",
	}
	for _, target := range remote {
		if isLoopbackTarget(target) {
			t.Errorf("isLoopbackTarget(%q) = true, want false", target)
		}
	}
}

// TestResolveOpControlCredentials_SecureByDefault is the core regression guard
// for AAASM-3871: a non-loopback target must resolve to TLS transport
// credentials, while a loopback target keeps plaintext for the local dev
// gateway. The op-control channel carries the agent identity and operator
// pause/terminate signals, so the default to a remote host must be encrypted.
func TestResolveOpControlCredentials_SecureByDefault(t *testing.T) {
	t.Parallel()

	insecureProto := insecure.NewCredentials().Info().SecurityProtocol // "insecure"

	// Loopback → plaintext is allowed for the local dev gateway.
	if proto := resolveOpControlCredentials("localhost:7391").Info().SecurityProtocol; proto != insecureProto {
		t.Errorf("loopback target: got security protocol %q, want %q", proto, insecureProto)
	}
	if proto := resolveOpControlCredentials("[::1]:7391").Info().SecurityProtocol; proto != insecureProto {
		t.Errorf("loopback IPv6 target: got security protocol %q, want %q", proto, insecureProto)
	}

	// Non-loopback → TLS, never plaintext. This is the regression: the old
	// default handed every target insecure.NewCredentials().
	if proto := resolveOpControlCredentials("gateway.example.com:50051").Info().SecurityProtocol; proto == insecureProto {
		t.Errorf("remote target resolved to insecure transport %q; want TLS", proto)
	} else if proto != "tls" {
		t.Errorf("remote target: got security protocol %q, want \"tls\"", proto)
	}
}
