package nacha

type ParseOptions struct {
	StrictPadding bool
	StrictLengths bool
}

func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		StrictPadding: true,
		StrictLengths: true,
	}
}

type ParseResult struct {
	File        *File
	Diagnostics []Diagnostic
}
