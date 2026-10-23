# Timbrex for VS Code and Cursor

[简体中文](readme.zh-cn.md)

This extension highlights `.tbx` and `.timbrex` scripts in VS Code and Cursor. The language rules are in [DSL Specification v1](../../docs/spec.md). The extension provides syntax highlighting; use the Timbrex CLI for validation. Completion and an LSP are not included.

## Load the extension

With your editor's CLI available on `PATH`, run one of these commands from the repository root. It opens a development window that loads the extension directly from this checkout:

```bash
code --extensionDevelopmentPath="$PWD/editors/vscode" "$PWD/examples/expressive.tbx"
# Or use Cursor's CLI:
cursor --extensionDevelopmentPath="$PWD/editors/vscode" "$PWD/examples/expressive.tbx"
```

VS Code documents this source-loading option in its [extension testing guide](https://code.visualstudio.com/api/working-with-extensions/testing-extension). No extension build step is needed: the manifest loads the checked-in JSON grammar.

For a persistent local source installation, copy the whole `editors/vscode` directory into your editor's extensions directory under a name such as `timbrex-local`, then reload the window. Default local locations are `~/.vscode/extensions` for VS Code and `~/.cursor/extensions` for Cursor; remote or custom setups can use other locations. To update this copy, replace it with the revised directory and reload again. The repository does not include a VSIX. [`code --install-extension`](https://code.visualstudio.com/docs/configure/command-line#working-with-extensions) accepts an extension ID or a VSIX path.

## Edit and validate a script

Open a `.tbx` or `.timbrex` file and check that the status bar shows **Timbrex** as its language. If it does not, use **Change Language Mode** to select Timbrex. From the repository root, validate a script with:

```bash
go run . check examples/expressive.tbx
```

Highlighting covers front matter, line comments, raw fences, tags and nested spans, quoted header arguments, numeric suffixes, and escapes. Quotes in span content remain ordinary spoken text. Every column-zero ASCII `ident:` is colored as a speaker; the highlighter cannot read `cast`, so the compiler determines whether that prefix actually starts a turn. A highlighted tag may still be unknown or unsupported by a provider.

## Maintain the grammar

Authoring source: [syntaxes/tbx.tmLanguage.yaml](syntaxes/tbx.tmLanguage.yaml). Packaged runtime grammar: [syntaxes/tbx.tmLanguage.json](syntaxes/tbx.tmLanguage.json). VS Code requires the JSON form, as explained in its [syntax highlighting guide](https://code.visualstudio.com/api/language-extensions/syntax-highlight-guide#using-yaml-to-write-a-grammar).

After editing YAML, regenerate from the repository root:

```bash
go test ./editors/vscode -update
```

The runtime regression tests use the actual TextMate tokenizer and Oniguruma engine. With Node.js 24 installed, run from this extension directory:

```bash
npm ci --ignore-scripts
npm test
```

CI runs these tests separately from the Go tests. The packages are development-only; users copying the extension do not need Node.js or `node_modules`.

Normal tests fail if the JSON differs from the YAML or the extension manifest points elsewhere. The language ID is `timbrex`, and the TextMate scope is `source.tbx`. TextMate-compatible editors may reuse the JSON. A tree-sitter grammar for Neovim or Helix is not provided.

## License

Copyright (c) 2026 zyy17

This extension is licensed under the [Apache License 2.0](license). Keep this copyright notice and the license file when copying the extension directory on its own.
