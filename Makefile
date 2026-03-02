.PHONY: build test lint install clean

BINARY := beads-plan
BUILD_DIR := ./build

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/beads-plan/

test:
	go test ./... -v

lint:
	golangci-lint run ./...

install:
	go install ./cmd/beads-plan/

clean:
	rm -rf $(BUILD_DIR)
