package nacha

import (
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
