package nacha

import (
	"strings"
	"testing"
)

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
