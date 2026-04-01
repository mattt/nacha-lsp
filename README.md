# nacha-lsp

A language server for [NACHA](https://achdevguide.nacha.org/ach-file-overview) ACH files.

https://github.com/user-attachments/assets/9902920b-2d54-4e6c-96c5-9d3f0a89ca15

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
code --extensionDevelopmentPath="$(pwd)" ../../nacha/testdata/
```
