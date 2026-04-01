# nacha-lsp

`nacha-lsp` is a Language Server Protocol implementation for NACHA ACH text files.
It includes a self-contained NACHA parser and serializer used by diagnostics and hover.

## Features

- Diagnostics on save for core NACHA structure checks:
  - each record is 94 characters,
  - valid record type prefixes (`1`, `5`, `6`, `7`, `8`, `9`),
  - record ordering and batch envelope checks,
  - control-level count/hash/total consistency checks,
  - blocking factor warning when line count is not a multiple of 10.
- Hover documentation for field ranges in `1`, `5`, `6`, `7`, `8`, and `9` records.
- Document symbols for file/batch/entry/addenda outline.
- Completion suggestions for key NACHA code fields:
  - batch `Service Class Code` (`200`, `220`, `225`),
  - batch `Standard Entry Class Code` (`PPD`, `CCD`, `CTX`, `IAT`),
  - entry `Transaction Code` (`22`, `27`, `32`, `37`),
  - addenda `Addenda Type Code` (`02`, `05`, `98`, `99`).
- Document formatting via parse/serialize canonicalization (non-destructive; returns no edits when parse has errors).
- Quick-fix code actions for:
  - normalizing line length to 94,
  - appending `9` padding records to satisfy the block factor,
  - inserting a trailing newline at EOF.

## Parser coverage

The internal parser supports typed record variants across the NACHA families used in the reference:

- Origination-oriented records:
  - File header/control, domestic batch header/control, domestic entry detail, addenda `05`, POS addenda `02`, NOC addenda `98`.
  - International batch/header and entry context with IAT addenda `10` through `18`.
- Return-oriented variants:
  - Return-style entry/addenda discrimination for addenda `99` (including dishonored heuristic variant).
- Padding:
  - File padding `9` records are tracked after the file control record.

Public internal API entry points:

- `nacha.Parse(text)` returns a typed file model and parser diagnostics.
- `(*nacha.File).Serialize()` round-trips parsed records back to NACHA text.
- `nacha.Validate(text)` returns parser-backed validation diagnostics.
- `nacha.LookupPosition(record, column)` resolves schema-backed position metadata for a single NACHA record.

Reference: [ACH File Overview](https://achdevguide.nacha.org/ach-file-overview).

## Build and run

```bash
go build -o bin/nacha-lsp ./cmd/nacha-lsp
```

The server uses stdio transport.

## VS Code wiring (minimal)

You can wire this binary from a VS Code extension (or local test extension host) with a language client config similar to:

```json
{
  "contributes": {
    "languages": [
      {
        "id": "nacha",
        "extensions": [".ach", ".nacha"],
        "aliases": ["NACHA"]
      }
    ]
  },
  "activationEvents": ["onLanguage:nacha"],
  "serverOptions": {
    "command": "/absolute/path/to/bin/nacha-lsp"
  }
}
```

## Manual smoke test

1. Start VS Code with NACHA file association enabled (`.ach` or `.nacha`).
2. Open a valid NACHA file and save: no diagnostics should appear.
3. Break a line to fewer than 94 characters and save: diagnostics should appear.
4. Hover over a `6` record at columns 2-3: transaction-code hover details should appear.
5. Request document symbols: file/batch/entry structure should appear.
6. Trigger completion at a known code field (for example batch columns 2-4): code suggestions should appear.
7. Run format document on a valid file with CRLF line endings: output should be canonical LF NACHA records.
8. Request code actions on a 94-character diagnostic: quick fixes should include record-length normalization.

## Round-trip guarantee

For well-formed parsed records, serialization preserves 94-character fixed-width records and
record ordering so `parse -> serialize -> parse` remains stable in tests.

## Example NACHA sample

```
101 03130001212345678902604011200A094101DEST BANK              ORIGIN CO                      
5200ACME COMPANY                         1234567890PPDPAYROLL         260401   1123456780000001
622031300012987654321        0000001000EMP001         JOHN DOE                0123456780000001
820000000100031300010000000000000000000010001234567890                         123456780000001
90000010000010000000100031300010000000000000000001000                                       
9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999
9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999
9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999
9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999
9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999
```
