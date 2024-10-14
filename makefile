BINDIR := bin
GO111MODULE := on
SHELL := /bin/bash
export
PLUGIN_NAME := k8sclustervitals
PLUGIN_VERSION := v0.0.1

.PHONY: build
.PHONY: format
.PHONY: lint
.PHONY: run
.PHONY: release

build:
	go build -o ./k8sclustervitals

run:
    # go version 1.19 required
    # go mod tidy
	go run main.go

format:
	go vet
	go fmt

lint:
	golangci-lint -v run

release:
	@echo "--> Deleting all builds"
	@rm -rf ./$(BINDIR)
	@echo "* linux64-amd"; env GOOS=linux GOARCH=amd64 go build -o $(BINDIR)/$(PLUGIN_NAME)-$(PLUGIN_VERSION)-linux-amd64 ./main.go
	@echo "* linux64-arm"; env GOOS=linux GOARCH=arm64 go build -o $(BINDIR)/$(PLUGIN_NAME)-$(PLUGIN_VERSION)-linux-arm64 ./main.go
	@echo "* darwin64-amd"; env GOOS=darwin GOARCH=amd64 go build -o $(BINDIR)/$(PLUGIN_NAME)-$(PLUGIN_VERSION)-darwin-amd64 ./main.go
	@echo "* darwin64-arm"; env GOOS=darwin GOARCH=arm64 go build -o $(BINDIR)/$(PLUGIN_NAME)-$(PLUGIN_VERSION)-darwin-arm64 ./main.go