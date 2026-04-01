package nacha

type parseContext struct {
	appendDiag   func(line, start, end int, message string, severity Severity)
	currentBatch *Batch
	lastEntry    EntryRecord

	seenFileHeader   bool
	seenFileControl  bool
	inBatch          bool
	entrySeenInBatch bool
	lastType         byte
}

func (c *parseContext) trackStructure(recordType byte, line int) {
	switch recordType {
	case '1':
		if line != 0 {
			c.appendDiag(line, 0, 1, "file header record (1) must be first", SeverityError)
		}
		if c.seenFileHeader {
			c.appendDiag(line, 0, 1, "file must contain only one file header record (1)", SeverityError)
		}
		if c.seenFileControl {
			c.appendDiag(line, 0, 1, "record type 1 is not allowed after file control", SeverityError)
		}
		c.seenFileHeader = true
	case '5':
		if !c.seenFileHeader {
			c.appendDiag(line, 0, 1, "batch header record (5) requires a preceding file header (1)", SeverityError)
		}
		if c.seenFileControl {
			c.appendDiag(line, 0, 1, "batch header record (5) is not allowed after file control", SeverityError)
		}
		if c.inBatch {
			c.appendDiag(line, 0, 1, "batch header record (5) cannot appear before batch control (8)", SeverityError)
		}
		c.inBatch = true
		c.entrySeenInBatch = false
	case '6':
		if !c.inBatch {
			c.appendDiag(line, 0, 1, "entry detail record (6) must be inside a batch", SeverityError)
		}
		if c.seenFileControl {
			c.appendDiag(line, 0, 1, "entry detail record (6) is not allowed after file control", SeverityError)
		}
		c.entrySeenInBatch = true
	case '7':
		if !c.inBatch {
			c.appendDiag(line, 0, 1, "addenda record (7) must be inside a batch", SeverityError)
		}
		if c.lastType != '6' && c.lastType != '7' {
			c.appendDiag(line, 0, 1, "addenda record (7) must follow an entry detail (6) or addenda (7)", SeverityError)
		}
		if c.seenFileControl {
			c.appendDiag(line, 0, 1, "addenda record (7) is not allowed after file control", SeverityError)
		}
	case '8':
		if !c.inBatch {
			c.appendDiag(line, 0, 1, "batch control record (8) must close an open batch", SeverityError)
		}
		if c.seenFileControl {
			c.appendDiag(line, 0, 1, "batch control record (8) is not allowed after file control", SeverityError)
		}
		if c.inBatch && !c.entrySeenInBatch {
			c.appendDiag(line, 0, 1, "batch must include at least one entry detail record (6) before control (8)", SeverityError)
		}
		c.inBatch = false
	case '9':
		if !c.seenFileControl {
			if c.inBatch {
				c.appendDiag(line, 0, 1, "file control record (9) cannot appear before batch control (8)", SeverityError)
			}
			if !c.seenFileHeader {
				c.appendDiag(line, 0, 1, "file control record (9) requires a file header (1)", SeverityError)
			}
			c.seenFileControl = true
		}
	}
	c.lastType = recordType
}

func (c *parseContext) finish(totalLines int) {
	lastLine := max(0, totalLines-1)
	if !c.seenFileHeader {
		c.appendDiag(0, 0, 1, "file must start with a file header record (1)", SeverityError)
	}
	if c.inBatch {
		c.appendDiag(lastLine, 0, 1, "file ended with an open batch; missing batch control record (8)", SeverityError)
	}
	if !c.seenFileControl {
		c.appendDiag(lastLine, 0, 1, "file must include a file control record (9)", SeverityError)
	}
}
