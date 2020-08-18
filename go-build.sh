#!/usr/bin/env bash
set -e

golangci-lint run --enable unparam,gofmt,whitespace,golint \
    --exclude "Use protoreflect.MessageDescriptor.FullName instead." \
    --exclude "Use the .google.golang.org/protobuf/proto. package instead."

DIRS=$(go list -e ./... | grep -v vendor | grep -v mocks | grep -v cmd)

for d in $DIRS; do
    go vet $d
    go build $d
done

