// Command nacha-lsp is a Language Server Protocol server for ACH
// (Automated Clearing House) files. It communicates over standard input/output
// using the JSON-RPC 2.0 transport defined by the LSP specification and
// provides diagnostics, hover, completions, formatting, document symbols, and
// quick-fix code actions for ACH files.
package main

import (
	"context"
	"log"

	"github.com/mattt/nacha-lsp/internal/handler"
	"github.com/owenrumney/go-lsp/server"
)

func main() {
	h := handler.New()
	srv := server.NewServer(h)
	if err := srv.Run(context.Background(), server.RunStdio()); err != nil {
		log.Fatal(err)
	}
}
