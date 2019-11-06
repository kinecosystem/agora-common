#!/usr/bin/env bash
set -e

echo "" > coverage.txt
DIRS=$(go list -e ./... | grep -v vendor | grep -v mocks | grep -v cmd)

for d in $DIRS; do
    go vet $d
    go test -v -race -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done

