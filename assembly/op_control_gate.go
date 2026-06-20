package assembly

import (
	"context"

	oteltrace "go.opentelemetry.io/otel/trace"
)

// OpController is the slice of [OpControlSubscriber] the tool wrapper consults
// before executing a tool (AAASM-3501). It is an interface so a tool path can
// be wired with the live subscriber in production while tests inject a fake.
// The concrete *OpControlSubscriber satisfies it via WaitForOp.
//
// WaitForOp blocks until the op identified by opID is runnable, returns nil
// when it may proceed, and returns an *OpTerminatedError when the gateway has
// terminated it. A paused op blocks until the gateway resumes (or terminates)
// it; this is the cooperative-pause point on the Go tool path.
type OpController interface {
	WaitForOp(ctx context.Context, opID string) error
}

// The live subscriber must remain wirable as an OpController, so a terminate /
// pause from the gateway's OpControlStream reaches the tool path (AAASM-3501).
var _ OpController = (*OpControlSubscriber)(nil)

// resolveOpID derives the op-control op identifier ("{trace_id}:{span_id}") for
// the current tool call (AAASM-3501).
//
// An explicit op ID — set via [WithOpID] — wins. Otherwise the ID is composed
// from the call's trace and span identity: the assembly trace ID (explicit or
// from the active OpenTelemetry span) joined with the active span ID. When no
// trace identity is present the call is not part of a tracked op, so there is
// nothing for the kill switch to address and the empty string is returned —
// the caller skips op control entirely.
func resolveOpID(ctx context.Context) string {
	if explicit := opIDFromContext(ctx); explicit != "" {
		return explicit
	}
	traceID := TraceIDFromContext(ctx)
	if traceID == "" {
		return ""
	}
	return traceID + ":" + spanIDFromContext(ctx)
}

// spanIDFromContext returns the active OpenTelemetry span ID, or an empty
// string when no valid span is on the context.
func spanIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	spanCtx := oteltrace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		return spanCtx.SpanID().String()
	}
	return ""
}
