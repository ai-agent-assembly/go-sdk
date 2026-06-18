package assembly

import (
	"errors"
	"testing"
)

func TestValidateRuntimeOptions_ReturnsFirstCollectedError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("collected option error")
	err := validateRuntimeOptions(runtimeOptions{
		gatewayURL: "https://gw.example.com",
		errs:       []error{sentinel, errors.New("second")},
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected the first collected error, got %v", err)
	}
}

func TestValidateRuntimeOptions_RejectsBlankGatewayURL(t *testing.T) {
	t.Parallel()

	for _, blank := range []string{"", "   ", "\t"} {
		err := validateRuntimeOptions(runtimeOptions{gatewayURL: blank})
		if !errors.Is(err, ErrInvalidGateway) {
			t.Fatalf("gatewayURL %q: expected ErrInvalidGateway, got %v", blank, err)
		}
	}
}

func TestValidateRuntimeOptions_AcceptsEmptyAPIKey(t *testing.T) {
	t.Parallel()

	// Local mode is unauth-accepting: a set gateway URL with an empty API key
	// must validate cleanly.
	if err := validateRuntimeOptions(runtimeOptions{gatewayURL: "https://gw.example.com"}); err != nil {
		t.Fatalf("expected nil error for empty api key in local mode, got %v", err)
	}
}
