package assembly

import (
	"errors"
	"testing"
)

// TestWithControlPlaneURLStoresValue verifies the WithControlPlaneURL option
// records its argument on the internal runtime options struct.
func TestWithControlPlaneURLStoresValue(t *testing.T) {
	opts := defaultRuntimeOptions()
	WithControlPlaneURL("https://cp.example.com")(&opts)
	if opts.controlPlaneURL != "https://cp.example.com" {
		t.Fatalf("expected controlPlaneURL to be stored, got %q", opts.controlPlaneURL)
	}
}

// TestResolveURLEnvFallback drives the explicit > env-var > error precedence
// for both the gateway and control-plane resolvers in a table.
func TestResolveURLEnvFallback(t *testing.T) {
	tests := []struct {
		name     string
		explicit string
		envVar   string
		envValue string
		required bool
		resolve  func(explicit string, required bool) (string, error)
		wantURL  string
		wantErr  error
	}{
		{
			name:     "control-plane explicit option wins over env",
			explicit: "https://explicit.example.com",
			envVar:   envFallbackControlPlaneURL,
			envValue: "https://env.example.com",
			required: true,
			resolve:  resolveControlPlaneURL,
			wantURL:  "https://explicit.example.com",
			wantErr:  nil,
		},
		{
			name:     "control-plane env-var fallback when option empty",
			explicit: "",
			envVar:   envFallbackControlPlaneURL,
			envValue: "https://env.example.com",
			required: true,
			resolve:  resolveControlPlaneURL,
			wantURL:  "https://env.example.com",
			wantErr:  nil,
		},
		{
			name:     "control-plane required missing returns error",
			explicit: "",
			envVar:   envFallbackControlPlaneURL,
			envValue: "",
			required: true,
			resolve:  resolveControlPlaneURL,
			wantURL:  "",
			wantErr:  ErrInvalidControlPlane,
		},
		{
			name:     "control-plane optional missing returns empty",
			explicit: "",
			envVar:   envFallbackControlPlaneURL,
			envValue: "",
			required: false,
			resolve:  resolveControlPlaneURL,
			wantURL:  "",
			wantErr:  nil,
		},
		{
			name:     "gateway explicit option wins over env",
			explicit: "https://gw-explicit.example.com",
			envVar:   envFallbackGatewayURL,
			envValue: "https://gw-env.example.com",
			required: true,
			resolve:  resolveGatewayURLWithEnvFallback,
			wantURL:  "https://gw-explicit.example.com",
			wantErr:  nil,
		},
		{
			name:     "gateway env-var fallback when option empty",
			explicit: "",
			envVar:   envFallbackGatewayURL,
			envValue: "https://gw-env.example.com",
			required: true,
			resolve:  resolveGatewayURLWithEnvFallback,
			wantURL:  "https://gw-env.example.com",
			wantErr:  nil,
		},
		{
			name:     "gateway required missing returns error",
			explicit: "",
			envVar:   envFallbackGatewayURL,
			envValue: "",
			required: true,
			resolve:  resolveGatewayURLWithEnvFallback,
			wantURL:  "",
			wantErr:  ErrInvalidGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envVar, tt.envValue)

			gotURL, gotErr := tt.resolve(tt.explicit, tt.required)
			if gotURL != tt.wantURL {
				t.Errorf("url = %q, want %q", gotURL, tt.wantURL)
			}
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("err = %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}
