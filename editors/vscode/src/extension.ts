import * as path from "node:path";
import type { ExtensionContext } from "vscode";
import {
  LanguageClient,
  type LanguageClientOptions,
  type ServerOptions,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

export function activate(context: ExtensionContext): void {
  const binaryPath = context.asAbsolutePath(path.join("bin", "nacha-lsp"));

  const serverOptions: ServerOptions = {
    command: binaryPath,
    args: [],
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "nacha" }],
  };

  client = new LanguageClient(
    "nachaLSP",
    "NACHA Language Server",
    serverOptions,
    clientOptions
  );

  void client.start();
}

export async function deactivate(): Promise<void> {
  if (!client) {
    return;
  }

  await client.stop();
  client = undefined;
}
