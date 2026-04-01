# NACHA VSCode Wrapper

This extension starts the local `nacha-lsp` server binary over stdio for smoke testing diagnostics and hovers.

## Quick start

1. From the repo root, build and copy the server:
   - `make build-dev`
2. Install extension dependencies:
   - `cd editors/vscode && npm install`
3. Press `F5` in the `editors/vscode` workspace to launch the Extension Development Host.
4. Open and save a file ending in `.ach`.
