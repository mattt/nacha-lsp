package nacha

import (
	"strings"
	"testing"
)

func TestValidate_ValidFile(t *testing.T) {
	text := validDomesticFile()
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
		makeRecord('1'),
		makeRecord('6'),
		makeRecord('9'),
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
		makeRecord('1'),
		makeRecord('5'),
		makeRecord('6'),
		makeRecord('8'),
		makeRecord('9'),
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

func TestParseIATAddendaVariant(t *testing.T) {
	lines := []string{
		makeRecord('1'),
		makeIATBatchHeader(),
		makeIATEntry(),
		makeIATAddenda("10"),
		makeBatchControl("200", 2, "0003130001", 0, 1000),
		makeFileControl(1, 1, 2, "0003130001", 0, 1000),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
	}
	parsed := Parse(strings.Join(lines, "\n"))
	if len(parsed.File.Batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(parsed.File.Batches))
	}
	entry := parsed.File.Batches[0].Entries[0]
	if _, ok := entry.(*InternationalEntryDetail); !ok {
		t.Fatalf("expected international entry variant, got %T", entry)
	}
	addenda := entry.AddendaRecords()[0]
	if _, ok := addenda.(*InternationalAddenda10); !ok {
		t.Fatalf("expected international addenda 10 variant, got %T", addenda)
	}
}

func TestValidate_ControlMismatchDiagnostics(t *testing.T) {
	lines := []string{
		makeFileHeader(),
		makeBatchHeader("200", "PPD"),
		makeEntry("22", "03130001", 1000),
		makeBatchControl("200", 99, "9999999999", 9999, 9999),
		makeFileControl(99, 99, 99, "9999999999", 9999, 9999),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
	}
	diags := Validate(strings.Join(lines, "\n"))
	if len(diags) == 0 {
		t.Fatal("expected diagnostics")
	}
	wanted := []string{
		"batch control entry/addenda count does not match",
		"batch control entry hash does not match",
		"file control batch count does not match",
		"file control block count does not match",
	}
	for _, want := range wanted {
		found := false
		for _, d := range diags {
			if strings.Contains(d.Message, want) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected diagnostic containing %q, got %+v", want, diags)
		}
	}
}
