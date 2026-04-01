package nacha

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readValidFixture(t *testing.T) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("testdata", "sample-valid.ach"))
	if err != nil {
		t.Fatalf("read valid fixture: %v", err)
	}
	return string(content)
}

func validDomesticFile() string {
	lines := []string{
		makeFileHeader(),
		makeBatchHeader("200", "PPD"),
		makeEntry("22", "03130001", 1000),
		makeBatchControl("200", 1, "0003130001", 0, 1000),
		makeFileControl(1, 1, 1, "0003130001", 0, 1000),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
		strings.Repeat("9", 94),
	}
	return strings.Join(lines, "\n")
}

func makeRecord(recordType byte) string {
	b := []byte(strings.Repeat(" ", 94))
	b[0] = recordType
	return string(b)
}

func writeField(line []byte, start, end int, value string) {
	width := end - start + 1
	if len(value) > width {
		value = value[:width]
	}
	padded := value + strings.Repeat(" ", width-len(value))
	copy(line[start-1:end], []byte(padded))
}

func writeNumeric(line []byte, start, end int, value int64) {
	width := end - start + 1
	s := formatUnsigned(value, width)
	copy(line[start-1:end], []byte(s))
}

func makeFileHeader() string {
	line := []byte(makeRecord('1'))
	writeField(line, 2, 3, "01")
	writeField(line, 4, 13, " 031300012")
	writeField(line, 14, 23, "1234567890")
	writeField(line, 24, 29, "260401")
	writeField(line, 30, 33, "1200")
	writeField(line, 34, 34, "A")
	writeField(line, 35, 37, "094")
	writeField(line, 38, 39, "10")
	writeField(line, 40, 40, "1")
	writeField(line, 41, 63, "DEST BANK")
	writeField(line, 64, 86, "ORIGIN CO")
	return string(line)
}

func makeBatchHeader(serviceClass, sec string) string {
	line := []byte(makeRecord('5'))
	writeField(line, 2, 4, serviceClass)
	writeField(line, 5, 20, "ACME COMPANY")
	writeField(line, 41, 50, "1234567890")
	writeField(line, 51, 53, sec)
	writeField(line, 54, 63, "PAYROLL")
	writeField(line, 70, 75, "260401")
	writeField(line, 78, 78, "1")
	writeField(line, 80, 87, "12345678")
	writeField(line, 88, 94, "0000001")
	return string(line)
}

func makeEntry(transactionCode, rdfiPrefix string, amount int64) string {
	line := []byte(makeRecord('6'))
	writeField(line, 2, 3, transactionCode)
	writeField(line, 4, 11, rdfiPrefix)
	writeField(line, 12, 12, "2")
	writeField(line, 13, 29, "987654321")
	writeNumeric(line, 30, 39, amount)
	writeField(line, 40, 54, "EMP001")
	writeField(line, 55, 76, "JOHN DOE")
	writeField(line, 79, 79, "0")
	writeField(line, 80, 94, "123456780000001")
	return string(line)
}

func makeBatchControl(serviceClass string, count int64, hash string, debit, credit int64) string {
	line := []byte(makeRecord('8'))
	writeField(line, 2, 4, serviceClass)
	writeNumeric(line, 5, 10, count)
	writeField(line, 11, 20, hash)
	writeNumeric(line, 21, 32, debit)
	writeNumeric(line, 33, 44, credit)
	writeField(line, 45, 54, "1234567890")
	writeField(line, 80, 87, "12345678")
	writeField(line, 88, 94, "0000001")
	return string(line)
}

func makeFileControl(batchCount, blockCount, entryCount int64, hash string, debit, credit int64) string {
	line := []byte(makeRecord('9'))
	writeNumeric(line, 2, 7, batchCount)
	writeNumeric(line, 8, 13, blockCount)
	writeNumeric(line, 14, 21, entryCount)
	writeField(line, 22, 31, hash)
	writeNumeric(line, 32, 43, debit)
	writeNumeric(line, 44, 55, credit)
	return string(line)
}

func makeIATBatchHeader() string {
	line := []byte(makeRecord('5'))
	writeField(line, 2, 4, "200")
	writeField(line, 21, 22, "FF")
	writeField(line, 40, 49, "1234567890")
	writeField(line, 50, 52, "IAT")
	writeField(line, 80, 87, "12345678")
	writeField(line, 88, 94, "0000001")
	return string(line)
}

func makeIATEntry() string {
	line := []byte(makeRecord('6'))
	writeField(line, 2, 3, "22")
	writeField(line, 4, 11, "03130001")
	writeNumeric(line, 13, 16, 1)
	writeNumeric(line, 30, 39, 1000)
	writeField(line, 40, 74, "FOREIGN-ACCOUNT-123")
	writeField(line, 80, 94, "123456780000001")
	return string(line)
}

func makeIATAddenda(code string) string {
	line := []byte(makeRecord('7'))
	writeField(line, 2, 3, code)
	writeField(line, 4, 83, "IAT ADDENDA")
	return string(line)
}
