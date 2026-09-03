BINARY ?= agentforum
PKG    ?= ./cmd/agentforum

.PHONY: build test vet fmt tidy run clean

build:
	GOWORK=off go build -o $(BINARY) $(PKG)

test:
	GOWORK=off go test ./... -count=1

vet:
	GOWORK=off go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './ttmp/*')

tidy:
	GOWORK=off go mod tidy

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
