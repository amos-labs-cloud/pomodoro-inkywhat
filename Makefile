.PHONY: all build clean

all: clean deps build

clean:
	rm -rf build

deps:
	go mod tidy
	go mod vendor

build: deps
	rm -rf build/binaries
	mkdir -p build/binaries
	CGO_CFLAGS_ALLOW='-Xpreprocessor' go build -o build/binaries/piw main.go

run:
	CGO_CFLAGS_ALLOW='-Xpreprocessor' go run main.go