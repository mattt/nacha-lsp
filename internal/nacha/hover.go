package nacha

import (
	"fmt"
	"strings"
)

type fieldInfo struct {
	start       int
	end         int
	name        string
	description string
}

var recordFields = map[byte][]fieldInfo{
	'1': {
		{start: 1, end: 1, name: "Record Type Code", description: "File Header record identifier; must be `1`."},
		{start: 4, end: 13, name: "Immediate Destination", description: "Routing number of the destination bank."},
		{start: 14, end: 23, name: "Immediate Origin", description: "Routing number or identifier of the originating bank."},
		{start: 24, end: 29, name: "File Creation Date", description: "YYMMDD date the file was created."},
		{start: 30, end: 33, name: "File Creation Time", description: "HHMM time the file was created."},
	},
	'5': {
		{start: 1, end: 1, name: "Record Type Code", description: "Batch Header record identifier; must be `5`."},
		{start: 2, end: 4, name: "Service Class Code", description: "Batch class: `200` mixed, `220` credits, `225` debits."},
		{start: 41, end: 50, name: "Company Identification", description: "Company ID for the originator in this batch."},
		{start: 51, end: 53, name: "Standard Entry Class Code", description: "SEC code describing the payment type."},
		{start: 70, end: 75, name: "Effective Entry Date", description: "Requested settlement date (YYMMDD)."},
	},
	'6': {
		{start: 1, end: 1, name: "Record Type Code", description: "Entry Detail record identifier; must be `6`."},
		{start: 2, end: 3, name: "Transaction Code", description: "Account type and debit/credit indicator (for example `22`, `27`, `32`, `37`)."},
		{start: 4, end: 11, name: "Receiving DFI Identification", description: "First 8 digits of receiving bank routing number."},
		{start: 12, end: 12, name: "Check Digit", description: "Ninth digit of receiving bank routing number."},
		{start: 13, end: 29, name: "DFI Account Number", description: "Receiver account number."},
		{start: 30, end: 39, name: "Amount", description: "Amount in cents, unsigned and zero-padded."},
		{start: 80, end: 94, name: "Trace Number", description: "Entry trace number, ascending within batch."},
	},
	'8': {
		{start: 1, end: 1, name: "Record Type Code", description: "Batch Control record identifier; must be `8`."},
		{start: 2, end: 4, name: "Service Class Code", description: "Must match the batch header service class code."},
		{start: 5, end: 10, name: "Entry/Addenda Count", description: "Count of `6` and `7` records in this batch."},
		{start: 11, end: 20, name: "Entry Hash", description: "Hash total of routing number prefixes from entry records."},
		{start: 21, end: 32, name: "Total Debit Entry Dollar Amount", description: "Total debits in cents for the batch."},
		{start: 33, end: 44, name: "Total Credit Entry Dollar Amount", description: "Total credits in cents for the batch."},
	},
	'9': {
		{start: 1, end: 1, name: "Record Type Code", description: "File Control (or padding) record identifier; begins with `9`."},
		{start: 2, end: 7, name: "Batch Count", description: "Total number of batch records in the file control record."},
		{start: 8, end: 13, name: "Block Count", description: "Total number of 10-record blocks in the file."},
		{start: 14, end: 21, name: "Entry/Addenda Count", description: "Total number of entry and addenda records."},
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

func HoverAt(text string, line, character int) string {
	lines := splitLines(text)
	if line < 0 || line >= len(lines) {
		return ""
	}

	record := lines[line]
	if len(record) == 0 {
		return ""
	}
	recordType := record[0]
	desc, ok := recordDescriptions[recordType]
	if !ok {
		return ""
	}

	if character < 0 {
		character = 0
	}
	position := character + 1

	fields := recordFields[recordType]
	for _, field := range fields {
		if position < field.start || position > field.end {
			continue
		}

		value := sliceValue(record, field.start, field.end)
		return fmt.Sprintf(
			"**%s** (`%c`)  \n**Field:** %s (positions %d-%d)  \n**Value:** `%s`  \n%s",
			desc,
			recordType,
			field.name,
			field.start,
			field.end,
			value,
			field.description,
		)
	}

	return fmt.Sprintf("**%s** (`%c`)  \nPosition %d has no MVP hover metadata yet.", desc, recordType, position)
}

func sliceValue(record string, start, end int) string {
	if start < 1 {
		start = 1
	}
	if end > len(record) {
		end = len(record)
	}
	if start > end || start > len(record) {
		return ""
	}
	value := record[start-1 : end]
	return strings.TrimSpace(value)
}
