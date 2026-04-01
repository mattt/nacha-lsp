package nacha

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		"9" + strings.Repeat("A", 93), // invalid padding content
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
	validPath := filepath.Join("testdata", "sample-valid.ach")
	invalidPath := filepath.Join("testdata", "sample-invalid.ach")

	validBytes, err := os.ReadFile(validPath)
	if err != nil {
		t.Fatalf("read valid fixture: %v", err)
	}
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
