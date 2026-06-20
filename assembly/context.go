package assembly

import (
	"context"
	"fmt"

	"github.com/oklog/ulid/v2"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	agentIDContextKey       contextKey = "assembly.agent_id"
	traceIDContextKey       contextKey = "assembly.trace_id"
	runIDContextKey         contextKey = "assembly.run_id"
	parentAgentIDContextKey contextKey = "assembly.parent_agent_id"
	spawnedByToolContextKey contextKey = "assembly.spawned_by_tool"
	opIDContextKey          contextKey = "assembly.op_id"
)

// WithOpID returns a new context carrying an explicit op-control op identifier
// ("{trace_id}:{span_id}"). An explicit op ID overrides the trace/span-derived
// one when the tool wrapper consults the live kill switch (AAASM-3501); set it
// only when the caller already knows the gateway's op ID for this call.
func WithOpID(ctx context.Context, opID string) context.Context {
	return context.WithValue(ctx, opIDContextKey, opID)
}

// opIDFromContext returns the explicit op ID set by [WithOpID], or an empty
// string if absent.
func opIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	opID, _ := ctx.Value(opIDContextKey).(string)
	return opID
}

// WithAgentID returns a new context containing the assembly agent ID.
func WithAgentID(ctx context.Context, agentID string) context.Context {
	return context.WithValue(ctx, agentIDContextKey, agentID)
}

// AgentIDFromContext returns the assembly agent ID, or an empty string if absent.
func AgentIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	agentID, _ := ctx.Value(agentIDContextKey).(string)
	return agentID
}

// WithTraceID returns a new context containing the assembly trace ID.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDContextKey, traceID)
}

// TraceIDFromContext returns the assembly trace ID, or an empty string if absent.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	traceID, _ := ctx.Value(traceIDContextKey).(string)
	if traceID != "" {
		return traceID
	}

	spanCtx := oteltrace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String()
	}

	return ""
}

// WithRunID returns a new context containing the assembly run ID.
func WithRunID(ctx context.Context, runID string) context.Context {
	return context.WithValue(ctx, runIDContextKey, runID)
}

// RunIDFromContext returns the assembly run ID, generating one when absent.
func RunIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return generateRunID()
	}

	runID, _ := ctx.Value(runIDContextKey).(string)
	if runID != "" {
		return runID
	}

	return generateRunID()
}

// EnsureRunID returns a context with a stable run ID for the current context tree.
func EnsureRunID(ctx context.Context) (context.Context, string) {
	if ctx == nil {
		ctx = context.Background()
	}

	runID, _ := ctx.Value(runIDContextKey).(string)
	if runID != "" {
		return ctx, runID
	}

	runID = generateRunID()
	return WithRunID(ctx, runID), runID
}

func generateRunID() string {
	return fmt.Sprintf("run_%s", ulid.Make().String())
}

// ContextWithParentAgentID returns a context carrying the parent agent's ID.
// Child agents that call Init with this context auto-inherit the parentAgentID
// without needing to pass it explicitly via WithParentAgentID.
func ContextWithParentAgentID(ctx context.Context, parentAgentID string) context.Context {
	return context.WithValue(ctx, parentAgentIDContextKey, parentAgentID)
}

// ParentAgentIDFromContext returns the parent agent ID set by ContextWithParentAgentID,
// or an empty string if absent.
func ParentAgentIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(parentAgentIDContextKey).(string)
	return id
}

// ContextWithSpawnedByTool returns a context carrying the name of the tool that
// spawned this agent, for topology tracking.
func ContextWithSpawnedByTool(ctx context.Context, tool string) context.Context {
	return context.WithValue(ctx, spawnedByToolContextKey, tool)
}

// SpawnedByToolFromContext returns the spawned-by-tool name set by ContextWithSpawnedByTool,
// or an empty string if absent.
func SpawnedByToolFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	tool, _ := ctx.Value(spawnedByToolContextKey).(string)
	return tool
}
