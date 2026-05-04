// Package ffi provides the low-level transport between the assembly runtime
// and the local sidecar or native Rust governance library.
//
// It ships two interchangeable transport implementations selected at compile
// time by build tags:
//
//   - When built with -tags aa_ffi_go (and CGO_ENABLED=1), the cgo bridge in
//     cgo_bridge.go links against the libaa_ffi_go shared library and calls
//     into the Rust runtime in-process for the lowest possible latency.
//   - Otherwise, the pure-Go fallback in fallback_uds_nocgo.go connects to
//     the sidecar over a Unix domain socket. This path needs no C toolchain
//     and is the default in CGO_ENABLED=0 builds and CI lanes.
//
// The binding_select_*.go files route callers through the active mode at
// build time so the rest of the SDK does not need to care which transport
// is in use.
//
// This package is internal: the public surface for governance lives in
// the assembly package. APIs here may change without notice.
package ffi
