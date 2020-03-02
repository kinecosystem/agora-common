all: genproto

.PHONY: genproto
genproto:
	rm -rf genproto
	docker run -v $(PWD):/proto -v $(PWD)/genproto:/genproto mfycheng/protoc-gen-go
	mv genproto/* .
	rm -rf genproto
