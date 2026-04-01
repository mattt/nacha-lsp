package nacha

type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

type Range struct {
	Line           int
	StartCharacter int
	EndCharacter   int
}

type Diagnostic struct {
	Range          Range
	Line           int
	StartCharacter int
	EndCharacter   int
	Message        string
	Severity       Severity
}

func newDiagnostic(line, start, end int, message string, severity Severity) Diagnostic {
	r := Range{
		Line:           line,
		StartCharacter: start,
		EndCharacter:   end,
	}
	return Diagnostic{
		Range:          r,
		Line:           r.Line,
		StartCharacter: r.StartCharacter,
		EndCharacter:   r.EndCharacter,
		Message:        message,
		Severity:       severity,
	}
}
