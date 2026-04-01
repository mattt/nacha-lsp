package nacha

import (
	"strconv"
	"strings"
)

func ensureRecordLength(raw string) string {
	if len(raw) == recordLength {
		return raw
	}
	if len(raw) > recordLength {
		return raw[:recordLength]
	}
	return raw + strings.Repeat(" ", recordLength-len(raw))
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

func parseInt64(s string) int64 {
	v, _ := parseInt64Strict(s)
	return v
}

func parseInt64Strict(s string) (int64, bool) {
	if s == "" {
		return 0, true
	}
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func parseIntField(raw string, line, start, end int, name string, appendDiag func(line, start, end int, message string, severity Severity)) int64 {
	value := fieldValueTrimmed(raw, start, end)
	if value == "" {
		return 0
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		appendDiag(line, start-1, end, "field "+name+" must be numeric", SeverityError)
		return 0
	}
	return n
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}

func isValidRecordType(recordType byte) bool {
	switch recordType {
	case '1', '5', '6', '7', '8', '9':
		return true
	default:
		return false
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
