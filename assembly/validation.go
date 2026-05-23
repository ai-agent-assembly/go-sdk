package assembly

import "strings"

func validateRuntimeOptions(opts runtimeOptions) error {
	if len(opts.errs) > 0 {
		return opts.errs[0]
	}

	if strings.TrimSpace(opts.gatewayURL) == "" {
		return ErrInvalidGateway
	}

	// apiKey is intentionally allowed to be empty — local mode is
	// unauth-accepting per Epic 17. The resolver fills it from env /
	// config when present, or defaults to "" for local development.

	return nil
}
