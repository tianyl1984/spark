BINARY := spark
BIN_DIR := bin

.PHONY: build run test fmt vet tidy clean install

install:
	./install.sh

build:
	go build -o $(BIN_DIR)/$(BINARY) .

run:
	go run .

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)
