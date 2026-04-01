package nacha

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
