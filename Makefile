.PHONY: test vet fmt build

test: vet
	go test ./...

vet:
	test -z "$$(gofmt -l spec task agent ai mcp cmd)"
	go vet ./...

fmt:
	gofmt -w spec task agent ai mcp cmd

build:
	go build -o bin/trilha-spec ./cmd/trilha-spec
