package nacha

import (
	"bytes"
	"io"
)

type ServiceClassCode string
type StandardEntryClassCode string
type TransactionCode string
type AddendaTypeCode string

const (
	SECCTX StandardEntryClassCode = "CTX"
	SECIAT StandardEntryClassCode = "IAT"
	SECCOR StandardEntryClassCode = "COR"
	SECACK StandardEntryClassCode = "ACK"
	SECATX StandardEntryClassCode = "ATX"
	SECADV StandardEntryClassCode = "ADV"
	SECDNE StandardEntryClassCode = "DNE"
	SECENR StandardEntryClassCode = "ENR"
)

const (
	AddendaPOS02 AddendaTypeCode = "02"
	AddendaPPD05 AddendaTypeCode = "05"
	AddendaNOC98 AddendaTypeCode = "98"
	AddendaRET99 AddendaTypeCode = "99"
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

type Record interface {
	RecordType() byte
	Dump() string
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

type File struct {
	Header  *FileHeader
	Control *FileControl

	Batches []*Batch
	Padding []PaddingRecord
	Records []Record
}

func (f *File) Serialize() string {
	return SerializeFile(f)
}

func (f *File) WriteTo(w io.Writer) (int64, error) {
	return WriteFile(w, f)
}

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

type FileHeader struct {
	recordBase
	ImmediateDestination string
	ImmediateOrigin      string
	FileCreationDate     string
	FileCreationTime     string
	FileIDModifier       string
}

type Batch struct {
	Header  BatchHeaderRecord
	Control *BatchControl
	Entries []EntryRecord
}

type BatchHeaderRecord interface {
	Record
	SECCode() StandardEntryClassCode
	IsInternational() bool
	ServiceClassCode() ServiceClassCode
}

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

type FileControl struct {
	recordBase
	BatchCount        int64
	BlockCount        int64
	EntryAddendaCount int64
	EntryHash         string
	TotalDebitAmount  int64
	TotalCreditAmount int64
}

type PaddingRecord struct {
	recordBase
}

type EntryRecord interface {
	Record
	TransactionCode() TransactionCode
	ReceivingDFIPrefix() string
	AmountCents() int64
	AddendaRecords() []AddendaRecord
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

type EntryDetail struct {
	entryBase
	AccountNumber string
	TraceNumber   string
}

type CorporateTradeExchangeEntryDetail struct {
	entryBase
	AccountNumber        string
	NumberOfAddenda      int64
	ReceivingCompanyName string
	TraceNumber          string
}

type InternationalEntryDetail struct {
	entryBase
	AddendaRecordCount int64
	AccountNumber      string
	TraceNumber        string
}

type ReturnIndividualEntry struct {
	entryBase
	AccountNumber string
	TraceNumber   string
}

type ReturnCorporateEntry struct {
	entryBase
	AccountNumber   string
	NumberOfAddenda int64
	TraceNumber     string
}

type ReturnInternationalEntry struct {
	entryBase
	AddendaRecordCount int64
	AccountNumber      string
	TraceNumber        string
}

type AddendaRecord interface {
	Record
	AddendaTypeCode() AddendaTypeCode
}

type addendaBase struct {
	recordBase
	AddendaType AddendaTypeCode
}

func (a *addendaBase) AddendaTypeCode() AddendaTypeCode { return a.AddendaType }

type Addenda05 struct {
	addendaBase
	PaymentRelatedInformation string
}

type PointOfSaleAddenda02 struct{ addendaBase }
type NotificationOfChangeAddenda98 struct{ addendaBase }
type InternationalAddenda10 struct{ addendaBase }
type InternationalAddenda11 struct{ addendaBase }
type InternationalAddenda12 struct{ addendaBase }
type InternationalAddenda13 struct{ addendaBase }
type InternationalAddenda14 struct{ addendaBase }
type InternationalAddenda15 struct{ addendaBase }
type InternationalAddenda16 struct{ addendaBase }
type InternationalAddenda17 struct{ addendaBase }
type InternationalAddenda18 struct{ addendaBase }
type ReturnAddenda99 struct{ addendaBase }
type DishonoredReturnAddenda99 struct{ addendaBase }
type InternationalReturnAddenda99 struct{ addendaBase }
