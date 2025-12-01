.PHONY: build clean test install

VERSION ?= dev

build:
	go build -ldflags "-X main.version=$(VERSION)" -o igloc ./cmd/igloc

install:
	go install -ldflags "-X main.version=$(VERSION)" ./cmd/igloc

test:
	go test -v ./...

clean:
	rm -f igloc
	rm -rf dist/
