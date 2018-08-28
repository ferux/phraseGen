CMD := markov
OUT := ./bin/$(CMD)
GO_PACKAGE := github.com/ferux/phraseGen
GIT_TAG := $(shell git describe --abbrev=0 --tags)
GIT_REV := $(shell git rev-parse --short HEAD)

build:
	@echo "\tRemoving previous build"
	-rm $(OUT)
	@echo "\tBuilding new application"
	go build -i -ldflags "-X $(GO_PACKAGE).Version=$(GIT_TAG) -X $(GO_PACKAGE).Revision=$(GIT_REV)" -o $(OUT) ./cmd/

run: build
	@echo "\tStarting application $(CMD)"
	@$(OUT)

check:
	go vet ./...
	errcheck ./...

env:
	@cp .env.example .env