// Package assembly provides the Go SDK for Agent Assembly governance.
//
// It enables AI agent tool calls to be intercepted, checked against a
// governance policy gateway, and recorded for audit. The SDK manages
// sidecar connectivity, context propagation (agent ID, trace ID, run ID),
// and HTTP/gRPC middleware for outbound interception.
//
// # Quick Start
//
//	a, err := assembly.Init(ctx,
//	    assembly.WithGatewayURL("https://gateway.example.com"),
//	    assembly.WithAPIKey("my-key"),
//	    // Point at a running gateway's gRPC address so Init can reach the
//	    // registration path; without it (or WithSidecarBinary) Init returns
//	    // ErrSidecarUnavailable.
//	    assembly.WithSidecarAddress("127.0.0.1:50051"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer a.Close()
//
// Gateway registration (the Ed25519 possession-proof handshake) runs only under
// the opt-in native cgo binding — build with `-tags aa_ffi_go` and
// CGO_ENABLED=1. The default pure-Go build has no native transport, so it does
// not self-register; [WithSidecarAddress] makes the path reachable, the native
// binding makes it enforce-grade.
//
// # Wrapping Tools
//
// Use [WrapTools] to add governance interception to a slice of tools:
//
//	wrapped := assembly.WrapTools(tools, governanceClient,
//	    assembly.WithFailClosed(true),
//	)
//
// Each wrapped tool calls [GovernanceClient.Check] before execution and
// [GovernanceClient.RecordResult] after execution.
//
// # Context Propagation
//
// Use [WithAgentID], [WithTraceID], and [WithRunID] to attach governance
// metadata to a context. These values are automatically forwarded to the
// governance gateway on every policy check.
//
// # Interceptors
//
// [HTTPMiddleware] and [UnaryClientInterceptor] / [StreamClientInterceptor]
// provide transport-level interception for outbound HTTP and gRPC calls.
//
// # Audit Events
//
// [AuditEvent] is the Go-side mirror of the gateway's audit-trail event
// shape, including the hierarchical [CallStackNode] tree that records
// LLM / tool / result steps for inline rendering in the dashboard's
// Live Ops view. Use [CallStackNodeKindLLM], [CallStackNodeKindTool],
// and [CallStackNodeKindResult] as the canonical Kind values.
package assembly
