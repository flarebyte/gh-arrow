#!/bin/sh
# Run from any directory. Requires Python 3 and flyb on PATH.
set -eu
cd "$(dirname "$0")/.."
sh scripts/validate-design.sh
flyb validate --config doc/design-meta
flyb generate markdown --config doc/design-meta
