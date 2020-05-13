CMD := markov
OUT := ./bin/$(CMD)
GO_PACKAGE := github.com/ferux/phraseGen
GIT_TAG := $(shell git describe --abbrev=0 --tags)
GIT_REV := $(shell git rev-parse --short HEAD)
ENV ?= dev

export CGO_ENABLED=0

build:
	@echo "Removing previous build"
	-rm $(OUT)
	@echo "**************************"
	@echo "Building new application"
	@go build -ldflags "-X $(GO_PACKAGE).Version=$(GIT_TAG) -X $(GO_PACKAGE).Revision=$(GIT_REV) -X $(GO_PACKAGE).Environment=$(ENV)" -o $(OUT) ./cmd/
	@echo "**************************"

build_linux: export GOOS := linux
build_linux: export GOARCH := amd64
build_linux: export OUT := bin/$(GOOS)_$(GOARCH)/$(CMD)
build_linux: export ENV := production
build_linux: build

run: build
	@echo "Starting application $(CMD)"
	@echo "**************************"
	@$(OUT)

check:
	go vet ./...
	errcheck ./...

env:
	@cp .env.example .env

