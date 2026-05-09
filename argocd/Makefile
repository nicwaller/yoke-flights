SOURCES := $(wildcard src/*.go) $(wildcard src/crds/*.yaml)

dist/argocd.wasm: $(SOURCES) go.mod
	mkdir -p dist
	GOOS=wasip1 GOARCH=wasm go build -o dist/argocd.wasm ./src

dist/argocd.wasm.gz: dist/argocd.wasm
	gzip -k -9 -f dist/argocd.wasm
