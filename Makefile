build:
	cp -rf "$(shell go env GOROOT)/misc/wasm/wasm_exec.js" wasm/
	GOOS=js GOARCH=wasm go get .
	GOOS=js GOARCH=wasm go build -o main.wasm

package:
	@set -euo pipefail; \
	bundle=$$(npm pack); \
	rm -fr dist; \
	mkdir -p dist; \
	mv "$${bundle}" dist;

test:
	go test $(shell find . -name '*_test.go')
