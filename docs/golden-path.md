---
title: "Start Here: The Golden Path"
weight: 1
---

# Start Here: The Golden Path

New to the Go SDK? Read this first. It is the **map**, not the territory: it lays
out the whole end-to-end journey in order and points you at the canonical page
that owns each step. Follow the numbered arc top to bottom the first time
through, then keep this page as your table of contents.

The same twelve-step arc appears in the
[Python](https://docs.agent-assembly.com/python-sdk/) and
[Node](https://docs.agent-assembly.com/node-sdk/) SDK docs — so whichever
language you standardise on, the shape of the journey is identical.

{{< callout type="info" >}}
This is a wayfinding page. It **links** to the pages that own each command and
API rather than repeating them, so nothing here can drift out of sync. Where a
step has a known limitation on Go today, this page says so and sends you to the
page that has the current detail.
{{< /callout >}}

## The two personas

There are two ways through the platform, and this page is the on-ramp for the
first:

- **Developer arc** — *this page*: add the SDK to your Go agent and govern its
  tool calls from inside your process.
- **Operator arc** — running and governing the gateway, policies, budgets, and
  audit from the outside. That journey lives on the shared
  [docs hub](https://docs.agent-assembly.com/): follow the operator /
  end-to-end governance walkthrough there.

The two meet in the middle: the policies an operator authors are the same ones
the SDK enforces in step 6.

## The journey

### 1. What you'll achieve

A **governed AI agent whose tool calls you can allow, deny, observe, and
control — without changing the agent's logic.** You wrap your existing tool
slice once; from then on every call is checked against your gateway's policy
before it runs and recorded after it finishes. The end state is an agent that
behaves exactly as before, except the actions it takes are now governable.

### 2. Before you begin

Confirm the prerequisites — a supported Go toolchain and, for the register
handshake, the opt-in native binding — and understand what the SDK is and isn't.
→ [go-sdk introduction]({{< relref "/" >}}) and the
[Quick Start]({{< relref "/quick-start" >}}) prerequisites.

### 3. Install

Add the `assembly` package to your module. The default build is pure-Go and
compiles with `CGO_ENABLED=0`.
→ [Quick Start — install]({{< relref "/quick-start" >}}).

### 4. Connect to the gateway

Point the SDK at your gateway (or let local mode auto-discover one) and set your
agent identity. Every `Init` option, the resolution order for the gateway URL and
API key, and the enforcement modes are here.
→ [Configuration]({{< relref "/configuration" >}}).

### 5. Your first governed action (allowed)

Wrap a tool, call it, and watch an **allowed** call pass through — the inner tool
runs and a result is recorded.
→ [Quick Start]({{< relref "/quick-start" >}}).

{{< callout type="warning" >}}
**Be aware of one Go-specific gap before you rely on the dashboard.** Wrapping
and governing tool calls works on the default pure-Go build, but the *register*
handshake that makes your agent appear in the dashboard runs only under the
opt-in native binding (`-tags aa_ffi_go`, `CGO_ENABLED=1`), which is not yet
published for a plain `go get`. The [Quick Start]({{< relref "/quick-start" >}})
explains exactly what does and doesn't work today — read its note before
step 5.
{{< /callout >}}

### 6. See a policy denial

Now trigger a **denied** call. The wrapper stops the inner tool from running and
returns a typed `*PolicyViolationError` carrying the tool name and the gateway's
reason. This is the governance actually taking effect.
→ [Handle allow/deny decisions and errors]({{< relref "/guides/handle-decisions-and-errors" >}}).

### 7. Approvals (human-in-the-loop)

Some calls come back **pending** — they need an out-of-band human decision. The
wrapper blocks on approval and then either runs the tool or returns a denial. The
same guide covers the `Pending` decision path.
→ [Handle allow/deny decisions and errors]({{< relref "/guides/handle-decisions-and-errors#the-decision-the-gateway-returns" >}}).

### 8. Observe your agent

Every governed call is recorded. See the audit trail and dashboard, and use
`aasm audit` from the operator side, to watch what your agent actually did.
→ [Docs hub — observability & audit](https://docs.agent-assembly.com/).

### 9. Tune governance

Change a policy and watch behaviour change: a call that was allowed becomes
denied (or the reverse) with no code change on your side. Policy authoring and
the policy reference live on the hub; the SDK-side settings that interact with it
(enforcement mode, fail-closed vs fail-open) are in Configuration.
→ [Configuration]({{< relref "/configuration" >}}) and the
[policy reference on the hub](https://docs.agent-assembly.com/).

### 10. Operate it

Run and govern the platform from the operator side — start the gateway, manage
policies and budgets, inspect topology.
→ [Docs hub — operator path](https://docs.agent-assembly.com/).

### 11. Explore framework examples

See the arc applied to real, runnable Go code: a basic agent, per-tool
allow/deny, a LangChainGo integration, and `aasm` CLI runtime integration.
→ [Examples]({{< relref "/examples" >}}).

### 12. You've experienced the core value

You've taken a plain Go agent, wrapped its tools, seen a call allowed and a call
denied, observed the audit trail, and changed a policy — all without touching the
agent's own logic. That is the whole point: **governance as a checkpoint in front
of tool calls, not a rewrite of your agent.**

**What's next:**

- More frameworks and patterns → [Guides]({{< relref "/guides" >}}) and
  [Examples]({{< relref "/examples" >}}).
- The operator / end-to-end governance walkthrough and cross-cutting reference →
  [docs hub](https://docs.agent-assembly.com/).
- Sibling SDKs on the same control plane →
  [Python](https://docs.agent-assembly.com/python-sdk/) and
  [Node](https://docs.agent-assembly.com/node-sdk/).
</content>
</invoke>
