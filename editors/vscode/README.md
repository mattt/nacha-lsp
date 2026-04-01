# NACHA VS Code Extension

This extension starts the local `nacha-lsp` server binary over stdio
for smoke testing diagnostics and hovers.
It also contributes NACHA syntax highlighting.

## Quick start

1. From the repo root, build and copy the server:
   `make build-dev`
2. Install extension dependencies:
   `cd editors/vscode && npm install`
3. Press **F5** in the `editors/vscode` workspace
   to launch the Extension Development Host.
4. Open a file ending in `.ach` and type to see live diagnostics.
