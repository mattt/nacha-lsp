package nacha

import (
	"bytes"
	"testing"
)

func TestWriteToMatchesSerialize(t *testing.T) {
	text := readValidFixture(t)
	parsed := ParseWithOptions(text, ParseOptions{StrictLengths: false, StrictPadding: true})
	if parsed.File == nil || len(parsed.File.Records) == 0 {
		t.Fatalf("expected parsed records")
	}

	var buf bytes.Buffer
	if _, err := parsed.File.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo returned error: %v", err)
	}
	if got, want := buf.String(), parsed.File.Serialize(); got != want {
		t.Fatalf("WriteTo output mismatch")
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
