package assembly

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Credential/token hygiene for the Go SDK (AAASM-3647).
//
// The Go layer's credential surface is the configured apiKey (the gateway
// credential_token lives Rust-side in aa-sdk-client). These tests pin that the
// apiKey never reaches a log line — including the fail-open governance path
// that logs on check error — and that no production source formats a
// *CheckActionRequest (whose generated String() would dump CredentialToken)
// into a logger.

const sentinelAPIKey = "SENTINEL-API-KEY-DO-NOT-LOG"

// redactStubClient is a GovernanceClient whose Check fails, to exercise the
// fail-open log path in tool_wrapper.go.
type redactStubClient struct{}

func (redactStubClient) Check(context.Context, CheckRequest) (Decision, error) {
	return Decision{}, errors.New("gateway unreachable")
}
func (redactStubClient) WaitForApproval(context.Context, ApprovalRequest) (Decision, error) {
	return Decision{}, nil
}
func (redactStubClient) RecordResult(context.Context, RecordRequest) error { return nil }
func (redactStubClient) Close() error                                      { return nil }

// redactTool is a minimal Tool for driving the wrapper.
type redactTool struct{}

func (redactTool) Name() string                                 { return "web_search" }
func (redactTool) Description() string                          { return "desc" }
func (redactTool) Call(context.Context, string) (string, error) { return "ok", nil }

func TestRuntimeOptionsStringElidesAPIKey(t *testing.T) {
	t.Parallel()

	opts := defaultRuntimeOptions()
	opts.apiKey = sentinelAPIKey
	opts.gatewayURL = "https://gw.example.com"

	for _, rendered := range []string{
		opts.String(),
		fmt.Sprintf("%v", opts),
		fmt.Sprintf("%+v", opts),
		fmt.Sprint(opts),
	} {
		if strings.Contains(rendered, sentinelAPIKey) {
			t.Fatalf("runtimeOptions render leaked the apiKey: %s", rendered)
		}
		if !strings.Contains(rendered, "<redacted>") {
			t.Fatalf("expected <redacted> marker, got: %s", rendered)
		}
	}
}

func TestFailOpenGovernanceLogDoesNotLeakAPIKey(t *testing.T) {
	buf := captureLog(t)

	opts := defaultRuntimeOptions()
	opts.apiKey = sentinelAPIKey
	// Observe posture so a check error fails open and logs (the leak surface).
	opts.enforcementMode = EnforcementModeObserve
	opts.failClosed = false

	wrapped := newAssemblyTool(redactTool{}, redactStubClient{}, opts)

	// Call must succeed (fail-open) and emit the "allowing tool call" warning.
	if _, err := wrapped.Call(context.Background(), "query"); err != nil {
		t.Fatalf("fail-open call should not error, got %v", err)
	}

	logged := buf.String()
	if !strings.Contains(logged, "governance check failed") {
		t.Fatalf("expected the fail-open warning to be logged, got: %q", logged)
	}
	if strings.Contains(logged, sentinelAPIKey) {
		t.Fatalf("fail-open log leaked the apiKey: %q", logged)
	}
}

// TestNoProductionSourceLogsCheckActionRequest is a lint-style guard: the
// generated proto CheckActionRequest.String() dumps CredentialToken, so no
// production source may format a *CheckActionRequest into a log/print. Scans the
// non-test .go sources under assembly/ for a log/fmt call whose arguments
// reference a CheckActionRequest.
func TestNoProductionSourceLogsCheckActionRequest(t *testing.T) {
	t.Parallel()

	root := "."
	logCall := regexp.MustCompile(`(?s)\b(log\.(Print|Printf|Println|Fatal|Fatalf|Fatalln|Panic|Panicf|Panicln)|fmt\.(Print|Printf|Println|Sprint|Sprintf|Sprintln))\([^)]*CheckActionRequest`)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if logCall.Match(src) {
			t.Errorf("%s formats a CheckActionRequest into a logger — its String() dumps CredentialToken", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
}
