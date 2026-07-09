---
title: Tool policy
weight: 3
---

# Tool policy

Two tools with different risk profiles, one policy: a safe read is **allowed**
and a destructive delete is **denied**. This is the example to read to see what a
denied tool call looks like from your code.

## What this example demonstrates

- Defining multiple tools with different risk profiles.
- A `GovernanceClient` that applies per-tool allow/deny rules.
- An **allowed** tool call (`read-file`) completing normally.
- A **denied** tool call (`delete-file`) returning
  `assembly.PolicyViolationError`.
- How a policy decision surfaces through the SDK layer.

## The framework / library

**No framework.** Like [Basic agent]({{< relref "/examples/basic-agent" >}}),
this uses only the Go SDK
([`github.com/ai-agent-assembly/go-sdk`](https://pkg.go.dev/github.com/ai-agent-assembly/go-sdk))
and the standard library — the focus is the policy decision, not an agent
framework.

## How it works

1. `tools.go` defines `readFileTool` (safe, read-only) and `deleteFileTool`
   (destructive).
2. `policy.go` defines `policyClient` — a `GovernanceClient` whose `Check`
   consults a `blockedTools` map and returns `Decision{Denied: true, Reason: …}`
   for any blocked tool name. Here only `delete-file` is blocked.
3. `main.go` wraps both tools with `assembly.WrapTools` using `policyClient`,
   then calls each one.
4. The allowed call returns its result. The denied call returns an error that
   the caller matches with `errors.As(err, &pve)` against
   `*assembly.PolicyViolationError`.

This maps directly onto production: swap `policyClient` for a real gateway-backed
client and the same `WrapTools` logic controls your agent's tool access.

## Prerequisites & running it

Complete [Preparing the runtime environment]({{< relref "/examples/setup" >}})
first. Then:

```bash
cd agent-assembly-examples/go/tool-policy
go mod download
go run .
```

No gateway is required — the policy is applied in-process by the mock client.

## Code walkthrough

The policy lives in a small map and a `Check` method:

```go
// blockedTools lists tool names that this policy client will deny.
var blockedTools = map[string]string{
	"delete-file": "delete operations are blocked by policy",
}

func (p *policyClient) Check(_ context.Context, req assembly.CheckRequest) (assembly.Decision, error) {
	if reason, blocked := blockedTools[req.ToolName]; blocked {
		fmt.Printf("[policy] DENIED   tool=%s  reason=%q\n", req.ToolName, reason)
		return assembly.Decision{Denied: true, Reason: reason}, nil
	}
	fmt.Printf("[policy] ALLOWED  tool=%s\n", req.ToolName)
	return assembly.Decision{Denied: false}, nil
}
```

`main` wraps both tools and runs each one. The denied call is detected by type:

```go
func runTool(ctx context.Context, tool assembly.Tool, input string) {
	result, err := tool.Call(ctx, input)
	if err != nil {
		var pve *assembly.PolicyViolationError
		if errors.As(err, &pve) {
			fmt.Printf("[tool] error: %v\n", pve)
			return
		}
		fmt.Printf("[tool] unexpected error: %v\n", err)
		return
	}
	fmt.Printf("[tool] result: %s\n", result)
}
```

## Notes & caveats

> **A denied call returns a typed error, not a panic.** Match it with
> `errors.As(err, &pve)` where `pve` is a `*assembly.PolicyViolationError`. See
> [Handle allow/deny decisions and errors]({{< relref "/guides/handle-decisions-and-errors" >}})
> for the full set of typed errors and failure postures.

> **The deny rule is a local map here.** In production the decision comes from
> your gateway's policy, not a hard-coded `blockedTools` map — only the
> `GovernanceClient` changes; the wrapping and error handling are identical.

## Expected behavior

```text
[policy] client loaded: read-file=ALLOW, delete-file=DENY

[tool] calling: read-file  input="config.yaml"
[policy] ALLOWED  tool=read-file
[tool] result: (contents of config.yaml)

[tool] calling: delete-file  input="config.yaml"
[policy] DENIED   tool=delete-file  reason="delete operations are blocked by policy"
[tool] error: assembly: policy violation: tool=delete-file reason=delete operations are blocked by policy
```

`go test ./...` runs the same allow/deny paths offline.

## Links

- Example directory: [`go/tool-policy`](https://github.com/ai-agent-assembly/examples/tree/master/go/tool-policy)
- [`README.md`](https://github.com/ai-agent-assembly/examples/blob/master/go/tool-policy/README.md)
- Related: [Handle allow/deny decisions and errors]({{< relref "/guides/handle-decisions-and-errors" >}})
