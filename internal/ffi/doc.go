// Package ffi provides the low-level transport between the assembly runtime
// and the local sidecar or native Rust governance library.
//
// It ships two interchangeable transport implementations selected at compile
// time by build tags:
//
//   - When built with -tags aa_ffi_go (and CGO_ENABLED=1), the cgo bridge in
//     cgo_bridge.go links the vendored native/aa-ffi-go library (a thin C-ABI
//     over the SHA-pinned aa-sdk-client) and routes events through the
//     authoritative runtime in-process.
//   - Otherwise, the fail-closed fallback in fallback_uds_nocgo.go performs no
//     transport: with no native binding there is no runtime to enforce, so it
//     reports ErrRuntimeUnavailable rather than silently allowing traffic. It
//     needs no C toolchain and is the default in CGO_ENABLED=0 builds.
//
// The binding_select_*.go files route callers through the active mode at
// build time so the rest of the SDK does not need to care which transport
// is in use.
//
// This package is internal: the public surface for governance lives in
// the assembly package. APIs here may change without notice.
package ffi
