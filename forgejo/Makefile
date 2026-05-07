SOURCES := $(wildcard src/*.go) $(wildcard src/*.tmpl)

dist/forgejo.wasm: $(SOURCES) go.mod
	mkdir -p dist
	GOOS=wasip1 GOARCH=wasm go build -o dist/forgejo.wasm ./src

dist/forgejo.wasm.gz: dist/forgejo.wasm
	gzip -k -9 -f dist/forgejo.wasm
