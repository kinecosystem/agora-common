all: generate

.PHONY: generate
generate:
    docker run -v $(pwd)/task:/proto -v $(pwd)/task:/genproto kinecosystem/protoc-gen-go
