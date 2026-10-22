# Timbrex for VS Code and Cursor

[简体中文](readme.zh-cn.md)

Syntax highlighting for `.tbx` and `.timbrex`. The language ID is `timbrex`, and the scope is `source.tbx`. This extension provides highlighting, not completion, validation, or an LSP. Run `timbrex check` to validate scripts.

## Try the extension

From the repository root, launch an extension development window:

```sh
code --extensionDevelopmentPath="$PWD/editors/vscode" "$PWD/examples/expressive.tbx"
# Or use Cursor's CLI:
cursor --extensionDevelopmentPath="$PWD/editors/vscode" "$PWD/examples/expressive.tbx"
```

For a local installation, copy `editors/vscode` into your editor's extensions directory under a name such as `timbrex-local`, then reload the window. VS Code normally uses `~/.vscode/extensions`; Cursor normally uses `~/.cursor/extensions`. Custom profiles and remote environments may use other locations. There is no published Marketplace package in this repository. `code --install-extension` takes an extension ID or a VSIX, not this source directory.

## Grammar

Authoring source: [syntaxes/tbx.tmLanguage.yaml](syntaxes/tbx.tmLanguage.yaml). Packaged runtime grammar: [syntaxes/tbx.tmLanguage.json](syntaxes/tbx.tmLanguage.json). VS Code requires the JSON form, as explained in its [syntax highlighting guide](https://code.visualstudio.com/api/language-extensions/syntax-highlight-guide#using-yaml-to-write-a-grammar).

After editing YAML, regenerate from the repository root:

```sh
go test ./editors/vscode -update
```

Normal tests fail if the JSON differs from the YAML or the extension manifest points elsewhere. TextMate-compatible editors may reuse the JSON. A tree-sitter grammar for Neovim or Helix is not provided.

Highlighting covers front matter, line comments, raw fences, tags and nested spans, quoted header arguments, numeric suffixes, and escapes. Quotes in span content remain ordinary spoken text. Any column-zero ASCII `ident:` is colored as a speaker; the highlighter cannot read the cast, so only the compiler determines whether it is actually a turn. The grammar is a visual aid, not a replacement for the parser.
