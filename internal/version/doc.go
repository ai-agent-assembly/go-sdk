// Package version exposes generated constants for the Go SDK's authoritative
// module / protocol / URL metadata. The values are populated by
// scripts/gen-metadata.go from go.mod, the top-level VERSION file, and
// metadata/sdk.yaml — never edit metadata.go by hand.
//
// This package is under internal/ and has no public API surface. External
// callers should continue to read go.mod / VERSION directly rather than
// depending on these constants.
//
// AAASM-4311 / Epic AAASM-4309.
package version

// Run `go generate ./...` from the repo root to regenerate metadata.go. The
// generator walks up from its working directory until it finds go.mod, so it
// works both when invoked directly (`go run scripts/gen-metadata.go` from the
// repo root) and when invoked here by `go generate` from this package's dir.

//go:generate go run ../../scripts/gen-metadata.go
