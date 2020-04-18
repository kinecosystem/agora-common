all: test build

.PHONY: deps
deps:
	@go get ./...

.PHONY: build
build:
	@./go-build.sh

.PHONY: test
test: build
test:
	@./go-test.sh
