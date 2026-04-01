package nacha

import (
	"io"
	"strings"
)

// SerializeFile returns file as a newline-separated string of 94-character
// records in source order. It is a convenience wrapper around [WriteFile].
func SerializeFile(file *File) string {
	var b strings.Builder
	_, _ = WriteFile(&b, file)
	return b.String()
}

// WriteFile writes the records of file to w as newline-separated 94-character
// lines in source order. Records shorter than 94 characters are space-padded;
// records longer than 94 characters are truncated. It returns the number of
// bytes written and any write error from w.
func WriteFile(w io.Writer, file *File) (int64, error) {
	if file == nil || len(file.Records) == 0 {
		return 0, nil
	}

	lines := make([]string, 0, len(file.Records))
	for _, record := range file.Records {
		raw := record.Dump()
		if len(raw) == 0 {
			continue
		}
		if len(raw) != recordLength {
			raw = ensureRecordLength(raw)
		}
		lines = append(lines, raw)
	}

	n, err := io.WriteString(w, strings.Join(lines, "\n"))
	return int64(n), err
}
