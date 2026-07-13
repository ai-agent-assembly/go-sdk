// Package quickstartgen is the `go generate` anchor for the docs/quick-start.md
// per-framework tabs. It has no runtime code — it exists only to host the
// generate directive next to a stable import path, mirroring how
// internal/version anchors the metadata generator.
//
// The generator (scripts/gen-quickstart-tabs.go) renders the hextra tabs block
// from the vendored source data under metadata/quickstart/, and is drift-checked
// in CI (`go generate ./...` + `git diff`). See metadata/quickstart/README.md.
//
// Epic AAASM-4511 / AAASM-4515.
package quickstartgen

// Run `go generate ./...` from the repo root to regenerate the tabs block. The
// generator walks up from its working directory until it finds go.mod, so it
// works both when invoked directly and when invoked here by `go generate`.

//go:generate go run ../../scripts/gen-quickstart-tabs.go
