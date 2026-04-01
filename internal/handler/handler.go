package handler

import (
	"context"

	"github.com/mattt/nacha-lsp/internal/nacha"
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/server"
)

type Handler struct {
	documents map[lsp.DocumentURI]string
	client    *server.Client
}

func New() *Handler {
	return &Handler{
		documents: make(map[lsp.DocumentURI]string),
	}
}

func (h *Handler) SetClient(client *server.Client) {
	h.client = client
}

func (h *Handler) Initialize(_ context.Context, _ *lsp.InitializeParams) (*lsp.InitializeResult, error) {
	return &lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptions{
				OpenClose: boolPtr(true),
				Change:    lsp.SyncFull,
				Save:      &lsp.SaveOptions{IncludeText: boolPtr(true)},
			},
			HoverProvider: boolPtr(true),
		},
		ServerInfo: &lsp.ServerInfo{
			Name:    "nacha-lsp",
			Version: "0.1.0",
		},
	}, nil
}

func (h *Handler) Shutdown(_ context.Context) error { return nil }

func (h *Handler) DidOpen(_ context.Context, params *lsp.DidOpenTextDocumentParams) error {
	h.documents[params.TextDocument.URI] = params.TextDocument.Text
	return nil
}

func (h *Handler) DidChange(_ context.Context, params *lsp.DidChangeTextDocumentParams) error {
	if len(params.ContentChanges) > 0 {
		h.documents[params.TextDocument.URI] = params.ContentChanges[len(params.ContentChanges)-1].Text
	}
	return nil
}

func (h *Handler) DidClose(_ context.Context, params *lsp.DidCloseTextDocumentParams) error {
	delete(h.documents, params.TextDocument.URI)
	return nil
}

func (h *Handler) DidSave(ctx context.Context, params *lsp.DidSaveTextDocumentParams) error {
	if params.Text != nil {
		h.documents[params.TextDocument.URI] = *params.Text
	}

	text := h.documents[params.TextDocument.URI]
	rawDiagnostics := nacha.Validate(text)
	diagnostics := make([]lsp.Diagnostic, 0, len(rawDiagnostics))
	for _, diag := range rawDiagnostics {
		severity := lsp.SeverityError
		if diag.Severity == nacha.SeverityWarning {
			severity = lsp.SeverityWarning
		}
		diagnostics = append(diagnostics, lsp.Diagnostic{
			Range: lsp.Range{
				Start: lsp.Position{Line: diag.Line, Character: diag.StartCharacter},
				End:   lsp.Position{Line: diag.Line, Character: diag.EndCharacter},
			},
			Severity: &severity,
			Source:   "nacha-lsp",
			Message:  diag.Message,
		})
	}

	return h.client.PublishDiagnostics(ctx, &lsp.PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: diagnostics,
	})
}

func (h *Handler) Hover(_ context.Context, params *lsp.HoverParams) (*lsp.Hover, error) {
	text, ok := h.documents[params.TextDocument.URI]
	if !ok {
		return nil, nil
	}

	content := nacha.HoverAt(text, params.Position.Line, params.Position.Character)
	if content == "" {
		return nil, nil
	}

	return &lsp.Hover{
		Contents: lsp.MarkupContent{
			Kind:  lsp.Markdown,
			Value: content,
		},
	}, nil
}

func boolPtr(b bool) *bool { return &b }
