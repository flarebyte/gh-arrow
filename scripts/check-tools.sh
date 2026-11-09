#!/bin/sh
set -eu
for tool in go gh python3 flyb; do
    if command -v "$tool" >/dev/null 2>&1; then
        printf '%s=true\n' "$tool"
    else
        printf '%s=false\n' "$tool"
    fi
done
if gh flarebyte --version >/dev/null 2>&1; then
    printf 'gh-flarebyte=true\n'
else
    printf 'gh-flarebyte=false\n'
fi
