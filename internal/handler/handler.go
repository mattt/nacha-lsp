package handler

import (
	"context"
	"fmt"
	"strings"

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

	content := hoverAt(text, params.Position.Line, params.Position.Character)
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

type fieldInfo struct {
	start       int
	end         int
	name        string
	description string
}

var recordFields = map[byte][]fieldInfo{
	'1': {
		{start: 1, end: 1, name: "Record Type Code", description: "File Header record identifier; must be `1`."},
		{start: 2, end: 3, name: "Priority Code", description: "Priority code; expected `01`."},
		{start: 4, end: 13, name: "Immediate Destination", description: "Routing number of destination financial institution."},
		{start: 14, end: 23, name: "Immediate Origin", description: "Originating financial institution identifier."},
		{start: 24, end: 29, name: "File Creation Date", description: "Creation date in YYMMDD."},
		{start: 30, end: 33, name: "File Creation Time", description: "Creation time in HHMM."},
		{start: 34, end: 34, name: "File ID Modifier", description: "Uniqueness discriminator for files created the same day."},
		{start: 35, end: 37, name: "Record Size", description: "Must be `094`."},
		{start: 38, end: 39, name: "Blocking Factor", description: "Must be `10`."},
		{start: 40, end: 40, name: "Format Code", description: "Must be `1`."},
	},
	'5': {
		{start: 1, end: 1, name: "Record Type Code", description: "Batch Header record identifier; must be `5`."},
		{start: 2, end: 4, name: "Service Class Code", description: "Batch class: `200` mixed, `220` credits, `225` debits."},
		{start: 41, end: 50, name: "Company Identification", description: "Company ID for the originator in this batch."},
		{start: 51, end: 53, name: "Standard Entry Class Code", description: "SEC code describing the payment type."},
		{start: 70, end: 75, name: "Effective Entry Date", description: "Requested settlement date (YYMMDD)."},
	},
	'6': {
		{start: 1, end: 1, name: "Record Type Code", description: "Entry Detail record identifier; must be `6`."},
		{start: 2, end: 3, name: "Transaction Code", description: "Account type and debit/credit code."},
		{start: 4, end: 11, name: "Receiving DFI Identification", description: "First 8 digits of RDFI routing number."},
		{start: 12, end: 12, name: "Check Digit", description: "Ninth digit of RDFI routing number."},
		{start: 30, end: 39, name: "Amount", description: "Entry amount in cents, right-justified and zero-filled."},
		{start: 80, end: 94, name: "Trace Number", description: "ODFI trace number for the entry."},
	},
	'7': {
		{start: 1, end: 1, name: "Record Type Code", description: "Addenda record identifier; must be `7`."},
		{start: 2, end: 3, name: "Addenda Type Code", description: "Addenda discriminator (`02`, `05`, `10`-`18`, `98`, `99`)."},
	},
	'8': {
		{start: 1, end: 1, name: "Record Type Code", description: "Batch Control record identifier; must be `8`."},
		{start: 2, end: 4, name: "Service Class Code", description: "Must match corresponding batch header."},
		{start: 5, end: 10, name: "Entry/Addenda Count", description: "Count of detail and addenda records in batch."},
		{start: 11, end: 20, name: "Entry Hash", description: "Hash sum of RDFI prefixes, modulo 10 digits."},
		{start: 21, end: 32, name: "Total Debit Amount", description: "Total debit amount in cents."},
		{start: 33, end: 44, name: "Total Credit Amount", description: "Total credit amount in cents."},
	},
	'9': {
		{start: 1, end: 1, name: "Record Type Code", description: "File Control or padding record identifier; starts with `9`."},
		{start: 2, end: 7, name: "Batch Count", description: "Total batches in file."},
		{start: 8, end: 13, name: "Block Count", description: "Total 10-record blocks in file."},
		{start: 14, end: 21, name: "Entry/Addenda Count", description: "Total entry and addenda records in file."},
		{start: 22, end: 31, name: "Entry Hash", description: "File-level entry hash."},
		{start: 32, end: 43, name: "Total Debit Amount", description: "Total debit amount in cents."},
		{start: 44, end: 55, name: "Total Credit Amount", description: "Total credit amount in cents."},
	},
}

var recordDescriptions = map[byte]string{
	'1': "File Header",
	'5': "Batch Header",
	'6': "Entry Detail",
	'7': "Addenda",
	'8': "Batch Control",
	'9': "File Control or Padding",
}

func hoverAt(text string, line, character int) string {
	lines := splitLines(text)
	if line < 0 || line >= len(lines) {
		return ""
	}
	record := lines[line]
	if len(record) == 0 {
		return ""
	}
	recordType := record[0]
	desc, ok := recordDescriptions[recordType]
	if !ok {
		return ""
	}
	if character < 0 {
		character = 0
	}
	position := character + 1
	for _, field := range recordFields[recordType] {
		if position < field.start || position > field.end {
			continue
		}
		value := sliceValue(record, field.start, field.end)
		return fmt.Sprintf(
			"**%s** (`%c`)  \n**Field:** %s (positions %d-%d)  \n**Value:** `%s`  \n%s",
			desc, recordType, field.name, field.start, field.end, value, field.description,
		)
	}
	return fmt.Sprintf("**%s** (`%c`)  \nPosition %d has no hover metadata yet.", desc, recordType, position)
}

func splitLines(text string) []string {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

func sliceValue(record string, start, end int) string {
	if start < 1 {
		start = 1
	}
	if end > len(record) {
		end = len(record)
	}
	if start > end || start > len(record) {
		return ""
	}
	value := record[start-1 : end]
	return strings.TrimSpace(value)
}
