.PHONY: help build run test clean install

help:
	@echo "Available commands:"
	@echo "  make build      - Build the binary"
	@echo "  make run        - Run the application"
	@echo "  make test       - Run tests"
	@echo "  make clean      - Remove binary and build artifacts"
	@echo "  make install    - Install globally"
	@echo "  make fmt        - Format code"
	@echo "  make lint       - Run linter"

build:
	go build -o ruff-format-changes ./cmd/ruff-format-changes/

run: build
	./ruff-format-changes

test:
	go test -v ./...

clean:
	rm -f ruff-format-changes
	go clean

install: build
	go install ./cmd/ruff-format-changes

fmt:
	go fmt ./...

lint:
	go vet ./...
