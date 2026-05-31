.PHONY: build install clean test lint tidy fmt check

build:
	go build -o orc ./cmd/orc/...

install:
	go install ./cmd/orc/...

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
