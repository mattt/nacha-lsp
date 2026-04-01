# nacha-lsp

`nacha-lsp` is a Language Server Protocol implementation for NACHA ACH text files.
It includes a self-contained NACHA parser and serializer used by diagnostics and hover.

## MVP features

- Diagnostics on save for core NACHA structure checks:
  - each record is 94 characters,
  - valid record type prefixes (`1`, `5`, `6`, `7`, `8`, `9`),
  - record ordering and batch envelope checks,
  - control-level count/hash/total consistency checks,
  - blocking factor warning when line count is not a multiple of 10.
- Hover documentation for key field ranges in `1`, `5`, `6`, `7`, `8`, and `9` records.

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
- `nacha.HoverAt(text, line, character)` resolves field metadata from shared schema definitions.

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

## Round-trip guarantee

For well-formed parsed records, serialization preserves 94-character fixed-width records and
record ordering so `parse -> serialize -> parse` remains stable in tests.

## Example NACHA sample

Use fixed-width (94-char) records. This is a placeholder-form sample pattern:

- `1` + 93 chars
- `5` + 93 chars
- `6` + 93 chars
- `8` + 93 chars
- `9` + 93 chars
- five additional `9` padding lines to reach 10 total records
