package nacha

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse_StructureDiagnostics(t *testing.T) {
	tests := []struct {
		name    string
		lines   []string
		wantMsg string
	}{
		{
			name: "duplicate file header",
			lines: []string{
				makeRecord('1'),
				makeRecord('1'),
				makeRecord('9'),
			},
			wantMsg: "only one file header",
		},
		{
			name: "addenda before entry",
			lines: []string{
				makeRecord('1'),
				makeRecord('5'),
				makeRecord('7'),
				makeRecord('8'),
				makeRecord('9'),
			},
			wantMsg: "must follow an entry detail",
		},
		{
			name: "file ends with open batch",
			lines: []string{
				makeRecord('1'),
				makeRecord('5'),
				makeRecord('6'),
				makeRecord('9'),
			},
			wantMsg: "missing batch control",
		},
		{
			name: "missing file control",
			lines: []string{
				makeRecord('1'),
				makeRecord('5'),
				makeRecord('6'),
				makeRecord('8'),
			},
			wantMsg: "must include a file control",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(strings.Join(tt.lines, "\n"))
			found := false
			for _, d := range got.Diagnostics {
				if strings.Contains(strings.ToLower(d.Message), strings.ToLower(tt.wantMsg)) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected diagnostic containing %q, got %+v", tt.wantMsg, got.Diagnostics)
			}
		})
	}
}

func TestParse_NumericFieldDiagnostics(t *testing.T) {
	lines := []string{
		makeFileHeader(),
		makeBatchHeader("200", "PPD"),
		makeEntry("22", "03AB0001", 1000),
		makeBatchControl("200", 1, "0003130001", 0, 1000),
		makeFileControl(1, 1, 1, "0003130001", 0, 1000),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
	}

	result := Parse(strings.Join(lines, "\n"))
	found := false
	for _, d := range result.Diagnostics {
		if strings.Contains(d.Message, "receiving DFI identification must be numeric") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected numeric field diagnostic, got %+v", result.Diagnostics)
	}
}

func TestValidate_DiagnosticRangeMirrorsLegacyFields(t *testing.T) {
	diags := Validate("short")
	if len(diags) == 0 {
		t.Fatal("expected diagnostics")
	}
	for _, d := range diags {
		if d.Range.Line != d.Line || d.Range.StartCharacter != d.StartCharacter || d.Range.EndCharacter != d.EndCharacter {
			t.Fatalf("range mismatch: %+v", d)
		}
	}
}

func TestParseWithOptions_StrictLengths(t *testing.T) {
	text := "123"
	strict := ParseWithOptions(text, ParseOptions{StrictLengths: true, StrictPadding: true})
	loose := ParseWithOptions(text, ParseOptions{StrictLengths: false, StrictPadding: true})

	if len(strict.Diagnostics) == 0 {
		t.Fatalf("expected strict parse diagnostics")
	}
	if len(loose.Diagnostics) >= len(strict.Diagnostics) {
		t.Fatalf("expected loose parse to emit fewer diagnostics")
	}
}

func TestParseWithOptions_StrictPadding(t *testing.T) {
	lines := []string{
		makeFileHeader(),
		makeBatchHeader("200", "PPD"),
		makeEntry("22", "03130001", 1000),
		makeBatchControl("200", 1, "0003130001", 0, 1000),
		makeFileControl(1, 1, 1, "0003130001", 0, 1000),
		"9" + strings.Repeat("A", 93),
	}
	text := strings.Join(lines, "\n")

	strict := ParseWithOptions(text, ParseOptions{StrictLengths: true, StrictPadding: true})
	loose := ParseWithOptions(text, ParseOptions{StrictLengths: true, StrictPadding: false})

	strictFound := false
	for _, d := range strict.Diagnostics {
		if strings.Contains(d.Message, "padding records after file control must be all 9s") {
			strictFound = true
			break
		}
	}
	if !strictFound {
		t.Fatalf("expected strict padding diagnostic")
	}

	looseFound := false
	for _, d := range loose.Diagnostics {
		if strings.Contains(d.Message, "padding records after file control must be all 9s") {
			looseFound = true
			break
		}
	}
	if looseFound {
		t.Fatalf("did not expect padding diagnostic in loose mode")
	}
}

func TestParseFixtureFiles(t *testing.T) {
	invalidPath := filepath.Join("testdata", "sample-invalid.ach")

	validBytes := []byte(readValidFixture(t))
	invalidBytes, err := os.ReadFile(invalidPath)
	if err != nil {
		t.Fatalf("read invalid fixture: %v", err)
	}

	valid := ParseWithOptions(string(validBytes), DefaultParseOptions())
	if valid.File == nil || len(valid.File.Records) == 0 {
		t.Fatalf("expected valid fixture to produce records")
	}
	if len(valid.File.Batches) == 0 {
		t.Fatalf("expected valid fixture to produce batches")
	}

	invalid := ParseWithOptions(string(invalidBytes), DefaultParseOptions())
	if len(invalid.Diagnostics) == 0 {
		t.Fatalf("expected invalid fixture diagnostics")
	}
	foundInvalidType := false
	for _, d := range invalid.Diagnostics {
		if strings.Contains(d.Message, "record type must be one of") {
			foundInvalidType = true
			break
		}
	}
	if !foundInvalidType {
		t.Fatalf("expected invalid record type diagnostic")
	}
}

func TestParse_DomesticAddendaVariants(t *testing.T) {
	tests := []struct {
		code string
		want any
	}{
		{code: "02", want: &PointOfSaleAddenda02{}},
		{code: "05", want: &Addenda05{}},
		{code: "98", want: &NotificationOfChangeAddenda98{}},
		{code: "99", want: &ReturnAddenda99{}},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			line := []byte(makeRecord('7'))
			writeField(line, 2, 3, tt.code)
			writeField(line, 4, 83, "TEST")

			file := Parse(strings.Join([]string{
				makeFileHeader(),
				makeBatchHeader("200", "PPD"),
				makeEntry("22", "03130001", 1000),
				string(line),
				makeBatchControl("200", 2, "0003130001", 0, 1000),
				makeFileControl(1, 1, 2, "0003130001", 0, 1000),
				strings.Repeat("9", 94),
				strings.Repeat("9", 94),
				strings.Repeat("9", 94),
				strings.Repeat("9", 94),
			}, "\n"))

			entry := file.File.Batches[0].Entries[0]
			addenda := entry.AddendaRecords()[0]
			switch tt.want.(type) {
			case *PointOfSaleAddenda02:
				if _, ok := addenda.(*PointOfSaleAddenda02); !ok {
					t.Fatalf("expected POS addenda, got %T", addenda)
				}
			case *Addenda05:
				if _, ok := addenda.(*Addenda05); !ok {
					t.Fatalf("expected addenda05, got %T", addenda)
				}
			case *NotificationOfChangeAddenda98:
				if _, ok := addenda.(*NotificationOfChangeAddenda98); !ok {
					t.Fatalf("expected noc98, got %T", addenda)
				}
			case *ReturnAddenda99:
				if _, ok := addenda.(*ReturnAddenda99); !ok {
					t.Fatalf("expected ret99, got %T", addenda)
				}
			}
		})
	}
}

func TestParseReaderMatchesParse(t *testing.T) {
	text := readValidFixture(t)
	fromString := Parse(text)
	fromReader, err := ParseReader(strings.NewReader(text))
	if err != nil {
		t.Fatalf("ParseReader returned error: %v", err)
	}
	if len(fromString.Diagnostics) != len(fromReader.Diagnostics) {
		t.Fatalf("diagnostic count mismatch: %d vs %d", len(fromString.Diagnostics), len(fromReader.Diagnostics))
	}
	if got, want := fromReader.File.Serialize(), fromString.File.Serialize(); got != want {
		t.Fatalf("serialize mismatch between Parse and ParseReader")
	}
}

func TestParseFileAndReadFrom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ach")
	content := readValidFixture(t)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	parsedFromFile, err := ParseFileWithOptions(path, ParseOptions{StrictLengths: false, StrictPadding: true})
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	if parsedFromFile.File == nil || len(parsedFromFile.File.Records) == 0 {
		t.Fatalf("expected parsed file records")
	}

	var file File
	n, err := file.ReadFrom(strings.NewReader(content))
	if err != nil {
		t.Fatalf("ReadFrom returned error: %v", err)
	}
	if n != int64(len(content)) {
		t.Fatalf("ReadFrom byte count mismatch: got %d want %d", n, len(content))
	}
	expected := Parse(content).File.Serialize()
	if got, want := file.Serialize(), expected; got != want {
		t.Fatalf("ReadFrom/Serialize mismatch")
	}
}
