package assembly

import (
	"errors"
	"fmt"
)

// ErrRuntimeNotInitialized indicates runtime APIs were used before Init.
var ErrRuntimeNotInitialized = errors.New("assembly: runtime is not initialized")

// ErrGovernanceUnavailable indicates a tool was wrapped with no governance
// client (no runtime was reachable at Init) while the fail-closed enforce
// posture is active, so the call is denied rather than run unchecked
// (AAASM-3109).
var ErrGovernanceUnavailable = errors.New("assembly: governance client unavailable; denying tool call (fail-closed)")

// ErrOpControlUnavailable indicates the op-control kill switch could no longer
// govern a paused op because its gateway stream died while the op was paused.
// A paused op must not resume merely because the operator's control channel
// dropped, so the tool wrapper treats this as continue-blocking under the
// fail-closed enforce posture (deny) and lets observe/disabled proceed
// (AAASM-4019).
var ErrOpControlUnavailable = errors.New("assembly: op-control stream closed while op paused; cannot confirm resume (fail-closed)")

// PolicyViolationError indicates a policy decision denied tool execution.
type PolicyViolationError struct {
	// ToolName is the name of the tool whose execution was denied.
	ToolName string
	// Reason is the human-readable explanation from the governance gateway.
	Reason string
}

// Error returns a formatted message including the tool name and denial reason
// when available.
func (e *PolicyViolationError) Error() string {
	if e == nil {
		return "assembly: policy violation"
	}

	if e.ToolName == "" && e.Reason == "" {
		return "assembly: policy violation"
	}

	if e.ToolName == "" {
		return fmt.Sprintf("assembly: policy violation: %s", e.Reason)
	}

	if e.Reason == "" {
		return fmt.Sprintf("assembly: policy violation: tool=%s", e.ToolName)
	}

	return fmt.Sprintf("assembly: policy violation: tool=%s reason=%s", e.ToolName, e.Reason)
}
