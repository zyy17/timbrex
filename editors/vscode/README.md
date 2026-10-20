# Timbrex for VS Code and Cursor

Syntax highlighting for `.tbx` and `.timbrex` scripts.

The grammar is `syntaxes/tbx.tmLanguage.yaml`. It is a TextMate grammar, so Sublime Text and other TextMate editors can load that same file. Neovim and Helix do not consume TextMate grammars directly. A tree-sitter grammar is not part of M0; a later grammar can target the same tokens below.

Highlighted tokens, matching [docs/SPEC.md](../../docs/SPEC.md):

- YAML front matter between `---` at the start of the file
- `//` comments only at the start of a line
- raw fences whose info starts with an identifier (a line of three backticks, then an info string such as `ssml azure`)
- `[tag]`, `[tag args]`, `[tag|content]`, nested tags, quoted strings, `key=value`, and numeric suffixes (`ms`, `min`, `st`, `dB`, `%`, `s`)
- escapes `[[` `]]` `\[` `\]` `\|` `\\` `\:`

A column-0 `ident:` is colored as a speaker. The compiler still treats that prefix as narration unless the id is in `cast` or `cast` is `auto`. The highlighter does not read the cast list.

## Install from this repo

```sh
cd editors/vscode
# Cursor or VS Code:
cursor --install-extension .
# or
code --install-extension .
```

If the CLI has no install-from-directory command, use **Extensions: Install from Location** and choose `editors/vscode`, or symlink this directory into the editor's extensions folder.

Reload the window, then open a `.tbx` file. The language id is `timbrex`.
