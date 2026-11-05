BINARY_NAME=archguard
VERSION=0.1.0
BUILD_DIR=bin

.PHONY: all build clean test run install

all: test build

build:
	@mkdir -p $(BUILD_DIR)
	@echo "==> Building $(BINARY_NAME) v$(VERSION)..."
	go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/archguard

test:
	@echo "==> Running test suite..."
	go test -v ./...

run: build
	@./$(BUILD_DIR)/$(BINARY_NAME)

install: build
	@echo "==> Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
