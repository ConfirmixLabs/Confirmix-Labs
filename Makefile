.PHONY: build run-node run-example clean

BINARY_NAME=blockchain
EXAMPLE_BINARY=example

all: build

build:
	go build -o $(BINARY_NAME) ./cmd/blockchain

build-example:
	go build -o $(EXAMPLE_BINARY) ./examples

run-node:
	./$(BINARY_NAME) node --validator=true --poh-verify=true

run-example:
	./$(EXAMPLE_BINARY)

clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(EXAMPLE_BINARY)

tidy:
	go mod tidy

test:
	go test ./... 