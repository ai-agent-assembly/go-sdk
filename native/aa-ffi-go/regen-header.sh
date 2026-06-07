#!/usr/bin/env bash
# Regenerate the committed C ABI header for the vendored aa-ffi-go shim from
# its Rust source. Run after any change to the C ABI in src/lib.rs.
#
# Requires cbindgen:
#   cargo install cbindgen --locked
#
# CI can call this then `git diff --exit-code include/aa_ffi_go.h` to guard
# against the header drifting from src/lib.rs.
set -euo pipefail

crate_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if ! command -v cbindgen >/dev/null 2>&1; then
  echo "error: cbindgen not found — install it with: cargo install cbindgen --locked" >&2
  exit 1
fi

cd "${crate_dir}"
cbindgen --config cbindgen.toml --crate aa-ffi-go --output include/aa_ffi_go.h
echo "regenerated ${crate_dir}/include/aa_ffi_go.h"
