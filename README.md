# nacha-lsp

`nacha-lsp` is a minimal Language Server Protocol implementation for NACHA ACH text files.

## MVP features

- Diagnostics on save for core NACHA structure checks:
  - each record is 94 characters,
  - valid record type prefixes (`1`, `5`, `6`, `7`, `8`, `9`),
  - coarse record ordering and batch envelope checks,
  - blocking factor warning when line count is not a multiple of 10.
- Hover documentation for key field ranges in `1`, `5`, `6`, `8`, and `9` records.

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

## Example NACHA sample

Use fixed-width (94-char) records. This is a placeholder-form sample pattern:

- `1` + 93 chars
- `5` + 93 chars
- `6` + 93 chars
- `8` + 93 chars
- `9` + 93 chars
- five additional `9` padding lines to reach 10 total records
