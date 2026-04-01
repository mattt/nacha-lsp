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
	quickFixKinds := []lsp.CodeActionKind{lsp.CodeActionQuickFix}
	return &lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptions{
				OpenClose: boolPtr(true),
				Change:    lsp.SyncFull,
				Save:      &lsp.SaveOptions{IncludeText: boolPtr(true)},
			},
			HoverProvider:              boolPtr(true),
			DocumentSymbolProvider:     boolPtr(true),
			DocumentFormattingProvider: boolPtr(true),
			CompletionProvider: &lsp.CompletionOptions{
				TriggerCharacters: []string{" ", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			},
			CodeActionProvider: &lsp.CodeActionOptions{
				CodeActionKinds: quickFixKinds,
			},
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
	if h.client == nil {
		return nil
	}

	text := h.documents[params.TextDocument.URI]
	diagnostics := buildDiagnostics(text)

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

func (h *Handler) Completion(_ context.Context, params *lsp.CompletionParams) (*lsp.CompletionList, error) {
	text, ok := h.documents[params.TextDocument.URI]
	if !ok {
		return nil, nil
	}
	line, ok := lineAt(text, params.Position.Line)
	if !ok {
		return nil, nil
	}
	info, ok := nacha.LookupPosition(line, params.Position.Character)
	if !ok || info.Field == nil {
		return nil, nil
	}

	suggestions := completionSuggestions(info)
	if len(suggestions) == 0 {
		return nil, nil
	}

	items := make([]lsp.CompletionItem, 0, len(suggestions))
	for _, suggestion := range suggestions {
		doc := lsp.MarkupContent{Kind: lsp.Markdown, Value: suggestion.documentation}
		kind := lsp.CompletionItemKindEnumMember
		items = append(items, lsp.CompletionItem{
			Label:         suggestion.label,
			Kind:          &kind,
			Detail:        suggestion.detail,
			Documentation: &doc,
			InsertText:    suggestion.value,
			TextEdit: &lsp.TextEdit{
				Range: lsp.Range{
					Start: lsp.Position{Line: params.Position.Line, Character: info.Field.Start - 1},
					End:   lsp.Position{Line: params.Position.Line, Character: info.Field.End},
				},
				NewText: suggestion.value,
			},
		})
	}

	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

func (h *Handler) DocumentSymbol(_ context.Context, params *lsp.DocumentSymbolParams) ([]lsp.DocumentSymbol, error) {
	text, ok := h.documents[params.TextDocument.URI]
	if !ok {
		return nil, nil
	}

	parsed := nacha.Parse(text)
	if parsed.File == nil {
		return nil, nil
	}

	symbols := make([]lsp.DocumentSymbol, 0, len(parsed.File.Batches)+2)
	if parsed.File.Header != nil {
		symbols = append(symbols, lineSymbol("File Header", "Record 1", lsp.SymbolKindObject, parsed.File.Header.Line()))
	}

	for i, batch := range parsed.File.Batches {
		if batch == nil || batch.Header == nil {
			continue
		}
		children := make([]lsp.DocumentSymbol, 0, len(batch.Entries))
		for j, entry := range batch.Entries {
			entryChildren := make([]lsp.DocumentSymbol, 0, len(entry.AddendaRecords()))
			for _, addenda := range entry.AddendaRecords() {
				entryChildren = append(entryChildren, lineSymbol(
					fmt.Sprintf("Addenda %s", addenda.AddendaTypeCode()),
					"Record 7",
					lsp.SymbolKindObject,
					addenda.Line(),
				))
			}
			entrySymbol := lineSymbol(
				fmt.Sprintf("Entry %d (%s)", j+1, entry.TransactionCode()),
				"Record 6",
				lsp.SymbolKindField,
				entry.Line(),
			)
			if len(entryChildren) > 0 {
				entrySymbol.Children = entryChildren
			}
			children = append(children, entrySymbol)
		}

		startLine := batch.Header.Line()
		endLine := startLine
		if batch.Control != nil {
			endLine = batch.Control.Line()
		} else if len(children) > 0 {
			endLine = children[len(children)-1].Range.End.Line
		}
		kind := lsp.SymbolKindModule
		symbols = append(symbols, lsp.DocumentSymbol{
			Name:   fmt.Sprintf("Batch %d", i+1),
			Detail: fmt.Sprintf("SEC %s", batch.Header.SECCode()),
			Kind:   kind,
			Range: lsp.Range{
				Start: lsp.Position{Line: startLine, Character: 0},
				End:   lsp.Position{Line: endLine, Character: 94},
			},
			SelectionRange: lsp.Range{
				Start: lsp.Position{Line: startLine, Character: 0},
				End:   lsp.Position{Line: startLine, Character: 94},
			},
			Children: children,
		})
	}

	if parsed.File.Control != nil {
		symbols = append(symbols, lineSymbol("File Control", "Record 9", lsp.SymbolKindObject, parsed.File.Control.Line()))
	}

	return symbols, nil
}

func (h *Handler) Formatting(_ context.Context, params *lsp.DocumentFormattingParams) ([]lsp.TextEdit, error) {
	text, ok := h.documents[params.TextDocument.URI]
	if !ok {
		return nil, nil
	}

	parsed := nacha.Parse(text)
	if hasErrors(parsed.Diagnostics) || parsed.File == nil {
		return nil, nil
	}

	formatted := parsed.File.Serialize()
	if formatted == text {
		return nil, nil
	}

	return []lsp.TextEdit{
		{
			Range:   wholeDocumentRange(text),
			NewText: formatted,
		},
	}, nil
}

func (h *Handler) CodeAction(_ context.Context, params *lsp.CodeActionParams) ([]lsp.CodeAction, error) {
	if len(params.Context.Only) > 0 && !containsCodeActionKind(params.Context.Only, lsp.CodeActionQuickFix) {
		return nil, nil
	}

	text, ok := h.documents[params.TextDocument.URI]
	if !ok {
		return nil, nil
	}
	lines := splitLines(text)
	actions := make([]lsp.CodeAction, 0, 3)

	for _, diag := range params.Context.Diagnostics {
		switch {
		case strings.Contains(diag.Message, "record must be exactly 94 characters"):
			line := diag.Range.Start.Line
			if line < 0 || line >= len(lines) {
				continue
			}
			updated := ensureRecordLength94(lines[line])
			if updated == lines[line] {
				continue
			}
			kind := lsp.CodeActionQuickFix
			preferred := true
			actions = append(actions, lsp.CodeAction{
				Title:       "Normalize record length to 94 characters",
				Kind:        &kind,
				IsPreferred: &preferred,
				Diagnostics: []lsp.Diagnostic{diag},
				Edit: &lsp.WorkspaceEdit{
					Changes: map[lsp.DocumentURI][]lsp.TextEdit{
						params.TextDocument.URI: {
							{
								Range: lsp.Range{
									Start: lsp.Position{Line: line, Character: 0},
									End:   lsp.Position{Line: line, Character: len(lines[line])},
								},
								NewText: updated,
							},
						},
					},
				},
			})
		case strings.Contains(diag.Message, "line count must be a multiple of 10 (blocking factor)"):
			remainder := len(lines) % 10
			if remainder == 0 {
				continue
			}
			missing := 10 - remainder
			paddingLines := make([]string, missing)
			for i := range paddingLines {
				paddingLines[i] = strings.Repeat("9", 94)
			}
			insert := strings.Join(paddingLines, "\n")
			if len(text) > 0 {
				if strings.HasSuffix(text, "\n") || strings.HasSuffix(text, "\r\n") || strings.HasSuffix(text, "\r") {
					insert = insert
				} else {
					insert = "\n" + insert
				}
			}

			kind := lsp.CodeActionQuickFix
			preferred := true
			actions = append(actions, lsp.CodeAction{
				Title:       "Append 9-record padding to satisfy blocking factor",
				Kind:        &kind,
				IsPreferred: &preferred,
				Diagnostics: []lsp.Diagnostic{diag},
				Edit: &lsp.WorkspaceEdit{
					Changes: map[lsp.DocumentURI][]lsp.TextEdit{
						params.TextDocument.URI: {
							{
								Range: lsp.Range{
									Start: documentEndPosition(text),
									End:   documentEndPosition(text),
								},
								NewText: insert,
							},
						},
					},
				},
			})
		}
	}

	if len(text) > 0 && !strings.HasSuffix(text, "\n") && !strings.HasSuffix(text, "\r\n") && !strings.HasSuffix(text, "\r") {
		kind := lsp.CodeActionQuickFix
		actions = append(actions, lsp.CodeAction{
			Title: "Insert trailing newline at end of file",
			Kind:  &kind,
			Edit: &lsp.WorkspaceEdit{
				Changes: map[lsp.DocumentURI][]lsp.TextEdit{
					params.TextDocument.URI: {
						{
							Range: lsp.Range{
								Start: documentEndPosition(text),
								End:   documentEndPosition(text),
							},
							NewText: "\n",
						},
					},
				},
			},
		})
	}

	if len(actions) == 0 {
		return nil, nil
	}
	return actions, nil
}

func boolPtr(b bool) *bool { return &b }

func buildDiagnostics(text string) []lsp.Diagnostic {
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
	return diagnostics
}

func hoverAt(text string, line, character int) string {
	lines := splitLines(text)
	if line < 0 || line >= len(lines) {
		return ""
	}
	info, ok := nacha.LookupPosition(lines[line], character)
	if !ok {
		return ""
	}
	if info.Field != nil {
		return fmt.Sprintf(
			"**%s** (`%c`)  \n**Field:** %s (positions %d-%d)  \n**Value:** `%s`  \n%s",
			info.RecordName,
			info.RecordType,
			info.Field.Name,
			info.Field.Start,
			info.Field.End,
			info.FieldValue,
			info.Field.Description,
		)
	}
	return fmt.Sprintf(
		"**%s** (`%c`)  \nPosition %d has no hover metadata yet.",
		info.RecordName,
		info.RecordType,
		info.Position,
	)
}

func lineAt(text string, line int) (string, bool) {
	lines := splitLines(text)
	if line < 0 || line >= len(lines) {
		return "", false
	}
	return lines[line], true
}

func lineSymbol(name, detail string, kind lsp.SymbolKind, line int) lsp.DocumentSymbol {
	return lsp.DocumentSymbol{
		Name:   name,
		Detail: detail,
		Kind:   kind,
		Range: lsp.Range{
			Start: lsp.Position{Line: line, Character: 0},
			End:   lsp.Position{Line: line, Character: 94},
		},
		SelectionRange: lsp.Range{
			Start: lsp.Position{Line: line, Character: 0},
			End:   lsp.Position{Line: line, Character: 94},
		},
	}
}

func hasErrors(diags []nacha.Diagnostic) bool {
	for _, diag := range diags {
		if diag.Severity == nacha.SeverityError {
			return true
		}
	}
	return false
}

func wholeDocumentRange(text string) lsp.Range {
	end := documentEndPosition(text)
	return lsp.Range{
		Start: lsp.Position{Line: 0, Character: 0},
		End:   end,
	}
}

func documentEndPosition(text string) lsp.Position {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	if normalized == "" {
		return lsp.Position{Line: 0, Character: 0}
	}
	parts := strings.Split(normalized, "\n")
	if strings.HasSuffix(normalized, "\n") {
		return lsp.Position{Line: len(parts) - 1, Character: 0}
	}
	last := parts[len(parts)-1]
	return lsp.Position{Line: len(parts) - 1, Character: len(last)}
}

func containsCodeActionKind(kinds []lsp.CodeActionKind, kind lsp.CodeActionKind) bool {
	for _, value := range kinds {
		if value == kind {
			return true
		}
	}
	return false
}

func ensureRecordLength94(raw string) string {
	if len(raw) == 94 {
		return raw
	}
	if len(raw) > 94 {
		return raw[:94]
	}
	return raw + strings.Repeat(" ", 94-len(raw))
}

type completionSuggestion struct {
	label         string
	value         string
	detail        string
	documentation string
}

func completionSuggestions(info nacha.PositionInfo) []completionSuggestion {
	field := info.Field
	if field == nil {
		return nil
	}

	switch {
	case info.RecordType == '5' && field.Start == 2 && field.End == 4:
		return []completionSuggestion{
			{label: "200", value: "200", detail: "Service Class Code", documentation: "Mixed debits and credits."},
			{label: "220", value: "220", detail: "Service Class Code", documentation: "Credits only."},
			{label: "225", value: "225", detail: "Service Class Code", documentation: "Debits only."},
		}
	case info.RecordType == '5' && field.Start == 51 && field.End == 53:
		return []completionSuggestion{
			{label: "PPD", value: "PPD", detail: "Standard Entry Class", documentation: "Prearranged payment and deposits."},
			{label: "CCD", value: "CCD", detail: "Standard Entry Class", documentation: "Corporate credit or debit."},
			{label: "CTX", value: "CTX", detail: "Standard Entry Class", documentation: "Corporate trade exchange."},
			{label: "IAT", value: "IAT", detail: "Standard Entry Class", documentation: "International ACH transaction."},
		}
	case info.RecordType == '6' && field.Start == 2 && field.End == 3:
		return []completionSuggestion{
			{label: "22", value: "22", detail: "Checking Credit", documentation: "Credit to checking account."},
			{label: "27", value: "27", detail: "Checking Debit", documentation: "Debit to checking account."},
			{label: "32", value: "32", detail: "Savings Credit", documentation: "Credit to savings account."},
			{label: "37", value: "37", detail: "Savings Debit", documentation: "Debit to savings account."},
		}
	case info.RecordType == '7' && field.Start == 2 && field.End == 3:
		return []completionSuggestion{
			{label: "02", value: "02", detail: "POS Addenda", documentation: "Point-of-sale addenda."},
			{label: "05", value: "05", detail: "Payment Addenda", documentation: "Payment-related information addenda."},
			{label: "98", value: "98", detail: "NOC Addenda", documentation: "Notification of change addenda."},
			{label: "99", value: "99", detail: "Return Addenda", documentation: "Return or dishonored return addenda."},
		}
	default:
		return nil
	}
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

