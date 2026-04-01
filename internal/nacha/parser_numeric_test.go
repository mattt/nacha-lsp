package nacha

import (
	"strings"
	"testing"
)

func TestParse_NumericFieldDiagnostics(t *testing.T) {
	lines := []string{
		makeFileHeader(),
		makeBatchHeader("200", "PPD"),
		makeEntry("22", "03AB0001", 1000), // invalid numeric RDFI prefix
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
