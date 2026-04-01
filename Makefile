.PHONY: build-go build-dev

build-go:
	go build -o bin/nacha-lsp ./cmd/nacha-lsp

build-dev: build-go
	mkdir -p editors/vscode/bin
	cp bin/nacha-lsp editors/vscode/bin/nacha-lsp
