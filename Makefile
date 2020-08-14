VERSION := $(shell git describe --always |sed -e "s/^v//")

build:
	mkdir -p build
	go build -v -ldflags "-s -w -X main.version=$(VERSION)" -o build/moustachos.bin main.go

clean:
	@echo "Cleaning up workspace"
	@rm -rf build

.PHONY: build clean
