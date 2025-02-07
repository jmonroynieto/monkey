GIT_TAG := $(shell git rev-parse --short HEAD)
.PHONY: build
build:
        go build --ldflags="-X main.CommitID=$(GIT_TAG)" .