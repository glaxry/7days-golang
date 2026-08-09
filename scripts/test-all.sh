#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

while IFS= read -r -d '' module; do
  module_dir="$(dirname "$module")"
  echo "==> go test ${module_dir#"$repo_root"/}"
  (cd "$module_dir" && go test -timeout 2m ./...)
done < <(find "$repo_root" -name go.mod -print0 | sort -z)

echo "==> verify generated HTML documentation"
(cd "$repo_root/tools/docsite" && go run . -root "$repo_root" -check)

wasm_output="$(mktemp -d)"
trap 'rm -rf -- "$wasm_output"' EXIT

for demo in "$repo_root"/demo-wasm/*; do
  [[ -d "$demo" ]] || continue
  name="$(basename "$demo")"
  echo "==> GOOS=js GOARCH=wasm go build $name"
  GOOS=js GOARCH=wasm go build -o "$wasm_output/$name.wasm" "$demo/main.go"
done

echo "All modules and WebAssembly demos passed."
