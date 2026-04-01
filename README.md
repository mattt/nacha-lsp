# nacha-lsp

A Language Server Protocol (LSP) implementation for NACHA ACH files,
with a self-contained parser and serializer.

## Features

- Diagnostics while editing (open/change/save) for core NACHA structure checks:
  - each record is 94 characters
  - valid record type prefixes (`1`, `5`, `6`, `7`, `8`, `9`)
  - record ordering and batch envelope checks
  - control-level count/hash/total consistency
  - blocking factor warning when line count is not a multiple of 10
- Hover documentation for field ranges
  in `1`, `5`, `6`, `7`, `8`, and `9` records.
- Document symbols for file/batch/entry/addenda outline.
- Completion suggestions for key NACHA code fields:
  - batch Service Class Code (`200`, `220`, `225`)
  - batch Standard Entry Class Code (`PPD`, `CCD`, `CTX`, `IAT`)
  - entry Transaction Code (`22`, `27`, `32`, `37`)
  - addenda Addenda Type Code (`02`, `05`, `98`, `99`)
- Document formatting via canonical serialization
  (no edits when the file has parse errors).
- Quick-fix code actions for:
  - normalizing line length to 94
  - appending `9` padding records to satisfy the block factor
  - inserting a trailing newline at EOF

## Parser coverage

The internal parser supports typed record variants
across the NACHA families described in the reference:

- Origination-oriented records:
  - File header/control, domestic batch header/control,
    domestic entry detail, addenda `05`,
    POS addenda `02`, NOC addenda `98`.
  - International batch/header and entry context
    with IAT addenda `10` through `18`.
- Return-oriented variants:
  - Return-style entry/addenda discrimination for addenda `99`
    (including dishonored heuristic variant).
- Padding:
  - File padding `9` records tracked after the file control record.

API entry points (internal package):

- `nacha.Parse(text)` — returns a typed file model and parser diagnostics.
- `(*nacha.File).Serialize()` — round-trips parsed records back to NACHA text.
- `nacha.Validate(text)` — returns parser-backed validation diagnostics.
- `nacha.LookupPosition(record, column)` — resolves schema-backed position metadata
  for a single NACHA record.

Reference: [ACH File Overview](https://achdevguide.nacha.org/ach-file-overview).

## Build

```bash
go build -o bin/nacha-lsp ./cmd/nacha-lsp
```

The server uses stdio transport.

## VS Code extension

A minimal VS Code client wrapper is included at `editors/vscode`.
It also contributes NACHA syntax highlighting via a TextMate grammar.

Build and copy the latest server binary into the extension:

```bash
make build-dev
```

Install extension dependencies:

```bash
cd editors/vscode
npm install
```

Then press **F5** in the `editors/vscode` workspace
to launch an Extension Development Host.
The wrapper starts `editors/vscode/bin/nacha-lsp` over stdio.

If **F5** is not available,
use **Run and Debug > Run NACHA LSP Extension**
or **Command Palette > Debug: Start Debugging**.

You can also launch from the terminal:

```bash
cd editors/vscode
npm run compile
code --extensionDevelopmentPath="$(pwd)"
```

## Manual smoke test

1. Build and copy the server with `make build-dev`.
2. Launch the Extension Development Host from `editors/vscode` (**F5**).
3. In the host, create and save a `.ach` file.
4. Open a valid NACHA file and save — no diagnostics should appear.
5. Break a line to fewer than 94 characters and save — diagnostics should appear.
6. Hover over a `6` record at columns 2–3 — transaction-code hover details should appear.
7. Request document symbols — file/batch/entry structure should appear.
8. Trigger completion at a known code field
   (e.g. batch columns 2–4) — code suggestions should appear.
9. Run format document on a valid file with CRLF line endings —
   output should be canonical LF NACHA records.
10. Request code actions on a 94-character diagnostic —
    quick fixes should include record-length normalization.

## Round-trip guarantee

For well-formed parsed records,
serialization preserves 94-character fixed-width records and record ordering,
so `parse → serialize → parse` remains stable in tests.

## Example

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
