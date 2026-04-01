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
