.PHONY: fmt lint test proto native test-native

fmt:
	gofmt -w $(shell find . -name '*.go' -type f)

lint:
	golangci-lint run ./...

test:
	go test ./...

# Build the vendored native FFI shim (native/aa-ffi-go → libaa_ffi_go), a thin
# C-ABI over the SHA-pinned aa-sdk-client. Required before building/testing with
# the cgo binding (-tags aa_ffi_go). Needs a Rust toolchain + protoc on PATH.
native:
	cargo build --manifest-path native/aa-ffi-go/Cargo.toml

# Build the native shim, then run the suite against the real cgo binding. The
# cgo_bridge #cgo directives resolve the header + library from
# native/aa-ffi-go/{include,target/debug}.
test-native: native
	CGO_ENABLED=1 go test -tags aa_ffi_go ./...

# AAASM-1656 (PR-G): regenerate Go proto stubs from the sibling
# agent-assembly/proto checkout. Reads from $AA_PROTO_DIR (defaults
# to ../agent-assembly/proto, per the workspace's sibling-repo CI
# pattern). Requires protoc + protoc-gen-go + protoc-gen-go-grpc on
# PATH:
#   brew install protobuf
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
proto:
	@bash scripts/gen-proto.sh
