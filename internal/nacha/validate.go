package nacha

func Validate(text string) []Diagnostic {
	parseResult := ParseWithOptions(text, DefaultParseOptions())
	diags := append([]Diagnostic{}, parseResult.Diagnostics...)
	appendDiag := func(line, start, end int, message string, severity Severity) {
		diags = append(diags, newDiagnostic(line, start, end, message, severity))
	}

	file := parseResult.File
	if file == nil {
		return diags
	}

	lines := splitLines(text)
	lastLine := max(0, len(lines)-1)
	if len(lines)%blockSize != 0 {
		appendDiag(lastLine, 0, 1, "line count must be a multiple of 10 (blocking factor)", SeverityWarning)
	}

	if file.Header == nil {
		appendDiag(0, 0, 1, "file must start with a file header record (1)", SeverityError)
	}
	if file.Control == nil {
		appendDiag(lastLine, 0, 1, "file must include a file control record (9)", SeverityError)
		return diags
	}

	validateControlTotals(file, appendDiag)

	return diags
}
