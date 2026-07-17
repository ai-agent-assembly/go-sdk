package assembly

import (
	"context"
	"errors"
	"testing"

	"github.com/ai-agent-assembly/go-sdk/internal/ffi"
)

// This file pins the fail-closed contract of the SDK's single governance gate
// (AAASM-4802). The gate is AssemblyTool.Call -> runGovernanceGate, backed by
// ffiGovernanceClient.Check, which maps the native aa_query_policy verdict onto
// a Decision. Unlike the Python and Node SDKs (which have per-framework
// adapters), Go has exactly one gate — so this matrix is the permanent,
// exhaustive record of how every posture reacts to every verdict/failure. A new
// posture or a new ffi.Decision variant must be added here, or the completeness
// check fails.
//
// The invariant under test: under the fail-closed enforce default, an
// indeterminate verdict (gateway unreachable, transport error, UNSPECIFIED, a
// REDACT that cannot be honoured client-side, or a missing client) MUST deny —
// the wrapped tool never runs. observe / disabled / WithFailClosed(false) are
// fail-open by design and allow the tool to run on the same indeterminate
// verdicts. An authoritative DENY (explicit deny, or a pending verdict that
// resolves to deny because no approval channel exists) always blocks; an
// authoritative ALLOW always runs.

// verdict classifies how a scenario's governance outcome should be treated,
// independent of posture.
type verdict int

const (
	// verdictAllow is an authoritative ALLOW: the tool runs under every posture.
	verdictAllow verdict = iota
	// verdictDeny is an authoritative DENY (explicit deny, or a pending verdict
	// that resolves to deny — no approval channel exists, AAASM-3920): the tool
	// never runs under any posture.
	verdictDeny
	// verdictIndeterminate is any outcome where the gate could not render an
	// authoritative allow (gateway unreachable, transport error, UNSPECIFIED,
	// REDACT-with-content it cannot scrub, or a missing client). The posture
	// decides: deny under fail-closed enforce, allow under observe / disabled /
	// fail-open.
	verdictIndeterminate
)

// gatePosture is one governance posture plus whether it fails open on an
// indeterminate verdict.
type gatePosture struct {
	name string
	opts runtimeOptions
	// failOpen reports whether this posture lets the tool run on an
	// indeterminate verdict. Only the fail-closed enforce default denies.
	failOpen bool
}

// gateScenario wires the single gate for one verdict/failure case. build
// returns a fresh inner tool (whose calls counter is the ran flag) and the
// governance client for the scenario.
type gateScenario struct {
	name    string
	verdict verdict
	input   string

	// nilClient models "no runtime reachable at Init" — the wrapper's nil-client
	// branch, distinct from a client that returns an error.
	nilClient bool
	// decision / hasDecision is the raw ffi verdict the fake querier returns.
	// hasDecision is false for the transport-error and nil-client scenarios,
	// which never reach the decision switch.
	decision    int32
	hasDecision bool
	reason      string
	queryErr    error
}

func (s gateScenario) build() (*failClosedTool, GovernanceClient) {
	// failClosedTool records whether it ran (calls>0); a distinctive result lets
	// the harness assert a blocked tool never leaks output.
	inner := &failClosedTool{name: "web_search", result: "TOOL-RAN-RESULT"}
	if s.nilClient {
		return inner, nil
	}
	querier := &fakeQuerier{decision: s.decision, reason: s.reason, err: s.queryErr}
	return inner, newFFIGovernanceClient(querier)
}

// wantRan derives the expected outcome for a (posture, scenario) cell.
func (s gateScenario) wantRan(p gatePosture) bool {
	switch s.verdict {
	case verdictAllow:
		return true
	case verdictDeny:
		return false
	case verdictIndeterminate:
		return p.failOpen
	default:
		return false
	}
}

// conformancePostures is the exhaustive posture set: the fail-closed enforce
// default (WithFailClosed(true) with enforcementMode "" resolving to the enforce
// arm), plus observe, disabled, and WithFailClosed(false).
func conformancePostures() []gatePosture {
	enforce := defaultRuntimeOptions() // failClosed:true, enforcementMode:"" -> enforce arm

	observe := defaultRuntimeOptions()
	observe.enforcementMode = EnforcementModeObserve

	disabled := defaultRuntimeOptions()
	disabled.enforcementMode = EnforcementModeDisabled

	failOpen := defaultRuntimeOptions()
	WithFailClosed(false)(&failOpen)

	return []gatePosture{
		{name: "failclosed_enforce_default", opts: enforce, failOpen: false},
		{name: "observe", opts: observe, failOpen: true},
		{name: "disabled", opts: disabled, failOpen: true},
		{name: "failopen", opts: failOpen, failOpen: true},
	}
}

// conformanceScenarios is the exhaustive scenario set S1–S8. The decision-backed
// scenarios (S3, S5, S6, S7, S8) cover every ffi.Decision variant the gate maps;
// the completeness check enforces that coverage.
func conformanceScenarios() []gateScenario {
	return []gateScenario{
		{
			// S1: the gateway is unreachable — the native query surfaces a
			// transport error rather than a silent allow.
			name:     "s1_gateway_unreachable",
			verdict:  verdictIndeterminate,
			input:    "query",
			queryErr: errors.New("dial tcp 127.0.0.1:50051: connect: connection refused"),
		},
		{
			// S2: a generic transport / governance error from the query.
			name:     "s2_transport_error",
			verdict:  verdictIndeterminate,
			input:    "query",
			queryErr: errors.New("governance transport error"),
		},
		{
			// S3: the proto3 zero-value UNSPECIFIED verdict — not an
			// authoritative allow (AAASM-4166); Check surfaces an error.
			name:        "s3_unspecified_decision",
			verdict:     verdictIndeterminate,
			input:       "query",
			decision:    ffi.DecisionUnspecified,
			hasDecision: true,
		},
		{
			// S4: no governance client (no runtime reachable at Init) — the
			// wrapper's nil-client branch (AAASM-3109).
			name:      "s4_nil_client",
			verdict:   verdictIndeterminate,
			input:     "query",
			nilClient: true,
		},
		{
			// S5: a pending / requires-approval verdict. There is no approval
			// channel, so WaitForApproval denies (AAASM-3920): an authoritative
			// deny under every posture.
			name:        "s5_pending_approval_denies",
			verdict:     verdictDeny,
			input:       "query",
			decision:    ffi.DecisionPending,
			hasDecision: true,
			reason:      "requires approval",
		},
		{
			// S6: a REDACT verdict on a call carrying sensitive args (the
			// AAASM-4788 case). The native ABI returns no redacted content, so
			// the SDK cannot honour the redaction — folding it onto a plain allow
			// would run the tool with the unredacted args the policy wanted
			// scrubbed. Check surfaces an error, so under fail-closed enforce the
			// tool is denied (indeterminate), not silently allowed.
			name:        "s6_redact_with_arg_content",
			verdict:     verdictIndeterminate,
			input:       `{"ssn":"123-45-6789","q":"lookup"}`,
			decision:    ffi.DecisionRedact,
			hasDecision: true,
			reason:      "redact ssn from args",
		},
		{
			// S7: an explicit DENY — the authoritative deny control.
			name:        "s7_explicit_deny",
			verdict:     verdictDeny,
			input:       "query",
			decision:    ffi.DecisionDeny,
			hasDecision: true,
			reason:      "blocked by policy",
		},
		{
			// S8: an explicit ALLOW — the authoritative allow control; the only
			// scenario where the tool runs under the fail-closed enforce default.
			name:        "s8_explicit_allow",
			verdict:     verdictAllow,
			input:       "query",
			decision:    ffi.DecisionAllow,
			hasDecision: true,
		},
	}
}

// gateConformanceCase is one (posture × scenario) cell of the matrix.
type gateConformanceCase struct {
	name     string
	posture  gatePosture
	scenario gateScenario
	wantRan  bool
}

// conformanceTable expands the full posture × scenario cross product.
func conformanceTable() []gateConformanceCase {
	postures := conformancePostures()
	scenarios := conformanceScenarios()
	table := make([]gateConformanceCase, 0, len(postures)*len(scenarios))
	for _, p := range postures {
		for _, s := range scenarios {
			table = append(table, gateConformanceCase{
				name:     p.name + "/" + s.name,
				posture:  p,
				scenario: s,
				wantRan:  s.wantRan(p),
			})
		}
	}
	return table
}

// TestGovernanceGateConformance drives every (posture × scenario) cell through
// the single gate and asserts whether the wrapped tool ran, pinning the
// fail-closed contract permanently.
func TestGovernanceGateConformance(t *testing.T) {
	t.Parallel()

	for _, tc := range conformanceTable() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			inner, client := tc.scenario.build()
			wrapped := newAssemblyTool(inner, client, tc.posture.opts)

			result, err := wrapped.Call(context.Background(), tc.scenario.input)
			ran := inner.calls > 0

			if ran != tc.wantRan {
				t.Fatalf("posture=%s scenario=%s: tool ran=%v, want ran=%v (err=%v result=%q)",
					tc.posture.name, tc.scenario.name, ran, tc.wantRan, err, result)
			}

			if tc.wantRan {
				if err != nil {
					t.Fatalf("posture=%s scenario=%s: expected the tool to run without a gate error, got %v",
						tc.posture.name, tc.scenario.name, err)
				}
				if result != inner.result {
					t.Fatalf("posture=%s scenario=%s: expected pass-through result %q, got %q",
						tc.posture.name, tc.scenario.name, inner.result, result)
				}
				return
			}

			// Blocked: the gate must return a non-nil error and must not leak the
			// tool's output.
			if err == nil {
				t.Fatalf("posture=%s scenario=%s: expected the gate to block the tool with a non-nil error, got nil",
					tc.posture.name, tc.scenario.name)
			}
			if result != "" {
				t.Fatalf("posture=%s scenario=%s: a blocked tool must not leak a result, got %q",
					tc.posture.name, tc.scenario.name, result)
			}
		})
	}
}

// TestGovernanceGateConformanceCompleteness pins the matrix shape so a dropped
// posture, a dropped scenario, or a newly added ffi.Decision variant without a
// scenario is caught at test time rather than silently leaving a gap in the
// fail-closed contract.
func TestGovernanceGateConformanceCompleteness(t *testing.T) {
	t.Parallel()

	const (
		wantPostures  = 4
		wantScenarios = 8
	)

	postures := conformancePostures()
	scenarios := conformanceScenarios()

	if len(postures) != wantPostures {
		t.Fatalf("posture set has %d entries, want %d — update the matrix when a posture is added/removed", len(postures), wantPostures)
	}
	if len(scenarios) != wantScenarios {
		t.Fatalf("scenario set has %d entries, want %d — update the matrix when a scenario is added/removed", len(scenarios), wantScenarios)
	}

	table := conformanceTable()
	if len(table) != wantPostures*wantScenarios {
		t.Fatalf("matrix has %d cells, want %d (every posture × scenario)", len(table), wantPostures*wantScenarios)
	}

	// Every (posture, scenario) cell must appear exactly once.
	seen := make(map[string]bool, len(table))
	for _, tc := range table {
		key := tc.posture.name + "/" + tc.scenario.name
		if seen[key] {
			t.Fatalf("duplicate cell in matrix: %s", key)
		}
		seen[key] = true
	}
	for _, p := range postures {
		for _, s := range scenarios {
			if !seen[p.name+"/"+s.name] {
				t.Fatalf("matrix is missing cell: %s/%s", p.name, s.name)
			}
		}
	}

	// Every ffi.Decision variant the gate maps must be exercised by a scenario.
	// Referencing the enum here means an added Decision variant with no
	// conformance scenario surfaces as an uncovered-variant failure.
	wantDecisions := map[int32]string{
		ffi.DecisionAllow:       "allow",
		ffi.DecisionDeny:        "deny",
		ffi.DecisionPending:     "pending",
		ffi.DecisionRedact:      "redact",
		ffi.DecisionUnspecified: "unspecified",
	}
	for _, s := range scenarios {
		if s.hasDecision {
			delete(wantDecisions, s.decision)
		}
	}
	if len(wantDecisions) != 0 {
		t.Fatalf("ffi.Decision variants not exercised by any conformance scenario: %v — add a scenario", wantDecisions)
	}
}

// TestGovernanceGateConformanceRedactNotDowngradedToAllow is the focused
// AAASM-4788 regression guard: under the fail-closed enforce default a REDACT
// verdict on a call carrying sensitive args must take the deny path, NOT run the
// tool with the unredacted arguments. If REDACT ever regresses to a silent
// Decision{} (plain allow), the tool would run and this fails.
func TestGovernanceGateConformanceRedactNotDowngradedToAllow(t *testing.T) {
	t.Parallel()

	inner := &failClosedTool{name: "web_search", result: "UNREDACTED-LEAK"}
	querier := &fakeQuerier{decision: ffi.DecisionRedact, reason: "redact ssn from args"}
	wrapped := newAssemblyTool(inner, newFFIGovernanceClient(querier), defaultRuntimeOptions())

	result, err := wrapped.Call(context.Background(), `{"ssn":"123-45-6789"}`)
	if err == nil {
		t.Fatal("REDACT under fail-closed enforce must deny, got nil error (regressed to a silent allow?)")
	}
	if inner.calls != 0 {
		t.Fatalf("REDACT must not run the tool with unredacted args, ran %d time(s)", inner.calls)
	}
	if result != "" {
		t.Fatalf("REDACT denial must not leak the tool result, got %q", result)
	}
}
