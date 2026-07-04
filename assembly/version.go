// Package assembly — Version constant.
//
// Exposes the SDK version as a string constant so callers can inspect
// which release of the Go SDK they are running against. Kept in sync
// with the git tag that publishes the module on `proxy.golang.org`.
//
// Refs: AAASM-1935 (alpha-1 pre-release dry-run)
package assembly

// Version is the published version of the agent-assembly Go SDK.
//
// Pre-release values follow the SemVer pre-release identifier form
// (`MAJOR.MINOR.PATCH-alpha.N`). The git tag mirrors this with a
// leading `v` (e.g. `v0.0.1-alpha.1`).
const Version = "0.0.1-rc.3"
