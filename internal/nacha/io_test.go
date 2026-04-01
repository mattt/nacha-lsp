package nacha

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseReaderMatchesParse(t *testing.T) {
	text := validDomesticFile()
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

func TestWriteToMatchesSerialize(t *testing.T) {
	text := validDomesticFile()
	parsed := Parse(text)
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("unexpected parse diagnostics: %+v", parsed.Diagnostics)
	}

	var buf bytes.Buffer
	if _, err := parsed.File.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo returned error: %v", err)
	}
	if got, want := buf.String(), parsed.File.Serialize(); got != want {
		t.Fatalf("WriteTo output mismatch")
	}
}

func TestParseFileAndReadFrom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ach")
	content := validDomesticFile()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	parsedFromFile, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	if len(parsedFromFile.Diagnostics) != 0 {
		t.Fatalf("unexpected ParseFile diagnostics: %+v", parsedFromFile.Diagnostics)
	}

	var file File
	n, err := file.ReadFrom(strings.NewReader(content))
	if err != nil {
		t.Fatalf("ReadFrom returned error: %v", err)
	}
	if n != int64(len(content)) {
		t.Fatalf("ReadFrom byte count mismatch: got %d want %d", n, len(content))
	}
	if got, want := file.Serialize(), content; got != want {
		t.Fatalf("ReadFrom/Serialize mismatch")
	}
}
