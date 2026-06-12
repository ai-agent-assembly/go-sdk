---
title: Guides
weight: 3
---

# Guides

Task-first walkthroughs that show the SDK doing real work. Each guide is
self-contained and grounded in the actual `assembly` package API — start with
whichever matches what you're building.

| Guide | What you'll do |
|---|---|
| [Govern an agent's tools]({{< relref "/guides/govern-an-agents-tools" >}}) | Implement the `Tool` interface, wrap a tool slice, and run governed calls end to end. |
| [Integrate with a framework]({{< relref "/guides/framework-integration" >}}) | Plug the SDK into an existing agent framework (langchaingo-style chains) and propagate agent lineage across hops. |
| [Handle allow/deny decisions and errors]({{< relref "/guides/handle-decisions-and-errors" >}}) | Match the typed errors the SDK returns, react to deny / approval / fail-closed outcomes, and choose a failure posture. |

If a topic you need isn't here yet, open an issue against the
[go-sdk repo](https://github.com/ai-agent-assembly/go-sdk/issues).
