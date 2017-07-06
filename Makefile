VERSION := $(shell git describe --tags)
release:
	@goxc -n=iotit -d=./build \
		-bc="linux, windows, darwin" \
		-pv=$(VERSION) \
		-resources-exclude=README* \
		-tasks-=validate \
		-build-ldflags="-X main.version=$(VERSION)" \
		-q=true
