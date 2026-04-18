#!/usr/bin/env bash
# Vendors React, ReactDOM, and Babel-standalone into internal/admin/dist/vendor/
# so the admin dashboard ships fully self-contained inside the single Go binary.
# Run once after cloning; the downloaded files are committed to the repo.
#
# Usage: ./scripts/vendor-admin-assets.sh
set -euo pipefail

REACT_VERSION=18.3.1
BABEL_VERSION=7.29.0
DEST="$(git rev-parse --show-toplevel)/internal/admin/dist/vendor"

mkdir -p "$DEST"
cd "$DEST"

fetch() {
  local url=$1 out=$2
  echo "  -> $out"
  curl -fsSL -o "$out" "$url"
}

echo "Vendoring admin dashboard dependencies into $DEST"
fetch "https://unpkg.com/react@${REACT_VERSION}/umd/react.production.min.js"          react.production.min.js
fetch "https://unpkg.com/react-dom@${REACT_VERSION}/umd/react-dom.production.min.js"  react-dom.production.min.js
fetch "https://unpkg.com/@babel/standalone@${BABEL_VERSION}/babel.min.js"             babel.min.js

echo
echo "Done. Sizes:"
wc -c *.js | awk '{printf "  %7d bytes  %s\n", $1, $2}'
