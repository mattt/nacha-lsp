# nacha-lsp

A language server for [NACHA](https://achdevguide.nacha.org/ach-file-overview) ACH files.

## Features

- **Diagnostics** —
  record length, type prefixes, ordering, batch envelopes,
  control totals, and blocking factor.
- **Hover** —
  field-level documentation for all record types.
- **Document symbols** —
  file / batch / entry / addenda outline.
- **Completion** —
  suggestions for Service Class Code, SEC Code,
  Transaction Code, and Addenda Type Code fields.
- **Formatting** —
  canonical 94-character fixed-width serialization.
- **Code actions** —
  quick fixes for line length, block padding, and trailing newline.

## Build

```bash
go build -o bin/nacha-lsp ./cmd/nacha-lsp
```

The server communicates over stdio.

## Usage

A minimal VS Code extension is included under [`editors/vscode`](editors/vscode).

```bash
make build-dev          # build server and copy into the extension
cd editors/vscode
npm install             # install extension dependencies
```

Press **F5** to launch an Extension Development Host,
or run from the terminal:

```bash
code --extensionDevelopmentPath="$(pwd)"
```

## Example

```
101 03130001212345678902604011200A094101DEST BANK              ORIGIN CO                      
5200ACME COMPANY                        1234567890PPDPAYROLL         260401   1123456780000001
622031300012987654321        0000001000EMP001         JOHN DOE                0123456780000001
820000000100031300010000000000000000000010001234567890                         123456780000001
90000010000010000000100031300010000000000000000001000                                       
9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999
9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999
9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999
9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999
9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999
```
