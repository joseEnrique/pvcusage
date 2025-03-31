.PHONY: build test lint clean

# Binary name
BINARY_NAME=pvcusage

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint run

# Build flags
LDFLAGS=-ldflags "-w -s"

all: test build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) cmd/pvcusage/main.go

test:
	$(GOTEST) -v ./...

lint:
	$(GOLINT)

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

deps:
	$(GOMOD) tidy

run:
	$(GOBUILD) -o $(BINARY_NAME) cmd/pvcusage/main.go
	./$(BINARY_NAME)

.DEFAULT_GOAL := build 