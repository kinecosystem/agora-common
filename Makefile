all: test build

.PHONY: deps
deps:
	@go get ./...

.PHONY: deps-clean
deps-clean:
	@go mod tidy

.PHONY: build
build:
	@./go-build.sh

.PHONY: test
test: build
test:
	@./go-test.sh
