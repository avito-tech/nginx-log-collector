GOPACKAGES?=$(shell find . -name '*.go' -not -path "./vendor/*" -exec dirname {} \;| sort | uniq)
GOFILES?=$(shell find . -type f -name '*.go' -not -path "./vendor/*")

VERSION=$(shell date +%s)-$(shell git describe --abbrev=8 --dirty --always --tags)

all: help

.PHONY: help build fmt clean test coverage check vet lint

help:
	@echo "build          - build project"
	@echo "run            - run project with local config"
	@echo "deb            - build project & make deb package"
	@echo "fmt            - format application sources"
	@echo "clean          - remove artifacts"
	@echo "test           - run tests"
	@echo "coverage       - run tests with coverage"
	@echo "check          - check code style"
	@echo "vet            - run go vet"
	@echo "lint           - run golint"

fmt:
	go fmt $(GOPACKAGES)

build: clean
	go build -ldflags '-X main.Version=$(VERSION)' -o build/nginx-log-collector nginx-log-collector.go

deb: build
	go run -ldflags '-X main.Version=$(VERSION)' ./etc/make-deb-package.go

run:
	go run nginx-log-collector.go -config ./etc/examples/example_config.yaml

clean:
	go clean
	rm -rf ./build/

test: clean
	go test -v $(GOPACKAGES)

coverage: clean
	go test -v -cover $(GOPACKAGES)

check: vet lint

vet:
	go vet $(GOPACKAGES)

lint:
	ls $(GOFILES) | xargs -L1 golint
