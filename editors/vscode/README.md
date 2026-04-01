# NACHA LSP — VS Code Extension

VS Code client for [nacha-lsp](../../README.md).
Provides diagnostics, hover, completion, formatting,
code actions, and syntax highlighting for `.ach` files.

## Getting started

Build the server and copy it into the extension:

```bash
make build-dev
```

Install dependencies and launch:

```bash
cd editors/vscode
npm install
```

Press **F5** to open an Extension Development Host,
then open any `.ach` file.
