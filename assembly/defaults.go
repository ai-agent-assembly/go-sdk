package assembly

import "time"

const defaultGatewayTimeout = 500 * time.Millisecond

func defaultRuntimeOptions() runtimeOptions {
	return runtimeOptions{
		// Fail-closed is the secure default (AAASM-3108): a governance check
		// transport error or timeout denies the tool call rather than letting
		// it run unchecked. Callers can opt back into fail-open with
		// WithFailClosed(false), but only the observe/disabled enforcement
		// postures actually allow on check error.
		failClosed: true,
		timeout:    defaultGatewayTimeout,
	}
}
