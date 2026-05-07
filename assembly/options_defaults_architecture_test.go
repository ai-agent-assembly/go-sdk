package assembly

import (
	"testing"
	"time"
)

func TestDefaultRuntimeOptions(t *testing.T) {
	t.Parallel()

	opts := defaultRuntimeOptions()
	if opts.failClosed {
		t.Fatal("expected failClosed default to false")
	}
	if opts.timeout != defaultGatewayTimeout {
		t.Fatalf("expected default timeout %v, got %v", defaultGatewayTimeout, opts.timeout)
	}
}

func TestOptionsMutateRuntimeOptions(t *testing.T) {
	t.Parallel()

	opts := defaultRuntimeOptions()
	WithGatewayURL("https://gateway.example.com")(&opts)
	WithAPIKey("test-key")(&opts)
	WithFailClosed(true)(&opts)
	WithTimeout(3 * time.Second)(&opts)

	if opts.gatewayURL != "https://gateway.example.com" {
		t.Fatalf("expected gateway url to be set, got %q", opts.gatewayURL)
	}
	if opts.apiKey != "test-key" {
		t.Fatalf("expected api key to be set, got %q", opts.apiKey)
	}
	if !opts.failClosed {
		t.Fatal("expected failClosed to be true")
	}
	if opts.timeout != 3*time.Second {
		t.Fatalf("expected timeout to be 3s, got %v", opts.timeout)
	}
}

func TestTopologyOptionsSetFields(t *testing.T) {
	t.Parallel()

	opts := defaultRuntimeOptions()
	WithParentAgent("parent-123")(&opts)
	WithTeam("team-alpha")(&opts)
	WithDelegationReason("sub-task delegation")(&opts)

	if opts.parentAgentID != "parent-123" {
		t.Fatalf("expected parentAgentID %q, got %q", "parent-123", opts.parentAgentID)
	}
	if opts.teamID != "team-alpha" {
		t.Fatalf("expected teamID %q, got %q", "team-alpha", opts.teamID)
	}
	if opts.delegationReason != "sub-task delegation" {
		t.Fatalf("expected delegationReason %q, got %q", "sub-task delegation", opts.delegationReason)
	}
}

func TestTopologyOptionsDefaultToEmpty(t *testing.T) {
	t.Parallel()

	opts := defaultRuntimeOptions()

	if opts.parentAgentID != "" {
		t.Fatalf("expected empty parentAgentID by default, got %q", opts.parentAgentID)
	}
	if opts.teamID != "" {
		t.Fatalf("expected empty teamID by default, got %q", opts.teamID)
	}
	if opts.delegationReason != "" {
		t.Fatalf("expected empty delegationReason by default, got %q", opts.delegationReason)
	}
}
