// Outbound-transport hooks for the assembly package.
//
// The three constructors below (HTTPMiddleware, UnaryClientInterceptor,
// StreamClientInterceptor) are **reserved pass-throughs**: they wrap outbound
// HTTP/gRPC traffic and forward every request unmodified. They exist to reserve
// a stable wiring seam for a future in-transport enforcement layer; they do
// **not** themselves enforce policy, redact, or emit governance events today.
//
// Governance on the SDK path runs through the tool wrapper
// ([AssemblyTool.Call] → the FFI governance client and op-control gate), not
// through these hooks. Installing them is harmless but provides no interception —
// do not rely on them as a security boundary. The authoritative enforcement
// points remain the runtime, sidecar proxy, and eBPF layers.

package assembly

import (
	"context"
	"net/http"

	"google.golang.org/grpc"
)

// HTTPMiddleware wraps an outbound HTTP [http.RoundTripper]. It is a reserved
// no-op pass-through: every request is forwarded to next (or
// [http.DefaultTransport] when next is nil) unmodified. It does not enforce
// policy or emit events — see the package-level note on these hooks.
func HTTPMiddleware(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}

	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return next.RoundTrip(req)
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// UnaryClientInterceptor returns a reserved no-op outbound unary gRPC
// interceptor: it invokes the RPC unmodified and performs no governance. It
// reserves the wiring seam for future in-transport enforcement — see the
// package-level note on these hooks.
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor returns a reserved no-op outbound streaming gRPC
// interceptor: it opens the stream unmodified and performs no governance. It
// reserves the wiring seam for future in-transport enforcement — see the
// package-level note on these hooks.
func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		return streamer(ctx, desc, cc, method, opts...)
	}
}
