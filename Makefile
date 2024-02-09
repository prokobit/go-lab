BINARY_NAME:=hello
GOOS:=$(shell go env GOOS)

all: help

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## fmt: format code
.PHONY: fmt
fmt:
	go fmt ./...

## lint: run linter
.PHONY: lint
lint:
	golangci-lint run

## clean: Remove build related file
.PHONY: clean
clean: 
	rm -fr ./bin

## vendor: Copy of all packages needed to support builds and tests in the vendor directory
.PHONY: vendor
vendor: 
	go mod vendor

## tidy: tidy modfile
.PHONY: tidy
tidy:
	go mod tidy -v

## build: build the application
.PHONY: build
build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/${BINARY_NAME}-linux  cmd/${BINARY_NAME}/main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/${BINARY_NAME}-darwin cmd/${BINARY_NAME}/main.go

## run: run the  application
.PHONY: run
run: build
	bin/${BINARY_NAME}-${GOOS}

## build-image: build image of the application
.PHONY: build-image
build-image:
	podman build -t ${BINARY_NAME} .

## run-image: run the application in container
.PHONY: run-image
run-image: build-image
	podman run -d -p 8080:8080 ${BINARY_NAME}
