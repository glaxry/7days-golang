GOROOT := $(shell go env GOROOT)

.PHONY: all build serve clean

all: serve

build: static/main.wasm static/wasm_exec.js

serve: build
	go run ../serve.go -port 9999 .

static:
	mkdir -p static

static/wasm_exec.js: | static
	cp "$(GOROOT)/lib/wasm/wasm_exec.js" $@

static/main.wasm: main.go | static
	GOOS=js GOARCH=wasm go build -o $@ main.go

clean:
	rm -rf static
