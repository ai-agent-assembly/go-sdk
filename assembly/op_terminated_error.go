package assembly

import "fmt"

// OpTerminatedError is returned by [OpControlSubscriber.WaitForOp] when the
// gateway signals a terminate for the awaited op (AAASM-1422 PR-G).
//
// Carries the originating OpID so callers can correlate the failure against
// the operation they were tracking. Mirrors PR-E's python-sdk
// `OpTerminatedError(message, *, op_id)` and PR-F's node-sdk
// `OpTerminatedError(message, opId)` shapes so a future cross-SDK
// abstraction can unify them.
type OpTerminatedError struct {
	// OpID is the gateway's stable op identifier — "{trace_id}:{span_id}".
	OpID string
	// Reason is the human-readable message describing why the op was
	// terminated. Optional; falls back to a generic message when empty.
	Reason string
}

// Error implements the error interface.
func (e *OpTerminatedError) Error() string {
	if e == nil {
		return ""
	}
	if e.Reason == "" {
		return fmt.Sprintf("op %s was terminated by the gateway", e.OpID)
	}
	return fmt.Sprintf("op %s was terminated: %s", e.OpID, e.Reason)
}
