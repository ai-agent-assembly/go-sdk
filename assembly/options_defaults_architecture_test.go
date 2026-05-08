package assembly

import (
	"strings"
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
	WithParentAgentID("parent-123")(&opts)
	WithTeamID("team-alpha")(&opts)
	WithDelegationReason("sub-task delegation")(&opts)
	WithSpawnedByTool("search_tool")(&opts)

	if opts.parentAgentID != "parent-123" {
		t.Fatalf("expected parentAgentID %q, got %q", "parent-123", opts.parentAgentID)
	}
	if opts.teamID != "team-alpha" {
		t.Fatalf("expected teamID %q, got %q", "team-alpha", opts.teamID)
	}
	if opts.delegationReason != "sub-task delegation" {
		t.Fatalf("expected delegationReason %q, got %q", "sub-task delegation", opts.delegationReason)
	}
	if opts.spawnedByTool != "search_tool" {
		t.Fatalf("expected spawnedByTool %q, got %q", "search_tool", opts.spawnedByTool)
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
	if opts.spawnedByTool != "" {
		t.Fatalf("expected empty spawnedByTool by default, got %q", opts.spawnedByTool)
	}
}

func TestWithDelegationReasonOverlongIsRejected(t *testing.T) {
	t.Parallel()

	longReason := strings.Repeat("x", 257)
	opts := defaultRuntimeOptions()
	WithDelegationReason(longReason)(&opts)

	if len(opts.errs) == 0 {
		t.Fatal("expected error for overlong delegationReason, got none")
	}
	if opts.delegationReason != "" {
		t.Fatalf("expected delegationReason to be empty when invalid, got %q", opts.delegationReason)
	}
}

func TestWithDelegationReasonAcceptsExactly256(t *testing.T) {
	t.Parallel()

	exactReason := strings.Repeat("x", 256)
	opts := defaultRuntimeOptions()
	WithDelegationReason(exactReason)(&opts)

	if len(opts.errs) != 0 {
		t.Fatalf("expected no error for 256-char reason, got %v", opts.errs)
	}
	if opts.delegationReason != exactReason {
		t.Fatal("expected delegationReason to be stored when exactly 256 chars")
	}
}

func TestValidateRuntimeOptionsReturnsOptionError(t *testing.T) {
	t.Parallel()

	longReason := strings.Repeat("x", 257)
	opts := defaultRuntimeOptions()
	opts.gatewayURL = "https://gw.example.com"
	opts.apiKey = "key"
	WithDelegationReason(longReason)(&opts)

	err := validateRuntimeOptions(opts)
	if err == nil {
		t.Fatal("expected error from validateRuntimeOptions, got nil")
	}
}
