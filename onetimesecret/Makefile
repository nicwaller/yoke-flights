SOURCES := $(wildcard src/*.go)

dist/onetimesecret.wasm: $(SOURCES) go.mod
	mkdir -p dist
	GOOS=wasip1 GOARCH=wasm go build -o dist/onetimesecret.wasm ./src

dist/onetimesecret.wasm.gz: dist/onetimesecret.wasm
	gzip -k -9 -f dist/onetimesecret.wasm
