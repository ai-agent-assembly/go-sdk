---
title: Examples
weight: 8
---

# Examples

Complete, runnable Go examples live in the
[agent-assembly-examples](https://github.com/ai-agent-assembly/agent-assembly-examples/tree/master/go)
repository. Each is a self-contained Go module with its own `README.md` covering
setup, configuration, and how to run it — start with whichever matches what
you're building.

## How the SDK wraps tool calls

Every example follows the same shape as the [Quick Start]({{< relref "/quick-start" >}}):
import the SDK, initialise the runtime, then wrap your tools so each call is
checked against the gateway policy before execution and recorded after.

```go
import (
    "context"
    "log"

    "github.com/ai-agent-assembly/go-sdk/assembly"
)

func main() {
    ctx := assembly.WithAgentID(context.Background(), "my-agent")

    a, err := assembly.Init(ctx,
        assembly.WithGatewayURL("https://gateway.example.com"),
        assembly.WithAPIKey("..."),
    )
    if err != nil {
        log.Fatalf("init: %v", err)
    }
    defer a.Close()

    governed := assembly.WrapTools(myTools, nil)
    // hand `governed` to your agent — every Call is now governed
}
```

## Available examples

| Example | What it demonstrates |
|---|---|
| [basic-agent](https://github.com/ai-agent-assembly/agent-assembly-examples/tree/master/go/basic-agent) | A minimal governed agent — initialise the SDK and execute a single governed tool call. |
| [tool-policy](https://github.com/ai-agent-assembly/agent-assembly-examples/tree/master/go/tool-policy) | Tool-policy enforcement with no framework — explicit allow and deny across tools with different risk profiles. |
| [langchaingo](https://github.com/ai-agent-assembly/agent-assembly-examples/tree/master/go/langchaingo) | [LangChainGo](https://github.com/tmc/langchaingo) integration — govern a LangChainGo agent's tool calls through Agent Assembly. |
| [cli-runtime-integration](https://github.com/ai-agent-assembly/agent-assembly-examples/tree/master/go/cli-runtime-integration) | Integrating with the `aasm` CLI runtime — auto-start the `aasm` sidecar from a Go agent workflow. |

Each example's `README.md` has the full step-by-step instructions for running it.

## Where to next

- [Quick Start]({{< relref "/quick-start" >}}) — the three-step path these examples build on.
- [Guides]({{< relref "/guides" >}}) — task-first walkthroughs of wrapping tools, framework integration, and handling decisions.
- [Configuration]({{< relref "/configuration" >}}) — every `Init` option, defaults, and enforcement modes.
