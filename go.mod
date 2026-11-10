module github.com/ai-agent-assembly/go-sdk

go 1.26.0

// Pin the build/scan toolchain to go1.26.10 so govulncheck evaluates the stdlib
// against a patched one. AAASM-6454: go1.26.6 left ten stdlib advisories
// REACHABLE (GO-2026-6603/6605/6607/6608/6610/6611/6612/6613/6617 fixed in
// go1.26.9, GO-2026-6809 crypto/x509 fixed in go1.26.10). Earlier pins closed
// GO-2026-5856 (go1.26.5) and GO-2026-6218/6090/5972/5026 (go1.26.6).
// The `go` line stays at 1.26.0 (the language/min-version floor); this only raises
// the toolchain actually used to build and scan.
toolchain go1.26.10

require (
	github.com/oklog/ulid/v2 v2.1.2
	go.opentelemetry.io/otel/trace v1.46.0
	golang.org/x/tools v0.52.0
	google.golang.org/grpc v1.84.0
	google.golang.org/protobuf v1.36.12
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	go.opentelemetry.io/otel v1.46.0 // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/telemetry v0.0.0-20261009151841-97e8ea77aa2a // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/vuln v1.6.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
)

tool golang.org/x/vuln/cmd/govulncheck
