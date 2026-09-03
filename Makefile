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

# W7: web UI embed — build the frontend and stage it for go:embed.
.PHONY: build-web build-embed

build-web:
	pnpm --dir web install --frozen-lockfile
	pnpm --dir web build
	rm -rf internal/server/embed/public
	cp -r web/dist internal/server/embed/public

# Single binary with the UI embedded (requires build-web first).
build-embed: build-web
	GOWORK=off go build -tags embed -o $(BINARY) $(PKG)
