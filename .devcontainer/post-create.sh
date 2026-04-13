#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"

cd "${REPO_ROOT}"

mise trust --yes
mise install
mise exec -- go install github.com/entireio/roger-roger/cmd/roger-roger@latest
mise exec -- go install github.com/entireio/roger-roger/cmd/entire-agent-roger-roger@latest
