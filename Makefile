CMD := markov
OUT := ./bin/$(CMD)
GO_PACKAGE := github.com/ferux/phraseGen
GIT_TAG := $(shell git describe --abbrev=0 --tags)
GIT_REV := $(shell git rev-parse --short HEAD)

build:
	-rm $(OUT)
	go build -i -ldflags "-X $(GO_PACKAGE).Version=$(GIT_TAG) -X $(GO_PACKAGE).Revision=$(GIT_REV)" -o $(OUT) ./cmd/

run: build
	$(OUT)

check:
	go vet ./...
	errcheck ./...