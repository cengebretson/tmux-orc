.PHONY: build install clean test lint tidy fmt check

VERSION ?= dev
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o orc ./cmd/orc/...

install:
	go install $(LDFLAGS) ./cmd/orc/...

clean:
	rm -f orc

test:
	go test ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

fmt:
	gofmt -l -w ./cmd ./internal

check: lint test
