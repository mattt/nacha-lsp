package nacha

import (
	"bufio"
	"io"
	"os"
	"strings"
)

func Parse(text string) ParseResult {
	return ParseWithOptions(text, DefaultParseOptions())
}

func ParseWithOptions(text string, options ParseOptions) ParseResult {
	return parseLines(splitLines(text), options)
}

func ParseReader(r io.Reader) (ParseResult, error) {
	return ParseReaderWithOptions(r, DefaultParseOptions())
}

func ParseReaderWithOptions(r io.Reader, options ParseOptions) (ParseResult, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 256), 1024*1024)
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, strings.TrimRight(scanner.Text(), "\r"))
	}
	if err := scanner.Err(); err != nil {
		return ParseResult{}, err
	}
	return parseLines(lines, options), nil
}

func ParseFile(path string) (ParseResult, error) {
	return ParseFileWithOptions(path, DefaultParseOptions())
}

func ParseFileWithOptions(path string, options ParseOptions) (ParseResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return ParseResult{}, err
	}
	defer f.Close()
	return ParseReaderWithOptions(f, options)
}

func parseLines(lines []string, options ParseOptions) ParseResult {
	result := ParseResult{
		File: &File{
			Batches: make([]*Batch, 0),
			Padding: make([]PaddingRecord, 0),
			Records: make([]Record, 0),
		},
		Diagnostics: make([]Diagnostic, 0),
	}
	if len(lines) == 0 {
		result.Diagnostics = append(result.Diagnostics, newDiagnostic(0, 0, 0, "file is empty", SeverityError))
		return result
	}

	ctx := parseContext{
		appendDiag: func(line, start, end int, message string, severity Severity) {
			result.Diagnostics = append(result.Diagnostics, newDiagnostic(line, start, end, message, severity))
		},
	}

	for i, rawLine := range lines {
		if len(rawLine) != recordLength {
			if options.StrictLengths {
				ctx.appendDiag(i, 0, max(1, len(rawLine)), "record must be exactly 94 characters", SeverityError)
			}
		}
		if len(rawLine) == 0 {
			ctx.appendDiag(i, 0, 1, "record cannot be empty", SeverityError)
			continue
		}

		line := ensureRecordLength(rawLine)
		recordType := line[0]
		if !isValidRecordType(recordType) {
			ctx.appendDiag(i, 0, 1, "record type must be one of 1, 5, 6, 7, 8, 9", SeverityError)
			continue
		}

		ctx.trackStructure(recordType, i)

		switch recordType {
		case '1':
			rec := parseFileHeader(line, i)
			result.File.Header = rec
			result.File.Records = append(result.File.Records, rec)
		case '5':
			header := parseBatchHeader(line, i)
			ctx.currentBatch = &Batch{Header: header, Entries: make([]EntryRecord, 0)}
			result.File.Batches = append(result.File.Batches, ctx.currentBatch)
			result.File.Records = append(result.File.Records, header)
			ctx.lastEntry = nil
		case '6':
			rec := parseEntryDetail(line, i, ctx.currentBatch, ctx.appendDiag)
			if ctx.currentBatch != nil {
				ctx.currentBatch.Entries = append(ctx.currentBatch.Entries, rec)
				ctx.lastEntry = rec
			}
			result.File.Records = append(result.File.Records, rec)
		case '7':
			rec := parseAddenda(line, i, ctx.currentBatch)
			if ctx.lastEntry != nil {
				addenda := ctx.lastEntry.AddendaRecords()
				ctx.lastEntry.SetAddenda(append(addenda, rec))
			}
			result.File.Records = append(result.File.Records, rec)
		case '8':
			rec := parseBatchControl(line, i, ctx.appendDiag)
			if ctx.currentBatch != nil {
				ctx.currentBatch.Control = rec
			}
			result.File.Records = append(result.File.Records, rec)
			ctx.currentBatch = nil
			ctx.lastEntry = nil
		case '9':
			if result.File.Control == nil {
				rec := parseFileControl(line, i, ctx.appendDiag)
				result.File.Control = rec
				result.File.Records = append(result.File.Records, rec)
			} else {
				padding := PaddingRecord{
					recordBase: recordBase{Raw: line, LineNo: i, RecType: '9'},
				}
				if options.StrictPadding && !isAllNines(line) {
					ctx.appendDiag(i, 0, recordLength, "padding records after file control must be all 9s", SeverityWarning)
				}
				result.File.Padding = append(result.File.Padding, padding)
				result.File.Records = append(result.File.Records, padding)
			}
		}
	}

	ctx.finish(len(lines))

	return result
}

func parseFileHeader(raw string, line int) *FileHeader {
	return &FileHeader{
		recordBase:           recordBase{Raw: raw, LineNo: line, RecType: '1'},
		ImmediateDestination: fieldValueTrimmed(raw, 4, 13),
		ImmediateOrigin:      fieldValueTrimmed(raw, 14, 23),
		FileCreationDate:     fieldValueTrimmed(raw, 24, 29),
		FileCreationTime:     fieldValueTrimmed(raw, 30, 33),
		FileIDModifier:       fieldValueTrimmed(raw, 34, 34),
	}
}

func parseBatchHeader(raw string, line int) BatchHeaderRecord {
	if isInternationalBatchHeader(raw) {
		return &InternationalBatchHeader{
			recordBase:               recordBase{Raw: raw, LineNo: line, RecType: '5', Variant: "iat"},
			ServiceClass:             ServiceClassCode(fieldValueTrimmed(raw, 2, 4)),
			ForeignExchangeIndicator: fieldValueTrimmed(raw, 21, 22),
			OriginatorIdentification: fieldValueTrimmed(raw, 40, 49),
			SEC:                      StandardEntryClassCode(fieldValueTrimmed(raw, 50, 52)),
			ODFI:                     fieldValueTrimmed(raw, 80, 87),
			BatchNumber:              fieldValueTrimmed(raw, 88, 94),
		}
	}
	return &BatchHeader{
		recordBase:   recordBase{Raw: raw, LineNo: line, RecType: '5', Variant: "domestic"},
		ServiceClass: ServiceClassCode(fieldValueTrimmed(raw, 2, 4)),
		CompanyID:    fieldValueTrimmed(raw, 41, 50),
		SEC:          StandardEntryClassCode(fieldValueTrimmed(raw, 51, 53)),
		ODFI:         fieldValueTrimmed(raw, 80, 87),
		BatchNumber:  fieldValueTrimmed(raw, 88, 94),
	}
}

func parseEntryDetail(raw string, line int, batch *Batch, appendDiag func(line, start, end int, message string, severity Severity)) EntryRecord {
	base := entryBase{
		recordBase:           recordBase{Raw: raw, LineNo: line, RecType: '6'},
		TransactionCodeValue: TransactionCode(fieldValueTrimmed(raw, 2, 3)),
		ReceivingDFI:         fieldValue(raw, 4, 11),
		Amount:               parseIntField(raw, line, 30, 39, "amount", appendDiag),
		Addenda:              make([]AddendaRecord, 0),
	}
	parseIntField(raw, line, 4, 11, "receiving DFI identification", appendDiag)

	if batch == nil {
		return &EntryDetail{
			entryBase:     base,
			AccountNumber: fieldValueTrimmed(raw, 13, 29),
			TraceNumber:   fieldValueTrimmed(raw, 80, 94),
		}
	}

	if batch.Header != nil && batch.Header.IsInternational() {
		return &InternationalEntryDetail{
			entryBase:          base,
			AddendaRecordCount: parseIntField(raw, line, 13, 16, "addenda record count", appendDiag),
			AccountNumber:      fieldValueTrimmed(raw, 40, 74),
			TraceNumber:        fieldValueTrimmed(raw, 80, 94),
		}
	}

	sec := strings.ToUpper(strings.TrimSpace(string(batch.Header.SECCode())))
	switch sec {
	case string(SECCTX):
		return &CorporateTradeExchangeEntryDetail{
			entryBase:            base,
			AccountNumber:        fieldValueTrimmed(raw, 13, 29),
			NumberOfAddenda:      parseIntField(raw, line, 55, 58, "number of addenda", appendDiag),
			ReceivingCompanyName: fieldValueTrimmed(raw, 59, 74),
			TraceNumber:          fieldValueTrimmed(raw, 80, 94),
		}
	case string(SECCOR), string(SECACK), string(SECATX), string(SECADV), string(SECDNE), string(SECENR):
		return &ReturnIndividualEntry{
			entryBase:     base,
			AccountNumber: fieldValueTrimmed(raw, 13, 29),
			TraceNumber:   fieldValueTrimmed(raw, 80, 94),
		}
	default:
		return &EntryDetail{
			entryBase:     base,
			AccountNumber: fieldValueTrimmed(raw, 13, 29),
			TraceNumber:   fieldValueTrimmed(raw, 80, 94),
		}
	}
}

func parseAddenda(raw string, line int, batch *Batch) AddendaRecord {
	code := AddendaTypeCode(fieldValueTrimmed(raw, 2, 3))
	base := addendaBase{
		recordBase:  recordBase{Raw: raw, LineNo: line, RecType: '7'},
		AddendaType: code,
	}

	if batch != nil && batch.Header != nil && batch.Header.IsInternational() {
		switch code {
		case AddendaIAT10:
			return &InternationalAddenda10{addendaBase: base}
		case AddendaIAT11:
			return &InternationalAddenda11{addendaBase: base}
		case AddendaIAT12:
			return &InternationalAddenda12{addendaBase: base}
		case AddendaIAT13:
			return &InternationalAddenda13{addendaBase: base}
		case AddendaIAT14:
			return &InternationalAddenda14{addendaBase: base}
		case AddendaIAT15:
			return &InternationalAddenda15{addendaBase: base}
		case AddendaIAT16:
			return &InternationalAddenda16{addendaBase: base}
		case AddendaIAT17:
			return &InternationalAddenda17{addendaBase: base}
		case AddendaIAT18:
			return &InternationalAddenda18{addendaBase: base}
		case AddendaRET99:
			return &InternationalReturnAddenda99{addendaBase: base}
		}
	}

	switch code {
	case AddendaPOS02:
		return &PointOfSaleAddenda02{addendaBase: base}
	case AddendaPPD05:
		return &Addenda05{
			addendaBase:               base,
			PaymentRelatedInformation: fieldValueTrimmed(raw, 4, 83),
		}
	case AddendaNOC98:
		return &NotificationOfChangeAddenda98{addendaBase: base}
	case AddendaRET99:
		if strings.TrimSpace(fieldValue(raw, 50, 64)) != "" {
			return &DishonoredReturnAddenda99{addendaBase: base}
		}
		return &ReturnAddenda99{addendaBase: base}
	default:
		return &Addenda05{
			addendaBase:               base,
			PaymentRelatedInformation: fieldValueTrimmed(raw, 4, 83),
		}
	}
}

func parseBatchControl(raw string, line int, appendDiag func(line, start, end int, message string, severity Severity)) *BatchControl {
	return &BatchControl{
		recordBase:        recordBase{Raw: raw, LineNo: line, RecType: '8'},
		ServiceClassCode:  ServiceClassCode(fieldValueTrimmed(raw, 2, 4)),
		EntryAddendaCount: parseIntField(raw, line, 5, 10, "entry/addenda count", appendDiag),
		EntryHash:         fieldValueTrimmed(raw, 11, 20),
		TotalDebitAmount:  parseIntField(raw, line, 21, 32, "total debit amount", appendDiag),
		TotalCreditAmount: parseIntField(raw, line, 33, 44, "total credit amount", appendDiag),
		CompanyID:         fieldValueTrimmed(raw, 45, 54),
		ODFI:              fieldValueTrimmed(raw, 80, 87),
		BatchNumber:       fieldValueTrimmed(raw, 88, 94),
	}
}

func parseFileControl(raw string, line int, appendDiag func(line, start, end int, message string, severity Severity)) *FileControl {
	return &FileControl{
		recordBase:        recordBase{Raw: raw, LineNo: line, RecType: '9'},
		BatchCount:        parseIntField(raw, line, 2, 7, "batch count", appendDiag),
		BlockCount:        parseIntField(raw, line, 8, 13, "block count", appendDiag),
		EntryAddendaCount: parseIntField(raw, line, 14, 21, "entry/addenda count", appendDiag),
		EntryHash:         fieldValueTrimmed(raw, 22, 31),
		TotalDebitAmount:  parseIntField(raw, line, 32, 43, "total debit amount", appendDiag),
		TotalCreditAmount: parseIntField(raw, line, 44, 55, "total credit amount", appendDiag),
	}
}

func isInternationalBatchHeader(raw string) bool {
	if len(raw) < 52 || raw[0] != '5' {
		return false
	}
	fx := fieldValueTrimmed(raw, 21, 22)
	sec := fieldValueTrimmed(raw, 50, 52)
	return (fx == "FF" || fx == "FV" || fx == "VF") && strings.EqualFold(sec, string(SECIAT))
}
