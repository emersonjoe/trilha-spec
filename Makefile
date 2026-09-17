.PHONY: test vet fmt build conformance

test: vet
	go test ./...

vet:
	test -z "$$(gofmt -l spec task agent ai execution mcp cmd)"
	go vet ./...

fmt:
	gofmt -w spec task agent ai execution mcp cmd

conformance:
	./scripts/check-execution-contract.sh

build:
	go build -o bin/trilha-spec ./cmd/trilha-spec
