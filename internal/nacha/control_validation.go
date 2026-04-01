package nacha

func validateControlTotals(file *File, appendDiag func(line, start, end int, message string, severity Severity)) {
	totalEntryAddenda := int64(0)
	totalDebit := int64(0)
	totalCredit := int64(0)
	totalHash := int64(0)

	for _, batch := range file.Batches {
		if batch == nil {
			continue
		}
		batchEntryAddenda, batchDebit, batchCredit, batchHash := summarizeBatch(batch)
		totalEntryAddenda += batchEntryAddenda
		totalDebit += batchDebit
		totalCredit += batchCredit
		totalHash += batchHash

		if batch.Control == nil {
			if batch.Header != nil {
				appendDiag(batch.Header.Line(), 0, 1, "batch is missing batch control record (8)", SeverityError)
			}
			continue
		}

		if batch.Control.EntryAddendaCount != batchEntryAddenda {
			appendDiag(batch.Control.Line(), 4, 10, "batch control entry/addenda count does not match records in batch", SeverityError)
		}
		if batch.Control.TotalDebitAmount != batchDebit {
			appendDiag(batch.Control.Line(), 20, 32, "batch control debit total does not match calculated debit total", SeverityError)
		}
		if batch.Control.TotalCreditAmount != batchCredit {
			appendDiag(batch.Control.Line(), 32, 44, "batch control credit total does not match calculated credit total", SeverityError)
		}
		expectedBatchHash := formatUnsigned(batchHash, 10)
		if batch.Control.EntryHash != expectedBatchHash {
			appendDiag(batch.Control.Line(), 10, 20, "batch control entry hash does not match calculated hash", SeverityError)
		}
	}

	if file.Control.BatchCount != int64(len(file.Batches)) {
		appendDiag(file.Control.Line(), 1, 7, "file control batch count does not match number of batches", SeverityError)
	}
	if file.Control.EntryAddendaCount != totalEntryAddenda {
		appendDiag(file.Control.Line(), 13, 21, "file control entry/addenda count does not match calculated count", SeverityError)
	}
	if file.Control.TotalDebitAmount != totalDebit {
		appendDiag(file.Control.Line(), 31, 43, "file control debit total does not match calculated debit total", SeverityError)
	}
	if file.Control.TotalCreditAmount != totalCredit {
		appendDiag(file.Control.Line(), 43, 55, "file control credit total does not match calculated credit total", SeverityError)
	}
	expectedHash := formatUnsigned(totalHash, 10)
	if file.Control.EntryHash != expectedHash {
		appendDiag(file.Control.Line(), 21, 31, "file control entry hash does not match calculated hash", SeverityError)
	}

	recordsCount := int64(len(file.Records))
	expectedBlocks := (recordsCount + (blockSize - 1)) / blockSize
	if file.Control.BlockCount != expectedBlocks {
		appendDiag(file.Control.Line(), 7, 13, "file control block count does not match record count", SeverityError)
	}
}

func summarizeBatch(batch *Batch) (entryAddenda int64, debit int64, credit int64, hash int64) {
	for _, entry := range batch.Entries {
		entryAddenda++
		if isCreditTransactionCode(entry.TransactionCode()) {
			credit += entry.AmountCents()
		} else {
			debit += entry.AmountCents()
		}
		entryAddenda += int64(len(entry.AddendaRecords()))
		if prefix, ok := parseInt64Strict(entry.ReceivingDFIPrefix()); ok {
			hash += prefix
		}
	}
	return entryAddenda, debit, credit, hash
}

func isCreditTransactionCode(code TransactionCode) bool {
	switch string(code) {
	case "20", "21", "22", "23", "24",
		"30", "31", "32", "33", "34",
		"41", "42", "43", "44",
		"51", "52", "53", "54",
		"81", "83", "85", "87":
		return true
	default:
		return false
	}
}
