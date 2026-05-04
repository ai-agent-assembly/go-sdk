---
title: Architecture
weight: 2
---

# Architecture

This page describes how `go-sdk` is organised internally — the module layout,
the dual-mode FFI bridge to the Rust governance library, the HTTP and gRPC
interceptor flow, the context-propagation design, and how tool wrapping
threads governance checks around your agent's tool calls. Read it after
[Getting Started](getting-started/) when you want to know *why* the SDK is
shaped the way it is.
