---
title: CLI runtime integration
weight: 5
---

# CLI runtime integration

Auto-start the `aasm` CLI runtime sidecar from a Go agent workflow, and fall back
gracefully to offline governance when the binary isn't installed. This is the
example that touches the `aasm` binary directly.

## What this example demonstrates

- Using `assembly.InitAssembly` to probe for and auto-start the `aasm` sidecar.
- Handling `assembly.ErrBinaryNotFound` gracefully when `aasm` isn't installed.
- Falling back to offline mock governance when the sidecar is unavailable.
- Using `scripts/run-with-aasm.sh` to orchestrate sidecar startup before running.
- The relationship between the `aasm` binary, the SDK, and the governance layer.

## The framework / library

**No agent framework** — the moving part here is the **`aasm` CLI runtime**, the
sidecar process from the
[agent-assembly](https://github.com/ai-agent-assembly/agent-assembly) core repo.
The example uses the SDK's
[`assembly.InitAssembly`](https://pkg.go.dev/github.com/ai-agent-assembly/go-sdk/assembly#InitAssembly)
and [`assembly.ErrBinaryNotFound`](https://pkg.go.dev/github.com/ai-agent-assembly/go-sdk/assembly#ErrBinaryNotFound)
to manage that sidecar.

## How it works

1. `main.go` calls `assembly.InitAssembly("cli-runtime-demo")` — this probes
   `127.0.0.1:7878` (`assembly.DefaultRuntimeHost` / `assembly.DefaultPort`) and,
   if the sidecar isn't already running, finds and spawns the `aasm` binary.
2. If `assembly.ErrBinaryNotFound` is returned (detected with `errors.Is`), the
   example logs an install hint and falls back to the offline mock client — a
   non-fatal condition.
3. If the sidecar is reachable, `buildGovernanceClient` logs the sidecar address.
   For this example it still returns the offline mock client; in production you
   would swap that for a real transport-backed `GovernanceClient`.
4. A governed `echoTool` call runs through `assembly.WrapTools`.
5. `scripts/run-with-aasm.sh` handles sidecar startup orchestration for CI: it
   runs `aasm serve --port`, waits for the port to open, then runs the example.

## Prerequisites & running it

Complete [Preparing the runtime environment]({{< relref "/examples/setup" >}})
first — including the optional `aasm` install if you want the full sidecar path.
Then:

```bash
cd agent-assembly-examples/go/cli-runtime-integration
go mod download
```

**Fallback mode (always works, no `aasm` needed):**

```bash
go run .
```

**Full sidecar mode (requires `aasm` on `PATH`):**

```bash
bash scripts/run-with-aasm.sh
```

The script honours `AASM_PORT` (default `7878`) and `WAIT_SECONDS` (default `5`).

## Code walkthrough

`startSidecar` calls `InitAssembly` and treats a missing binary as recoverable:

```go
func startSidecar() bool {
	fmt.Println("[runtime] probing for aasm sidecar...")

	err := assembly.InitAssembly("cli-runtime-demo")
	if err != nil {
		if errors.Is(err, assembly.ErrBinaryNotFound) {
			fmt.Println("[runtime] aasm binary not found — continuing in offline fallback mode")
			fmt.Println("[runtime] install aasm: brew install agent-assembly/tap/aasm")
			return false
		}
		log.Printf("[runtime] sidecar init warning: %v", err)
		return false
	}

	fmt.Printf("[runtime] sidecar ready at %s:%d\n", assembly.DefaultRuntimeHost, assembly.DefaultPort)
	return true
}
```

`buildGovernanceClient` logs which path was taken, then returns the mock client
the example wraps its tool with:

```go
func buildGovernanceClient(sidecarRunning bool) assembly.GovernanceClient {
	if sidecarRunning {
		fmt.Printf("[runtime] sidecar is running — governance calls will reach %s:%d\n",
			assembly.DefaultRuntimeHost, assembly.DefaultPort)
		fmt.Println("[runtime] using offline mock client for this example (swap for real transport in production)")
	} else {
		fmt.Println("[runtime] using offline mock governance client")
	}
	return &mockClient{}
}
```

The `scripts/run-with-aasm.sh` helper starts the sidecar, waits for the port,
runs the example, and stops the sidecar on exit:

```bash
aasm serve --port "$AASM_PORT" >>"$AASM_LOG" 2>&1 &
AASM_PID=$!
# wait for 127.0.0.1:$AASM_PORT to open, then:
go run .
```

## Notes & caveats

> **The `aasm` binary is not bundled with the Go SDK.** It must be installed
> separately (Homebrew, the curl installer, or `go install`). See
> [Preparing the runtime environment]({{< relref "/examples/setup" >}}) for the
> install commands.

> **Even in full sidecar mode, this example uses the mock client** for the actual
> governance calls — it demonstrates *starting* the sidecar, not speaking a real
> transport to it. Swap `mockClient` for a transport-backed `GovernanceClient` to
> route decisions through the running sidecar in production.

> **Troubleshooting:** if `sidecar failed to start`, check `.aasm-runtime.log` in
> the working directory. If port `7878` is already in use, another `aasm`
> instance is running and the example connects to it.

## Expected behavior

**Fallback mode (no `aasm` binary):**

```text
[runtime] probing for aasm sidecar...
[runtime] aasm binary not found — continuing in offline fallback mode
[runtime] install aasm: brew install agent-assembly/tap/aasm
[runtime] using offline mock governance client
[assembly] governance: ALLOWED  tool=echo input="Hello from the CLI runtime!"
[assembly] tool result: Hello from the CLI runtime!
```

**With `aasm` installed:**

```text
[runtime] probing for aasm sidecar...
[runtime] sidecar ready at 127.0.0.1:7878
[runtime] sidecar is running — governance calls will reach 127.0.0.1:7878
[runtime] using offline mock client for this example (swap for real transport in production)
[assembly] governance: ALLOWED  tool=echo input="Hello from the CLI runtime!"
[assembly] tool result: Hello from the CLI runtime!
```

`go test ./...` verifies the `ErrBinaryNotFound` fallback path offline, without
needing `aasm` installed.

## Links

- Example directory: [`go/cli-runtime-integration`](https://github.com/ai-agent-assembly/agent-assembly-examples/tree/master/go/cli-runtime-integration)
- [`README.md`](https://github.com/ai-agent-assembly/agent-assembly-examples/blob/master/go/cli-runtime-integration/README.md)
- [`assembly.InitAssembly`](https://pkg.go.dev/github.com/ai-agent-assembly/go-sdk/assembly#InitAssembly) · [`assembly.ErrBinaryNotFound`](https://pkg.go.dev/github.com/ai-agent-assembly/go-sdk/assembly#ErrBinaryNotFound)
- Related: [Troubleshooting]({{< relref "/troubleshooting" >}})
