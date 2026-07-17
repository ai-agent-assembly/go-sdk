package assembly

import "strings"

// validateOptionErrors surfaces the first error an Option collected while
// applying (a malformed WithEnforcementMode / over-long WithDelegationReason).
// It is deliberately free of any dependency on resolved config so boot can call
// it before any side-effecting resolution — a malformed option must fail fast
// without spawning an aasm subprocess or probing the network (AAASM-4811).
func validateOptionErrors(opts runtimeOptions) error {
	if len(opts.errs) > 0 {
		return opts.errs[0]
	}
	return nil
}

func validateRuntimeOptions(opts runtimeOptions) error {
	if err := validateOptionErrors(opts); err != nil {
		return err
	}

	if strings.TrimSpace(opts.gatewayURL) == "" {
		return ErrInvalidGateway
	}

	// apiKey is intentionally allowed to be empty — local mode is
	// unauth-accepting per Epic 17. The resolver fills it from env /
	// config when present, or defaults to "" for local development.

	return nil
}
