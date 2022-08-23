BINARY?=docker-credentials-client
LDFLAGS="-s -w"
GO_BUILD=$(STATIC_FLAGS) go build -trimpath -ldflags=$(LDFLAGS)

all: cli

cli:
	@go build .

cross: ## Cross compile the server
	GOOS=linux  GOARCH=amd64 $(GO_BUILD) -o dist/linux-amd64/$(BINARY) .
	GOOS=linux  GOARCH=arm64 $(GO_BUILD) -o dist/linux-arm64/$(BINARY) .
	GOOS=darwin  GOARCH=amd64 $(GO_BUILD) -o dist/darwin-amd64/$(BINARY) .
	GOOS=darwin  GOARCH=arm64 $(GO_BUILD) -o dist/darwin-arm64/$(BINARY) .
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o dist/windows-amd64/$(BINARY).exe .

.PHONY: cli cross
