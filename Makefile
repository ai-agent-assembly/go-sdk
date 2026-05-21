.PHONY: fmt lint test proto

fmt:
	gofmt -w $(shell find . -name '*.go' -type f)

lint:
	golangci-lint run ./...

test:
	go test ./...

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
