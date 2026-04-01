package nacha

import "strings"

const (
	recordLength = 94
	blockSize    = 10
)

type FieldSpec struct {
	Name        string
	Start       int
	End         int
	Description string
}

type PositionInfo struct {
	RecordType byte
	RecordName string
	Position   int
	Field      *FieldSpec
	FieldValue string
}

func LookupPosition(record string, column int) (PositionInfo, bool) {
	if len(record) == 0 {
		return PositionInfo{}, false
	}

	recordType := record[0]
	recordName, ok := recordDescriptions[recordType]
	if !ok {
		return PositionInfo{}, false
	}

	if column < 0 {
		column = 0
	}
	position := column + 1

	info := PositionInfo{
		RecordType: recordType,
		RecordName: recordName,
		Position:   position,
	}

	for _, field := range baseRecordFields[recordType] {
		if position < field.Start || position > field.End {
			continue
		}
		field := field
		info.Field = &field
		info.FieldValue = fieldValueTrimmed(record, field.Start, field.End)
		return info, true
	}

	return info, true
}

var baseRecordFields = map[byte][]FieldSpec{
	'1': {
		{Name: "Record Type Code", Start: 1, End: 1, Description: "File Header record identifier; must be `1`."},
		{Name: "Priority Code", Start: 2, End: 3, Description: "Priority code; expected `01`."},
		{Name: "Immediate Destination", Start: 4, End: 13, Description: "Routing number of destination financial institution."},
		{Name: "Immediate Origin", Start: 14, End: 23, Description: "Originating financial institution identifier."},
		{Name: "File Creation Date", Start: 24, End: 29, Description: "Creation date in YYMMDD."},
		{Name: "File Creation Time", Start: 30, End: 33, Description: "Creation time in HHMM."},
		{Name: "File ID Modifier", Start: 34, End: 34, Description: "Uniqueness discriminator for files created the same day."},
		{Name: "Record Size", Start: 35, End: 37, Description: "Must be `094`."},
		{Name: "Blocking Factor", Start: 38, End: 39, Description: "Must be `10`."},
		{Name: "Format Code", Start: 40, End: 40, Description: "Must be `1`."},
		{Name: "Immediate Destination Name", Start: 41, End: 63, Description: "Destination institution name."},
		{Name: "Immediate Origin Name", Start: 64, End: 86, Description: "Origin institution name."},
		{Name: "Reference Code", Start: 87, End: 94, Description: "Optional reference code."},
	},
	'5': {
		{Name: "Record Type Code", Start: 1, End: 1, Description: "Batch Header record identifier; must be `5`."},
		{Name: "Service Class Code", Start: 2, End: 4, Description: "Batch class: `200` mixed, `220` credits, `225` debits."},
		{Name: "Company Name", Start: 5, End: 20, Description: "Company name associated with the batch."},
		{Name: "Company Discretionary Data", Start: 21, End: 40, Description: "Optional company-defined information."},
		{Name: "Company Identification", Start: 41, End: 50, Description: "Company identification used in the batch."},
		{Name: "Standard Entry Class Code", Start: 51, End: 53, Description: "SEC code (for example `PPD`, `CCD`, `CTX`, `IAT`)."},
		{Name: "Company Entry Description", Start: 54, End: 63, Description: "Description of entries in the batch."},
		{Name: "Company Descriptive Date", Start: 64, End: 69, Description: "Optional company-defined date description."},
		{Name: "Effective Entry Date", Start: 70, End: 75, Description: "Effective date in YYMMDD."},
		{Name: "Settlement Date (Julian)", Start: 76, End: 78, Description: "Reserved for ACH operator settlement date."},
		{Name: "Originator Status Code", Start: 79, End: 79, Description: "Originator status code; typically `1`."},
		{Name: "Originating DFI Identification", Start: 80, End: 87, Description: "ODFI routing prefix."},
		{Name: "Batch Number", Start: 88, End: 94, Description: "Batch sequence number."},
	},
	'6': {
		{Name: "Record Type Code", Start: 1, End: 1, Description: "Entry Detail record identifier; must be `6`."},
		{Name: "Transaction Code", Start: 2, End: 3, Description: "Account type and debit/credit code."},
		{Name: "Receiving DFI Identification", Start: 4, End: 11, Description: "First 8 digits of RDFI routing number."},
		{Name: "Check Digit", Start: 12, End: 12, Description: "Ninth digit of RDFI routing number."},
		{Name: "DFI Account Number", Start: 13, End: 29, Description: "Receiver account number at the RDFI."},
		{Name: "Amount", Start: 30, End: 39, Description: "Entry amount in cents, right-justified and zero-filled."},
		{Name: "Identification Number", Start: 40, End: 54, Description: "Optional originator identification for the entry."},
		{Name: "Individual Name", Start: 55, End: 76, Description: "Receiver or company name, depending on SEC code."},
		{Name: "Discretionary Data", Start: 77, End: 78, Description: "Optional sender-defined data."},
		{Name: "Addenda Record Indicator", Start: 79, End: 79, Description: "Indicates whether addenda records follow (`0` or `1`)."},
		{Name: "Trace Number", Start: 80, End: 94, Description: "ODFI trace number for the entry."},
	},
	'7': {
		{Name: "Record Type Code", Start: 1, End: 1, Description: "Addenda record identifier; must be `7`."},
		{Name: "Addenda Type Code", Start: 2, End: 3, Description: "Addenda discriminator (`02`, `05`, `10`-`18`, `98`, `99`)."},
		{Name: "Payment Related Information", Start: 4, End: 83, Description: "Addenda-specific payload."},
		{Name: "Addenda Sequence Number", Start: 84, End: 87, Description: "Sequence number of this addenda within the entry."},
		{Name: "Entry Detail Sequence Number", Start: 88, End: 94, Description: "Last 7 digits of related entry trace number."},
	},
	'8': {
		{Name: "Record Type Code", Start: 1, End: 1, Description: "Batch Control record identifier; must be `8`."},
		{Name: "Service Class Code", Start: 2, End: 4, Description: "Must match corresponding batch header."},
		{Name: "Entry/Addenda Count", Start: 5, End: 10, Description: "Count of detail and addenda records in batch."},
		{Name: "Entry Hash", Start: 11, End: 20, Description: "Hash sum of RDFI prefixes, modulo 10 digits."},
		{Name: "Total Debit Amount", Start: 21, End: 32, Description: "Total debit amount in cents."},
		{Name: "Total Credit Amount", Start: 33, End: 44, Description: "Total credit amount in cents."},
		{Name: "Company ID", Start: 45, End: 54, Description: "Company identifier."},
		{Name: "Message Authentication Code", Start: 55, End: 73, Description: "Message Authentication Code or reserved."},
		{Name: "Reserved", Start: 74, End: 79, Description: "Reserved; must be blank."},
		{Name: "Originating DFI Identification", Start: 80, End: 87, Description: "ODFI routing prefix."},
		{Name: "Batch Number", Start: 88, End: 94, Description: "Batch sequence number."},
	},
	'9': {
		{Name: "Record Type Code", Start: 1, End: 1, Description: "File Control or padding record identifier; starts with `9`."},
		{Name: "Batch Count", Start: 2, End: 7, Description: "Total batches in file."},
		{Name: "Block Count", Start: 8, End: 13, Description: "Total 10-record blocks in file."},
		{Name: "Entry/Addenda Count", Start: 14, End: 21, Description: "Total entry and addenda records in file."},
		{Name: "Entry Hash", Start: 22, End: 31, Description: "File-level entry hash."},
		{Name: "Total Debit Amount", Start: 32, End: 43, Description: "Total debit amount in cents."},
		{Name: "Total Credit Amount", Start: 44, End: 55, Description: "Total credit amount in cents."},
		{Name: "Reserved", Start: 56, End: 94, Description: "Reserved; should be blank in file control or all `9` in padding lines."},
	},
}

var recordDescriptions = map[byte]string{
	'1': "File Header",
	'5': "Batch Header",
	'6': "Entry Detail",
	'7': "Addenda",
	'8': "Batch Control",
	'9': "File Control or Padding",
}

func fieldValue(raw string, start, end int) string {
	if start < 1 {
		start = 1
	}
	if end > len(raw) {
		end = len(raw)
	}
	if start > end || start > len(raw) {
		return ""
	}
	return raw[start-1 : end]
}

func fieldValueTrimmed(raw string, start, end int) string {
	return strings.TrimSpace(fieldValue(raw, start, end))
}

func isAllNines(raw string) bool {
	if len(raw) == 0 {
		return false
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] != '9' {
			return false
		}
	}
	return true
}

func formatUnsigned(value int64, width int) string {
	if value < 0 {
		value = 0
	}
	s := strings.TrimSpace(int64ToString(value))
	if len(s) > width {
		return s[len(s)-width:]
	}
	if len(s) < width {
		return strings.Repeat("0", width-len(s)) + s
	}
	return s
}
