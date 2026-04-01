package nacha

import (
	"io"
	"strings"
)

func SerializeFile(file *File) string {
	var b strings.Builder
	_, _ = WriteFile(&b, file)
	return b.String()
}

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
