#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <binary> <arm64|amd64|x86_64>" >&2
  exit 2
fi

binary=$1
expected_arch=$2

if [[ ! -f "$binary" ]]; then
  echo "binary not found: $binary" >&2
  exit 1
fi

if [[ "$expected_arch" == "amd64" ]]; then
  expected_arch=x86_64
fi

actual_arch=$(lipo -archs "$binary")
if [[ "$actual_arch" != "$expected_arch" ]]; then
  echo "unexpected architecture: expected $expected_arch, got $actual_arch" >&2
  exit 1
fi

if ! otool -L "$binary" | grep -q '/System/Library/Frameworks/AppKit.framework/'; then
  echo "AppKit link not found: $binary" >&2
  exit 1
fi

minos=$(xcrun vtool -show-build "$binary" | awk '$1 == "minos" { print $2; exit }')
if [[ "$minos" != "12.0" ]]; then
  echo "unexpected minimum macOS version: expected 12.0, got ${minos:-missing}" >&2
  exit 1
fi

echo "verified $binary: arch=$actual_arch, minos=$minos, AppKit=linked"
