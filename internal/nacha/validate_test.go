package nacha

import (
	"strings"
	"testing"
)

func TestValidate_ValidFile(t *testing.T) {
	text := strings.Join([]string{
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

	diags := Validate(text)
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %d: %+v", len(diags), diags)
	}
}

func TestValidate_InvalidLength(t *testing.T) {
	text := "123"
	diags := Validate(text)
	if len(diags) == 0 {
		t.Fatal("expected diagnostics")
	}
}

func TestValidate_OrderError(t *testing.T) {
	text := strings.Join([]string{
		recordLine('1', 'A'),
		recordLine('6', 'C'),
		recordLine('9', '9'),
	}, "\n")

	diags := Validate(text)
	if len(diags) == 0 {
		t.Fatal("expected diagnostics")
	}

	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "must be inside a batch") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected batch ordering diagnostic, got %+v", diags)
	}
}

func TestValidate_BlockingFactorWarning(t *testing.T) {
	text := strings.Join([]string{
		recordLine('1', 'A'),
		recordLine('5', 'B'),
		recordLine('6', 'C'),
		recordLine('8', 'D'),
		recordLine('9', 'E'),
	}, "\n")

	diags := Validate(text)
	foundWarning := false
	for _, d := range diags {
		if d.Severity == SeverityWarning && strings.Contains(d.Message, "multiple of 10") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Fatalf("expected blocking-factor warning, got %+v", diags)
	}
}

func recordLine(recordType byte, fill byte) string {
	return string(recordType) + strings.Repeat(string(fill), 93)
}
