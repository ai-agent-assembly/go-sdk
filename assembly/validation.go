package assembly

import "strings"

func validateRuntimeOptions(opts runtimeOptions) error {
	if len(opts.errs) > 0 {
		return opts.errs[0]
	}

	if strings.TrimSpace(opts.gatewayURL) == "" {
		return ErrInvalidGateway
	}

	if strings.TrimSpace(opts.apiKey) == "" {
		return ErrInvalidAPIKey
	}

	return nil
}
