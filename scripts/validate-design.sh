#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
python3 scripts/validate_design_csv.py doc/design-meta/examples --single-line \
  --unique decisions.csv:id --unique commands.csv:id --unique usecases.csv:id \
  --unique rules.csv:id --unique constraints.csv:id --unique acceptance.csv:id \
  --unique open-questions.csv:id
