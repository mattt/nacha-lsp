// Package nacha provides types and functions for parsing, validating,
// and serializing ACH (Automated Clearing House) files as defined by
// the Nacha Operating Rules.
//
// ACH files are fixed-width ASCII files in which each line (record) is
// exactly 94 characters long. A file begins with a File Header record
// (type 1), contains one or more batches delimited by Batch Header (type 5)
// and Batch Control (type 8) records, and closes with a File Control record
// (type 9). Entry Detail records (type 6) carry individual transactions within
// a batch, and optional Addenda records (type 7) carry supplemental payment
// information. Padding records consisting entirely of nines are appended after
// the File Control record to pad the total line count to a multiple of ten
// (the blocking factor).
package nacha

import (
	"bytes"
	"io"
)

// ServiceClassCode identifies the general classification of dollar entries
// within a batch. It appears in positions 2–4 of the Batch Header (type 5)
// and Batch Control (type 8) records, and the value must be the same in both.
//
// Valid values are:
//   - "200" — mixed debits and credits
//   - "220" — credits only
//   - "225" — debits only
type ServiceClassCode string

// StandardEntryClassCode identifies the ACH application type for all entries
// within a batch. It appears in positions 51–53 of the Batch Header (type 5).
// The SEC code determines the entry record layout, the permitted addenda types,
// and the authorization requirements for the transaction.
type StandardEntryClassCode string

// TransactionCode identifies the account type and direction (debit or credit)
// of an Entry Detail record (type 6). It appears in positions 2–3 and is a
// mandatory numeric field.
//
// Common values:
//   - "22" — credit to checking account
//   - "27" — debit from checking account
//   - "32" — credit to savings account
//   - "37" — debit from savings account
type TransactionCode string

// AddendaTypeCode discriminates Addenda record (type 7) variants.
// It appears in positions 2–3 of every addenda record.
//
// Common values:
//   - "02" — Point-of-Sale addenda
//   - "05" — general payment-related information (PPD, CCD, WEB)
//   - "98" — Notification of Change
//   - "99" — return or dishonored return
//   - "10"–"18" — mandatory IAT addenda sequence
type AddendaTypeCode string

const (
	// SECCTX identifies a Corporate Trade Exchange batch. CTX entries support
	// up to 9,999 addenda records per entry and are used to carry full ANSI
	// ASC X12 or UN/EDIFACT messages between trading partners.
	SECCTX StandardEntryClassCode = "CTX"

	// SECIAT identifies an International ACH Transaction batch. IAT entries
	// involve a financial agency's office outside the territorial jurisdiction
	// of the United States and require a fixed sequence of addenda records
	// (types 10–18).
	SECIAT StandardEntryClassCode = "IAT"

	// SECCOR identifies a Notification of Change (formerly COR) batch, used
	// when an RDFI needs to inform an ODFI of incorrect or changed entry data.
	SECCOR StandardEntryClassCode = "COR"

	// SECACK identifies an Acknowledgment Entry for CCD batch.
	SECACK StandardEntryClassCode = "ACK"

	// SECATX identifies an Acknowledgment Entry for CTX batch.
	SECATX StandardEntryClassCode = "ATX"

	// SECADV identifies an Automated Accounting Advice batch.
	SECADV StandardEntryClassCode = "ADV"

	// SECDNE identifies a Death Notification Entry batch.
	SECDNE StandardEntryClassCode = "DNE"

	// SECENR identifies an Automated Enrollment Entry batch.
	SECENR StandardEntryClassCode = "ENR"
)

const (
	// AddendaPOS02 identifies a Point-of-Sale addenda record (type 7, code 02).
	AddendaPOS02 AddendaTypeCode = "02"

	// AddendaPPD05 identifies a general payment-related information addenda
	// record (type 7, code 05), used with PPD, CCD, and WEB entries. Only
	// one such record is allowed per entry for PPD and CCD.
	AddendaPPD05 AddendaTypeCode = "05"

	// AddendaNOC98 identifies a Notification of Change addenda record
	// (type 7, code 98), which carries the corrected account data returned
	// by the RDFI.
	AddendaNOC98 AddendaTypeCode = "98"

	// AddendaRET99 identifies a return or dishonored return addenda record
	// (type 7, code 99).
	AddendaRET99 AddendaTypeCode = "99"

	// AddendaIAT10 through AddendaIAT18 identify the sequential mandatory
	// addenda records required for IAT (International ACH Transaction) entries.
	AddendaIAT10 AddendaTypeCode = "10"
	AddendaIAT11 AddendaTypeCode = "11"
	AddendaIAT12 AddendaTypeCode = "12"
	AddendaIAT13 AddendaTypeCode = "13"
	AddendaIAT14 AddendaTypeCode = "14"
	AddendaIAT15 AddendaTypeCode = "15"
	AddendaIAT16 AddendaTypeCode = "16"
	AddendaIAT17 AddendaTypeCode = "17"
	AddendaIAT18 AddendaTypeCode = "18"
)

// Record is the common interface satisfied by all ACH record types.
// Every record stores its raw 94-character source line, its 0-based
// line number within the file, and its type code byte.
type Record interface {
	// RecordType returns the single-byte record type code: '1' (File Header),
	// '5' (Batch Header), '6' (Entry Detail), '7' (Addenda), '8' (Batch
	// Control), or '9' (File Control / padding).
	RecordType() byte

	// Dump returns the raw 94-character source line for this record.
	Dump() string

	// Line returns the 0-based line number of this record within the file.
	Line() int
}

type recordBase struct {
	Raw      string
	LineNo   int
	RecType  byte
	Variant  string
	Sequence int
}

func (r recordBase) RecordType() byte { return r.RecType }
func (r recordBase) Dump() string     { return r.Raw }
func (r recordBase) Line() int        { return r.LineNo }

// File represents a parsed ACH file in structured form alongside the flat
// sequence of all records in source order.
//
// A well-formed file has exactly one Header and one Control, at least one
// Batch, and zero or more Padding records appended to reach a line count
// that is a multiple of ten (the blocking factor).
type File struct {
	Header  *FileHeader
	Control *FileControl

	Batches []*Batch
	Padding []PaddingRecord
	Records []Record
}

// Serialize returns the file as a newline-separated string of 94-character
// records in source order. It is a convenience wrapper around [WriteFile].
func (f *File) Serialize() string {
	return SerializeFile(f)
}

// WriteTo writes the file to w as newline-separated 94-character records in
// source order. It implements [io.WriterTo].
func (f *File) WriteTo(w io.Writer) (int64, error) {
	return WriteFile(w, f)
}

// ReadFrom parses an ACH file from r and replaces the receiver's contents
// with the result. It implements [io.ReaderFrom].
func (f *File) ReadFrom(r io.Reader) (int64, error) {
	var buf bytes.Buffer
	n, err := io.Copy(&buf, r)
	if err != nil {
		return n, err
	}
	result := Parse(buf.String())
	*f = *result.File
	return n, nil
}

// FileHeader represents the File Header record (type 1). It must appear
// exactly once as the first record of every ACH file.
//
// Key fields and their 1-based positions in the Nacha specification:
//   - Immediate Destination (4–13): routing number of the receiving bank,
//     preceded by a blank space.
//   - Immediate Origin (14–23): originating institution identifier, often
//     a routing number preceded by a blank space.
//   - File Creation Date (24–29): creation date in YYMMDD format.
//   - File Creation Time (30–33): creation time in HHMM format (optional).
//   - File ID Modifier (34): single alphanumeric character that distinguishes
//     multiple files submitted by the same institution on the same day.
type FileHeader struct {
	recordBase
	ImmediateDestination string
	ImmediateOrigin      string
	FileCreationDate     string
	FileCreationTime     string
	FileIDModifier       string
}

// Batch represents a single ACH batch, which is delimited by a Batch Header
// record (type 5) and a Batch Control record (type 8). All entries within a
// batch share the same SEC code, effective entry date, company ID, and
// company entry description.
type Batch struct {
	Header  BatchHeaderRecord
	Control *BatchControl
	Entries []EntryRecord
}

// BatchHeaderRecord is the common interface for Batch Header records (type 5).
// Domestic batches are represented by [*BatchHeader]; international batches
// (SEC code IAT) are represented by [*InternationalBatchHeader].
type BatchHeaderRecord interface {
	Record

	// SECCode returns the Standard Entry Class code for the batch.
	SECCode() StandardEntryClassCode

	// IsInternational reports whether this is an IAT batch header, which
	// requires a different entry detail layout and mandatory addenda records.
	IsInternational() bool

	// ServiceClassCode returns the service class code that classifies the
	// batch as mixed (200), credits-only (220), or debits-only (225).
	ServiceClassCode() ServiceClassCode
}

// BatchHeader is the Batch Header record (type 5) for domestic ACH batches.
// International batches (SEC code IAT) use [InternationalBatchHeader] instead.
type BatchHeader struct {
	recordBase
	ServiceClass ServiceClassCode
	CompanyID    string
	SEC          StandardEntryClassCode
	ODFI         string
	BatchNumber  string
}

func (h *BatchHeader) SECCode() StandardEntryClassCode    { return h.SEC }
func (h *BatchHeader) IsInternational() bool              { return false }
func (h *BatchHeader) ServiceClassCode() ServiceClassCode { return h.ServiceClass }

// InternationalBatchHeader is the Batch Header record (type 5) for IAT
// (International ACH Transaction) batches. It differs from [BatchHeader] in
// that the Company Discretionary Data field carries a Foreign Exchange
// Indicator, and each entry requires a fixed sequence of addenda records
// (types 10–18) describing the foreign financial institution.
type InternationalBatchHeader struct {
	recordBase
	ServiceClass             ServiceClassCode
	ForeignExchangeIndicator string
	OriginatorIdentification string
	SEC                      StandardEntryClassCode
	ODFI                     string
	BatchNumber              string
}

func (h *InternationalBatchHeader) SECCode() StandardEntryClassCode    { return h.SEC }
func (h *InternationalBatchHeader) IsInternational() bool              { return true }
func (h *InternationalBatchHeader) ServiceClassCode() ServiceClassCode { return h.ServiceClass }

// BatchControl is the Batch Control record (type 8) that closes each batch.
// Its integrity fields — entry/addenda count, entry hash, and debit and
// credit totals — must match the corresponding records within the batch.
type BatchControl struct {
	recordBase
	ServiceClassCode  ServiceClassCode
	EntryAddendaCount int64
	EntryHash         string
	TotalDebitAmount  int64
	TotalCreditAmount int64
	CompanyID         string
	ODFI              string
	BatchNumber       string
}

// FileControl is the File Control record (type 9) that closes the ACH file.
// It contains file-level integrity totals: batch count, block count,
// entry/addenda count, entry hash sum, and total debit and credit amounts.
type FileControl struct {
	recordBase
	BatchCount        int64
	BlockCount        int64
	EntryAddendaCount int64
	EntryHash         string
	TotalDebitAmount  int64
	TotalCreditAmount int64
}

// PaddingRecord represents an all-nines record appended after the File Control
// record to pad the total line count to a multiple of ten (the blocking
// factor of 10 required by the Nacha specification).
type PaddingRecord struct {
	recordBase
}

// EntryRecord is the common interface for Entry Detail records (type 6).
// The concrete type varies by SEC code:
//   - [*EntryDetail] for most domestic entries (PPD, CCD, WEB, TEL, etc.)
//   - [*CorporateTradeExchangeEntryDetail] for CTX entries
//   - [*InternationalEntryDetail] for IAT entries
//   - [*ReturnIndividualEntry], [*ReturnCorporateEntry], and
//     [*ReturnInternationalEntry] for return batches
type EntryRecord interface {
	Record

	// TransactionCode returns the two-digit code that identifies the account
	// type (checking or savings) and direction (debit or credit) of the entry.
	TransactionCode() TransactionCode

	// ReceivingDFIPrefix returns the first eight digits of the Receiving
	// Depository Financial Institution's routing number. These eight digits
	// are summed across all entries in a batch to produce the Entry Hash.
	ReceivingDFIPrefix() string

	// AmountCents returns the transaction amount in cents.
	AmountCents() int64

	// AddendaRecords returns the addenda records associated with this entry.
	AddendaRecords() []AddendaRecord

	// SetAddenda replaces the set of addenda records for this entry.
	SetAddenda([]AddendaRecord)
}

type entryBase struct {
	recordBase
	TransactionCodeValue TransactionCode
	ReceivingDFI         string
	Amount               int64
	Addenda              []AddendaRecord
}

func (e *entryBase) TransactionCode() TransactionCode { return e.TransactionCodeValue }
func (e *entryBase) ReceivingDFIPrefix() string       { return e.ReceivingDFI }
func (e *entryBase) AmountCents() int64               { return e.Amount }
func (e *entryBase) AddendaRecords() []AddendaRecord  { return e.Addenda }
func (e *entryBase) SetAddenda(addenda []AddendaRecord) {
	e.Addenda = addenda
}

// EntryDetail is the standard Entry Detail record (type 6) used for most
// domestic SEC codes, including PPD (Prearranged Payment and Deposit),
// CCD (Corporate Credit or Debit), WEB (Internet-Initiated), and
// TEL (Telephone-Initiated).
type EntryDetail struct {
	entryBase
	AccountNumber string
	TraceNumber   string
}

// CorporateTradeExchangeEntryDetail is the Entry Detail record (type 6) for
// CTX (Corporate Trade Exchange) batches. CTX entries may carry up to 9,999
// addenda records per entry, so the record layout includes an explicit
// Number of Addenda field and a Receiving Company Name in place of the
// standard Individual Name field.
type CorporateTradeExchangeEntryDetail struct {
	entryBase
	AccountNumber        string
	NumberOfAddenda      int64
	ReceivingCompanyName string
	TraceNumber          string
}

// InternationalEntryDetail is the Entry Detail record (type 6) for IAT
// (International ACH Transaction) batches. Each IAT entry must be accompanied
// by a fixed sequence of addenda records (types 10–18) that describe the
// foreign financial institution and transaction parties.
type InternationalEntryDetail struct {
	entryBase
	AddendaRecordCount int64
	AccountNumber      string
	TraceNumber        string
}

// ReturnIndividualEntry is the Entry Detail record (type 6) for return
// batches that carry individual (consumer) entries.
type ReturnIndividualEntry struct {
	entryBase
	AccountNumber string
	TraceNumber   string
}

// ReturnCorporateEntry is the Entry Detail record (type 6) for return
// batches that carry corporate entries with addenda records.
type ReturnCorporateEntry struct {
	entryBase
	AccountNumber   string
	NumberOfAddenda int64
	TraceNumber     string
}

// ReturnInternationalEntry is the Entry Detail record (type 6) for return
// batches that carry IAT (International ACH Transaction) entries.
type ReturnInternationalEntry struct {
	entryBase
	AddendaRecordCount int64
	AccountNumber      string
	TraceNumber        string
}

// AddendaRecord is the common interface for Addenda records (type 7).
// The concrete type is determined by the two-digit Addenda Type Code in
// positions 2–3 of the record.
type AddendaRecord interface {
	Record

	// AddendaTypeCode returns the two-digit code that discriminates the
	// addenda record variant (e.g. "05", "98", "99", "10"–"18").
	AddendaTypeCode() AddendaTypeCode
}

type addendaBase struct {
	recordBase
	AddendaType AddendaTypeCode
}

func (a *addendaBase) AddendaTypeCode() AddendaTypeCode { return a.AddendaType }

// Addenda05 is the general-purpose Addenda record (type 7, code 05) for PPD,
// CCD, and WEB entries. It carries up to 80 characters of free-form payment-
// related information that accompanies the entry to the RDFI. Only one
// Addenda05 record is permitted per PPD or CCD entry.
type Addenda05 struct {
	addendaBase
	PaymentRelatedInformation string
}

// PointOfSaleAddenda02 is the Addenda record (type 7, code 02) for POS
// (Point of Sale) entries.
type PointOfSaleAddenda02 struct{ addendaBase }

// NotificationOfChangeAddenda98 is the Addenda record (type 7, code 98) for
// COR (Notification of Change) entries. It carries the corrected account or
// routing information that the RDFI is returning to the ODFI.
type NotificationOfChangeAddenda98 struct{ addendaBase }

// InternationalAddenda10 is the first mandatory Addenda record (type 7,
// code 10) for IAT entries, carrying transaction type code and foreign payment
// amount information.
type InternationalAddenda10 struct{ addendaBase }

// InternationalAddenda11 is the second mandatory Addenda record (type 7,
// code 11) for IAT entries, carrying the originator's name and address.
type InternationalAddenda11 struct{ addendaBase }

// InternationalAddenda12 is the third mandatory Addenda record (type 7,
// code 12) for IAT entries, carrying the originator's city, state, and
// country.
type InternationalAddenda12 struct{ addendaBase }

// InternationalAddenda13 is the fourth mandatory Addenda record (type 7,
// code 13) for IAT entries, carrying the originating DFI name and
// identification.
type InternationalAddenda13 struct{ addendaBase }

// InternationalAddenda14 is the fifth mandatory Addenda record (type 7,
// code 14) for IAT entries, carrying the receiving DFI name and
// identification.
type InternationalAddenda14 struct{ addendaBase }

// InternationalAddenda15 is the sixth mandatory Addenda record (type 7,
// code 15) for IAT entries, carrying the receiver's name and address.
type InternationalAddenda15 struct{ addendaBase }

// InternationalAddenda16 is the seventh mandatory Addenda record (type 7,
// code 16) for IAT entries, carrying the receiver's city, state, and country.
type InternationalAddenda16 struct{ addendaBase }

// InternationalAddenda17 is an optional Addenda record (type 7, code 17) for
// IAT entries, carrying remittance information.
type InternationalAddenda17 struct{ addendaBase }

// InternationalAddenda18 is an optional Addenda record (type 7, code 18) for
// IAT entries, carrying foreign correspondent bank information.
type InternationalAddenda18 struct{ addendaBase }

// ReturnAddenda99 is the Addenda record (type 7, code 99) for standard return
// entries. It carries the return reason code and original entry trace number.
type ReturnAddenda99 struct{ addendaBase }

// DishonoredReturnAddenda99 is the Addenda record (type 7, code 99) for
// dishonored return entries. It is distinguished from [ReturnAddenda99] by
// the presence of a non-blank Original Return Reason Code field
// (positions 51–64 of the record).
type DishonoredReturnAddenda99 struct{ addendaBase }

// InternationalReturnAddenda99 is the Addenda record (type 7, code 99) for
// IAT (International ACH Transaction) return entries.
type InternationalReturnAddenda99 struct{ addendaBase }
