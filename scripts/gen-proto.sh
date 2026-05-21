#!/usr/bin/env bash
# Regenerate Go proto stubs from the sibling agent-assembly checkout.
#
# AAASM-1656 (PR-G of AAASM-1422). Mirrors the python-sdk + node-sdk
# regen scripts (scripts/gen_proto.py / scripts/gen-proto.mjs).
# Generates only the protos this SDK actually consumes today
# (policy + common). The output lives under `internal/proto/` and is
# committed so module consumers don't need protoc installed.
#
# Usage:
#   make proto
#   # or with a non-default sibling location:
#   AA_PROTO_DIR=/some/other/agent-assembly/proto make proto
#
# Requires protoc + protoc-gen-go + protoc-gen-go-grpc on PATH.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Per memory `project_sibling_repo_ci_pattern`: cross-repo deps use
# sibling-checkout via env vars. Default mirrors the workspace layout
# ($REPO_PARENT/agent-assembly/proto).
PROTO_DIR="${AA_PROTO_DIR:-${REPO_ROOT}/../agent-assembly/proto}"

if [ ! -d "${PROTO_DIR}" ]; then
  echo "error: proto dir ${PROTO_DIR} does not exist" >&2
  echo "Set AA_PROTO_DIR to the agent-assembly/proto location." >&2
  exit 1
fi

# Only generate the protos the SDK actually consumes. Keeping the set
# tight keeps the committed-stubs diff readable and avoids accidentally
# coupling the SDK to RPCs it doesn't use.
PROTOS=(common.proto policy.proto)

OUTPUT_DIR="${REPO_ROOT}/internal/proto"
mkdir -p "${OUTPUT_DIR}"

# Make sure go-installed protoc plugins are reachable (a fresh shell
# may not have $GOPATH/bin on PATH).
GOBIN="$(go env GOPATH)/bin"
export PATH="${PATH}:${GOBIN}"

for tool in protoc protoc-gen-go protoc-gen-go-grpc; do
  if ! command -v "${tool}" >/dev/null 2>&1; then
    echo "error: ${tool} not on PATH" >&2
    echo "Install with:" >&2
    echo "  brew install protobuf" >&2
    echo "  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" >&2
    echo "  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest" >&2
    exit 1
  fi
done

# `paths=source_relative` keeps the output directory flat (no nested
# `assembly/...` dirs); the M= mappings re-route the `assembly.policy.v1`
# and `assembly.common.v1` package paths back to our internal Go import
# path so the generated files compile in this repo.
GO_OPTS=(
  "--go_out=${OUTPUT_DIR}"
  "--go_opt=paths=source_relative"
  "--go_opt=Mcommon.proto=github.com/AI-agent-assembly/go-sdk/internal/proto"
  "--go_opt=Mpolicy.proto=github.com/AI-agent-assembly/go-sdk/internal/proto"
  "--go-grpc_out=${OUTPUT_DIR}"
  "--go-grpc_opt=paths=source_relative"
  "--go-grpc_opt=Mcommon.proto=github.com/AI-agent-assembly/go-sdk/internal/proto"
  "--go-grpc_opt=Mpolicy.proto=github.com/AI-agent-assembly/go-sdk/internal/proto"
)

echo "running: protoc --proto_path=${PROTO_DIR} ${GO_OPTS[*]} ${PROTOS[*]}"
protoc \
  "--proto_path=${PROTO_DIR}" \
  "${GO_OPTS[@]}" \
  "${PROTOS[@]/#/${PROTO_DIR}/}"

echo "ok: generated ${#PROTOS[@]} protos under ${OUTPUT_DIR}"
