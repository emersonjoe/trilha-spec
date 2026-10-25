.PHONY: test vet fmt build conformance

test: vet
	go test ./...

vet:
	test -z "$$(gofmt -l .)"
	go vet ./...

fmt:
	gofmt -w .

conformance:
	./scripts/check-execution-contract.sh

build:
	go build -o bin/trilha-spec ./cmd/trilha-spec
