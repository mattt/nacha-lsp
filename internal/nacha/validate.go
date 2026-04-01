package nacha

import "strings"

type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

type Diagnostic struct {
	Line           int
	StartCharacter int
	EndCharacter   int
	Message        string
	Severity       Severity
}

const recordLength = 94

func Validate(text string) []Diagnostic {
	lines := splitLines(text)
	if len(lines) == 0 {
		return []Diagnostic{{
			Line:           0,
			StartCharacter: 0,
			EndCharacter:   0,
			Message:        "file is empty",
			Severity:       SeverityError,
		}}
	}

	diags := make([]Diagnostic, 0)
	appendDiag := func(line, start, end int, message string, severity Severity) {
		diags = append(diags, Diagnostic{
			Line:           line,
			StartCharacter: start,
			EndCharacter:   end,
			Message:        message,
			Severity:       severity,
		})
	}

	seenFileHeader := false
	seenFileControl := false
	inBatch := false
	entrySeenInBatch := false
	lastType := byte(0)

	for i, line := range lines {
		if len(line) != recordLength {
			appendDiag(i, 0, max(1, len(line)), "record must be exactly 94 characters", SeverityError)
		}
		if len(line) == 0 {
			appendDiag(i, 0, 1, "record cannot be empty", SeverityError)
			continue
		}

		rt := line[0]
		if !isValidRecordType(rt) {
			appendDiag(i, 0, 1, "record type must be one of 1, 5, 6, 7, 8, 9", SeverityError)
			lastType = rt
			continue
		}

		switch rt {
		case '1':
			if i != 0 {
				appendDiag(i, 0, 1, "file header record (1) must be first", SeverityError)
			}
			if seenFileHeader {
				appendDiag(i, 0, 1, "file must contain only one file header record (1)", SeverityError)
			}
			if seenFileControl {
				appendDiag(i, 0, 1, "record type 1 is not allowed after file control", SeverityError)
			}
			seenFileHeader = true
		case '5':
			if !seenFileHeader {
				appendDiag(i, 0, 1, "batch header record (5) requires a preceding file header (1)", SeverityError)
			}
			if seenFileControl {
				appendDiag(i, 0, 1, "batch header record (5) is not allowed after file control", SeverityError)
			}
			if inBatch {
				appendDiag(i, 0, 1, "batch header record (5) cannot appear before batch control (8)", SeverityError)
			}
			inBatch = true
			entrySeenInBatch = false
		case '6':
			if !inBatch {
				appendDiag(i, 0, 1, "entry detail record (6) must be inside a batch", SeverityError)
			}
			if seenFileControl {
				appendDiag(i, 0, 1, "entry detail record (6) is not allowed after file control", SeverityError)
			}
			entrySeenInBatch = true
		case '7':
			if !inBatch {
				appendDiag(i, 0, 1, "addenda record (7) must be inside a batch", SeverityError)
			}
			if lastType != '6' && lastType != '7' {
				appendDiag(i, 0, 1, "addenda record (7) must follow an entry detail (6) or addenda (7)", SeverityError)
			}
			if seenFileControl {
				appendDiag(i, 0, 1, "addenda record (7) is not allowed after file control", SeverityError)
			}
		case '8':
			if !inBatch {
				appendDiag(i, 0, 1, "batch control record (8) must close an open batch", SeverityError)
			}
			if seenFileControl {
				appendDiag(i, 0, 1, "batch control record (8) is not allowed after file control", SeverityError)
			}
			if inBatch && !entrySeenInBatch {
				appendDiag(i, 0, 1, "batch must include at least one entry detail record (6) before control (8)", SeverityError)
			}
			inBatch = false
		case '9':
			if !seenFileControl {
				if inBatch {
					appendDiag(i, 0, 1, "file control record (9) cannot appear before batch control (8)", SeverityError)
				}
				if !seenFileHeader {
					appendDiag(i, 0, 1, "file control record (9) requires a file header (1)", SeverityError)
				}
				seenFileControl = true
			} else if strings.Trim(line, "9") != "" {
				appendDiag(i, 0, min(len(line), recordLength), "padding records after file control must be all 9s", SeverityWarning)
			}
		}

		lastType = rt
	}

	if !seenFileHeader {
		appendDiag(0, 0, 1, "file must start with a file header record (1)", SeverityError)
	}
	if inBatch {
		appendDiag(len(lines)-1, 0, 1, "file ended with an open batch; missing batch control record (8)", SeverityError)
	}
	if !seenFileControl {
		appendDiag(len(lines)-1, 0, 1, "file must include a file control record (9)", SeverityError)
	}
	if len(lines)%10 != 0 {
		appendDiag(len(lines)-1, 0, 1, "line count must be a multiple of 10 (blocking factor)", SeverityWarning)
	}

	return diags
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

func isValidRecordType(recordType byte) bool {
	switch recordType {
	case '1', '5', '6', '7', '8', '9':
		return true
	default:
		return false
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
