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
}

func TestDidChangePublishesDiagnosticsWithoutSave(t *testing.T) {
	h := servertest.New(t, handler.New())
	uri := lsp.DocumentURI("file:///change.ach")

	if err := h.DidOpen(uri, "plaintext", validNachaFile()); err != nil {
		t.Fatal(err)
	}
	h.ClearDiagnostics()

	if err := h.DidChange(uri, 2, "gibberish"); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		diags := h.Diagnostics(uri)
		if len(diags) > 0 {
			return
		}
		if time.Now().After(deadline) {
			all := h.AllDiagnostics()
			t.Fatalf("expected diagnostics after typing invalid content; got %d publish notifications", len(all))
		}
		time.Sleep(10 * time.Millisecond)
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

func TestDocumentSymbolReturnsBatchAndEntryOutline(t *testing.T) {
	h := servertest.New(t, handler.New())
	uri := lsp.DocumentURI("file:///symbols.ach")

	if err := h.DidOpen(uri, "plaintext", validNachaFile()); err != nil {
		t.Fatal(err)
	}

	symbols, err := h.DocumentSymbol(uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) == 0 {
		t.Fatal("expected document symbols")
	}

	var batch *lsp.DocumentSymbol
	for i := range symbols {
		if strings.HasPrefix(symbols[i].Name, "Batch ") {
			batch = &symbols[i]
			break
		}
	}
	if batch == nil {
		t.Fatal("expected batch symbol")
	}
	if len(batch.Children) == 0 {
		t.Fatal("expected entry symbols under batch")
	}
}

func TestCompletionReturnsServiceClassSuggestions(t *testing.T) {
	h := servertest.New(t, handler.New())
	uri := lsp.DocumentURI("file:///completion.ach")

	if err := h.DidOpen(uri, "plaintext", validNachaFile()); err != nil {
		t.Fatal(err)
	}

	completion, err := h.Completion(uri, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if completion == nil || len(completion.Items) == 0 {
		t.Fatal("expected completion items")
	}

	if !hasCompletionLabel(completion.Items, "200") {
		t.Fatalf("expected service class completion options, got: %+v", completion.Items)
	}
}

func TestFormattingReturnsCanonicalEdit(t *testing.T) {
	h := servertest.New(t, handler.New())
	uri := lsp.DocumentURI("file:///formatting.ach")
	text := strings.ReplaceAll(validNachaFile(), "\n", "\r\n")

	if err := h.DidOpen(uri, "plaintext", text); err != nil {
		t.Fatal(err)
	}

	edits, err := h.Formatting(uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) == 0 {
		t.Fatal("expected formatting edits for CRLF input")
	}
	if strings.Contains(edits[0].NewText, "\r") {
		t.Fatalf("expected canonical LF output, got: %q", edits[0].NewText)
	}
}

func TestCodeActionReturnsLengthFix(t *testing.T) {
	h := servertest.New(t, handler.New())
	uri := lsp.DocumentURI("file:///codeactions.ach")

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
		t.Fatal("expected diagnostics for invalid content")
	}

	actions, err := h.CodeAction(&lsp.CodeActionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		Range:        diags[0].Range,
		Context:      lsp.CodeActionContext{Diagnostics: diags},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) == 0 {
		t.Fatal("expected quick fixes")
	}

	var found bool
	for _, action := range actions {
		if action.Title == "Normalize record length to 94 characters" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected record-length code action, got: %+v", actions)
	}
}

func hasCompletionLabel(items []lsp.CompletionItem, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}

func validNachaFile() string {
	return strings.Join([]string{
		makeFileHeader(),
		makeBatchHeader(),
		makeEntry(),
		makeBatchControl(),
		makeFileControl(),
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

func writeField(line []byte, start, end int, value string) {
	width := end - start + 1
	if len(value) > width {
		value = value[:width]
	}
	padded := value + strings.Repeat(" ", width-len(value))
	copy(line[start-1:end], []byte(padded))
}

func makeFileHeader() string {
	line := []byte(recordLine('1', ' '))
	writeField(line, 2, 3, "01")
	writeField(line, 35, 37, "094")
	writeField(line, 38, 39, "10")
	writeField(line, 40, 40, "1")
	return string(line)
}

func makeBatchHeader() string {
	line := []byte(recordLine('5', ' '))
	writeField(line, 2, 4, "200")
	writeField(line, 41, 50, "1234567890")
	writeField(line, 51, 53, "PPD")
	writeField(line, 80, 87, "12345678")
	writeField(line, 88, 94, "0000001")
	return string(line)
}

func makeEntry() string {
	line := []byte(recordLine('6', ' '))
	writeField(line, 2, 3, "22")
	writeField(line, 4, 11, "03130001")
	writeField(line, 30, 39, "0000001000")
	writeField(line, 80, 94, "123456780000001")
	return string(line)
}

func makeBatchControl() string {
	line := []byte(recordLine('8', ' '))
	writeField(line, 2, 4, "200")
	writeField(line, 5, 10, "000001")
	writeField(line, 11, 20, "0003130001")
	writeField(line, 21, 32, "000000000000")
	writeField(line, 33, 44, "000000001000")
	writeField(line, 45, 54, "1234567890")
	writeField(line, 80, 87, "12345678")
	writeField(line, 88, 94, "0000001")
	return string(line)
}

func makeFileControl() string {
	line := []byte(recordLine('9', ' '))
	writeField(line, 2, 7, "000001")
	writeField(line, 8, 13, "000001")
	writeField(line, 14, 21, "00000001")
	writeField(line, 22, 31, "0003130001")
	writeField(line, 32, 43, "000000000000")
	writeField(line, 44, 55, "000000001000")
	return string(line)
}
