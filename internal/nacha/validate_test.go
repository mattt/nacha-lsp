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

func TestParseSerializeRoundTrip(t *testing.T) {
	original := validDomesticFile()
	parsed := Parse(original)
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("unexpected parse diagnostics: %+v", parsed.Diagnostics)
	}
	roundtrip := parsed.File.Serialize()
	if roundtrip != original {
		t.Fatalf("roundtrip mismatch\nexpected:\n%s\n\ngot:\n%s", original, roundtrip)
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
