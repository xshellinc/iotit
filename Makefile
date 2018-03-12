VERSION := $(shell git describe --tags)
.PHONY: build

build:
	go build
