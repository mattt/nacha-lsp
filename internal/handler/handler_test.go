package handler_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mattt/nacha-lsp/internal/handler"
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/servertest"
)

func TestDidSavePublishesDiagnostics(t *testing.T) {
	h := servertest.New(t, handler.New())
	uri := lsp.DocumentURI("file:///test.ach")

	if err := h.DidOpen(uri, "plaintext", "short line"); err != nil {
		t.Fatal(err)
	}
	if err := h.DidSave(uri); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	diags, err := h.WaitForDiagnostics(ctx, uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) == 0 {
		t.Fatal("expected diagnostics for invalid NACHA content")
	}
}

func TestDidSaveValidFileHasNoDiagnostics(t *testing.T) {
	h := servertest.New(t, handler.New())
	uri := lsp.DocumentURI("file:///valid.ach")
	text := validNachaFile()

	if err := h.DidOpen(uri, "plaintext", text); err != nil {
		t.Fatal(err)
	}
	if err := h.DidSave(uri); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	diags, err := h.WaitForDiagnostics(ctx, uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", diags)
	}
}

func TestHoverReturnsFieldDocumentation(t *testing.T) {
	h := servertest.New(t, handler.New())
	uri := lsp.DocumentURI("file:///hover.ach")
	text := validNachaFile()

	if err := h.DidOpen(uri, "plaintext", text); err != nil {
		t.Fatal(err)
	}

	hover, err := h.Hover(uri, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if hover == nil {
		t.Fatal("expected hover result")
	}
	if !strings.Contains(hover.Contents.Value, "Transaction Code") {
		t.Fatalf("expected transaction-code hover details, got: %s", hover.Contents.Value)
	}
}

func validNachaFile() string {
	return strings.Join([]string{
		recordLine('1', 'A'),
		recordLine('5', 'B'),
		recordLine('6', 'C'),
		recordLine('8', 'D'),
		recordLine('9', 'E'),
		recordLine('9', '9'),
		recordLine('9', '9'),
		recordLine('9', '9'),
		recordLine('9', '9'),
		recordLine('9', '9'),
	}, "\n")
}

func recordLine(recordType byte, fill byte) string {
	return string(recordType) + strings.Repeat(string(fill), 93)
}
