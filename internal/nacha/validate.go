package nacha

// Severity classifies the urgency of a [Diagnostic].
type Severity int

const (
	// SeverityError indicates a structural or data-integrity violation that
	// makes the ACH file invalid per the Nacha Operating Rules.
	SeverityError Severity = iota

	// SeverityWarning indicates a deviation from best practice or a
	// recommendation that does not necessarily invalidate the file (for
	// example, a line count that is not a multiple of ten, or padding records
	// that are not all nines).
	SeverityWarning
)

// Range identifies a span of characters within a single line of an ACH file.
// Positions follow the LSP convention: both Line and character offsets are
// 0-based, StartCharacter is inclusive, and EndCharacter is exclusive.
type Range struct {
	Line           int
	StartCharacter int
	EndCharacter   int
}

// Diagnostic reports a problem found during parsing or validation of an ACH
// file. The embedded [Range] locates the problematic span within the source
// text so that editors can highlight the affected characters.
type Diagnostic struct {
	Range          Range
	Line           int
	StartCharacter int
	EndCharacter   int
	Message        string
	Severity       Severity
}

func newDiagnostic(line, start, end int, message string, severity Severity) Diagnostic {
	r := Range{
		Line:           line,
		StartCharacter: start,
		EndCharacter:   end,
	}
	return Diagnostic{
		Range:          r,
		Line:           r.Line,
		StartCharacter: r.StartCharacter,
		EndCharacter:   r.EndCharacter,
		Message:        message,
		Severity:       severity,
	}
}

// Validate parses text as an ACH file and runs integrity checks on the
// resulting [File], returning all diagnostics from both phases.
//
// Structural checks (from parsing) include correct record sequence, proper
// nesting of batches, and non-empty batch bodies.
//
// Integrity checks include:
//   - Line count is a multiple of ten (the blocking factor).
//   - Each batch control's entry/addenda count, debit total, credit total,
//     and entry hash match the computed values from the batch's entries.
//   - The file control's batch count, block count, entry/addenda count,
//     total debit amount, total credit amount, and entry hash match the
//     computed values from all batches.
func Validate(text string) []Diagnostic {
	parseResult := ParseWithOptions(text, DefaultParseOptions())
	diags := append([]Diagnostic{}, parseResult.Diagnostics...)
	appendDiag := func(line, start, end int, message string, severity Severity) {
		diags = append(diags, newDiagnostic(line, start, end, message, severity))
	}

	file := parseResult.File
	if file == nil {
		return diags
	}

	lines := splitLines(text)
	lastLine := max(0, len(lines)-1)
	if len(lines)%blockSize != 0 {
		appendDiag(lastLine, 0, 1, "line count must be a multiple of 10 (blocking factor)", SeverityWarning)
	}

	if file.Header == nil {
		appendDiag(0, 0, 1, "file must start with a file header record (1)", SeverityError)
	}
	if file.Control == nil {
		appendDiag(lastLine, 0, 1, "file must include a file control record (9)", SeverityError)
		return diags
	}

	validateControlTotals(file, appendDiag)

	return diags
}

func validateControlTotals(file *File, appendDiag func(line, start, end int, message string, severity Severity)) {
	totalEntryAddenda := int64(0)
	totalDebit := int64(0)
	totalCredit := int64(0)
	totalHash := int64(0)

	for _, batch := range file.Batches {
		if batch == nil {
			continue
		}
		batchEntryAddenda, batchDebit, batchCredit, batchHash := summarizeBatch(batch)
		totalEntryAddenda += batchEntryAddenda
		totalDebit += batchDebit
		totalCredit += batchCredit
		totalHash += batchHash

		if batch.Control == nil {
			if batch.Header != nil {
				appendDiag(batch.Header.Line(), 0, 1, "batch is missing batch control record (8)", SeverityError)
			}
			continue
		}

		if batch.Control.EntryAddendaCount != batchEntryAddenda {
			appendDiag(batch.Control.Line(), 4, 10, "batch control entry/addenda count does not match records in batch", SeverityError)
		}
		if batch.Control.TotalDebitAmount != batchDebit {
			appendDiag(batch.Control.Line(), 20, 32, "batch control debit total does not match calculated debit total", SeverityError)
		}
		if batch.Control.TotalCreditAmount != batchCredit {
			appendDiag(batch.Control.Line(), 32, 44, "batch control credit total does not match calculated credit total", SeverityError)
		}
		expectedBatchHash := formatUnsigned(batchHash, 10)
		if batch.Control.EntryHash != expectedBatchHash {
			appendDiag(batch.Control.Line(), 10, 20, "batch control entry hash does not match calculated hash", SeverityError)
		}
	}

	if file.Control.BatchCount != int64(len(file.Batches)) {
		appendDiag(file.Control.Line(), 1, 7, "file control batch count does not match number of batches", SeverityError)
	}
	if file.Control.EntryAddendaCount != totalEntryAddenda {
		appendDiag(file.Control.Line(), 13, 21, "file control entry/addenda count does not match calculated count", SeverityError)
	}
	if file.Control.TotalDebitAmount != totalDebit {
		appendDiag(file.Control.Line(), 31, 43, "file control debit total does not match calculated debit total", SeverityError)
	}
	if file.Control.TotalCreditAmount != totalCredit {
		appendDiag(file.Control.Line(), 43, 55, "file control credit total does not match calculated credit total", SeverityError)
	}
	expectedHash := formatUnsigned(totalHash, 10)
	if file.Control.EntryHash != expectedHash {
		appendDiag(file.Control.Line(), 21, 31, "file control entry hash does not match calculated hash", SeverityError)
	}

	recordsCount := int64(len(file.Records))
	expectedBlocks := (recordsCount + (blockSize - 1)) / blockSize
	if file.Control.BlockCount != expectedBlocks {
		appendDiag(file.Control.Line(), 7, 13, "file control block count does not match record count", SeverityError)
	}
}

func summarizeBatch(batch *Batch) (entryAddenda int64, debit int64, credit int64, hash int64) {
	for _, entry := range batch.Entries {
		entryAddenda++
		if isCreditTransactionCode(entry.TransactionCode()) {
			credit += entry.AmountCents()
		} else {
			debit += entry.AmountCents()
		}
		entryAddenda += int64(len(entry.AddendaRecords()))
		if prefix, ok := parseInt64Strict(entry.ReceivingDFIPrefix()); ok {
			hash += prefix
		}
	}
	return entryAddenda, debit, credit, hash
}

func isCreditTransactionCode(code TransactionCode) bool {
	switch string(code) {
	case "20", "21", "22", "23", "24",
		"30", "31", "32", "33", "34",
		"41", "42", "43", "44",
		"51", "52", "53", "54",
		"81", "83", "85", "87":
		return true
	default:
		return false
	}
}
