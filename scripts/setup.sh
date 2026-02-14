#!/usr/bin/env bash
set -euo pipefail

echo "=== max-cloud dev environment setup ==="

# Check prerequisites
for cmd in go pnpm; do
  if ! command -v "$cmd" &> /dev/null; then
    echo "ERROR: $cmd is not installed"
    exit 1
  fi
done

echo "Go version: $(go version)"
echo "pnpm version: $(pnpm --version)"

# Install Node dependencies (turbo)
echo "Installing Node dependencies..."
pnpm install

# Tidy Go modules
echo "Tidying Go modules..."
for mod in apps/api apps/cli packages/shared; do
  echo "  go mod tidy in $mod"
  (cd "$mod" && go mod tidy)
done

# Sync Go workspace
echo "Syncing Go workspace..."
go work sync

echo "=== Setup complete ==="
echo "Run 'pnpm turbo build' to build all packages"
