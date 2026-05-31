.PHONY: build install clean test lint tidy check

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

check: lint test
