package nacha

import (
	"strings"
	"testing"
)

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
