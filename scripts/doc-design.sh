#!/bin/sh
# Run from any directory. Requires Python 3 and flyb on PATH.
set -eu
cd "$(dirname "$0")/.."
python3 scripts/validate_design_csv.py doc/design-meta/examples --single-line \
  --unique decisions.csv:id --unique commands.csv:id --unique usecases.csv:id \
  --unique rules.csv:id --unique constraints.csv:id --unique acceptance.csv:id \
  --unique open-questions.csv:id
flyb validate --config doc/design-meta
flyb generate markdown --config doc/design-meta
