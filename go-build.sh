#!/usr/bin/env bash
set -e

DIRS=$(go list -e ./... | grep -v vendor | grep -v mocks | grep -v cmd)

for d in $DIRS; do
    go vet $d
    go build $d
done

