package nacha

import "testing"

func TestLookupPositionReturnsExpandedBatchHeaderField(t *testing.T) {
	record := makeBatchHeader("200", "PPD")

	info, ok := LookupPosition(record, 51)
	if !ok || info.Field == nil {
		t.Fatal("expected field metadata for batch header SEC position")
	}
	if got, want := info.Field.Name, "Standard Entry Class Code"; got != want {
		t.Fatalf("field name = %q, want %q", got, want)
	}
	if got, want := info.FieldValue, "PPD"; got != want {
		t.Fatalf("field value = %q, want %q", got, want)
	}
}

func TestLookupPositionReturnsExpandedAddendaField(t *testing.T) {
	record := makeIATAddenda("10")

	info, ok := LookupPosition(record, 85)
	if !ok || info.Field == nil {
		t.Fatal("expected field metadata for addenda sequence position")
	}
	if got, want := info.Field.Name, "Addenda Sequence Number"; got != want {
		t.Fatalf("field name = %q, want %q", got, want)
	}
}
