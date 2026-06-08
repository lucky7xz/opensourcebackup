#!/usr/bin/env bash
#
# Activate the repo's versioned git hooks (scripts/hooks/).
# Run once after cloning:  bash scripts/install-hooks.sh
#
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

git config core.hooksPath scripts/hooks
chmod +x scripts/hooks/* 2>/dev/null || true

echo "✓ core.hooksPath → scripts/hooks"
echo "✓ pre-push quality gate active"
echo ""
echo "  The gate runs on every 'git push':"
echo "    1. secret / private-data scan"
echo "    2. go vet   (internal + cmd)"
echo "    3. go test  (internal + cmd)"
echo "    4. tsc --noEmit (when web/ changed)"
echo ""
echo "  Emergency bypass:  git push --no-verify"
