SOURCES := $(wildcard src/*.go)

dist/cloudflare-tunnel.wasm: $(SOURCES) go.mod
	mkdir -p dist
	GOOS=wasip1 GOARCH=wasm go build -o dist/cloudflare-tunnel.wasm ./src

dist/cloudflare-tunnel.wasm.gz: dist/cloudflare-tunnel.wasm
	gzip -k -9 -f dist/cloudflare-tunnel.wasm
