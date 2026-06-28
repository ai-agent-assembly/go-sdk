// Shared transport-security decision for the op-control gRPC channel
// (AAASM-3871). Mirrors python-sdk's core/transport_security.py and node-sdk's
// op-control.ts resolveOpControlCredentials so all three SDKs agree on what
// "secure by default" means for this channel.
//
// The op-control stream carries the agent identity triple and operator
// pause / terminate / resume signals. An unencrypted channel to a non-loopback
// host lets an on-path attacker read the AgentId, suppress TERMINATE, or inject
// PAUSE in cleartext — so plaintext is only the documented default for a
// loopback (local dev) gateway; any remote target must be encrypted unless the
// caller explicitly opts out by supplying their own DialOptions.

package assembly

import (
	"crypto/tls"
	"strings"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// loopbackHosts are the hosts treated as loopback for the secure-by-default
// transport decision. A loopback gateway is the local dev-mode control plane,
// where plaintext gRPC is the documented default; anything else is presumed
// remote and must be encrypted. Mirrors LOOPBACK_HOSTS in python-sdk/node-sdk.
var loopbackHosts = map[string]struct{}{
	"localhost": {},
	"127.0.0.1": {},
	"::1":       {},
	"[::1]":     {},
}

// gatewayHostOf extracts the bare host from a gRPC target. It accepts a
// "host:port" target, a bare host, or a URL form ("scheme://host:port/path").
// A bracketed IPv6 form ("[::1]:7391") keeps its brackets so it matches
// loopbackHosts; the result is lower-cased for case-insensitive comparison.
func gatewayHostOf(gatewayURL string) string {
	target := strings.TrimSpace(gatewayURL)

	// URL form with a scheme — drop everything up to and including "://".
	if i := strings.Index(target, "://"); i != -1 {
		target = target[i+3:]
	}
	// Drop any userinfo ("user:pass@host") — only the host matters here.
	if i := strings.LastIndex(target, "@"); i != -1 {
		target = target[i+1:]
	}
	// Drop a path/query suffix if a URL form was passed.
	if i := strings.IndexByte(target, '/'); i != -1 {
		target = target[:i]
	}

	// Bracketed IPv6 ("[::1]" or "[::1]:7391"): keep the brackets, drop only
	// the trailing ":port" outside the closing bracket.
	if strings.HasPrefix(target, "[") {
		if closeIdx := strings.IndexByte(target, ']'); closeIdx != -1 {
			return strings.ToLower(target[:closeIdx+1])
		}
		return strings.ToLower(target)
	}
	// Bare IPv6 without brackets (more than one colon) — no port to strip.
	if strings.Count(target, ":") > 1 {
		return strings.ToLower(target)
	}
	// host or host:port — drop the port.
	if i := strings.LastIndex(target, ":"); i != -1 {
		target = target[:i]
	}
	return strings.ToLower(target)
}

// isLoopbackTarget reports whether gatewayURL points at a loopback host.
func isLoopbackTarget(gatewayURL string) bool {
	_, ok := loopbackHosts[gatewayHostOf(gatewayURL)]
	return ok
}

// resolveOpControlCredentials picks the transport credentials for the
// op-control channel, secure by default. A loopback target gets plaintext (the
// local dev gateway); any other target gets TLS using the host's system root
// CAs. Mirrors resolveOpControlCredentials in node-sdk and
// require_secure_grpc_target in python-sdk.
//
// Callers that need a different posture (a custom CA, mTLS, or plaintext to a
// remote host) pass their own DialOptions to Connect, which bypasses this
// default entirely — that is the explicit opt-out.
func resolveOpControlCredentials(gatewayURL string) credentials.TransportCredentials {
	if isLoopbackTarget(gatewayURL) {
		return insecure.NewCredentials()
	}
	return credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
}
