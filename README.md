# Timbrex

Timbrex is a tag-based scripting language for expressive text-to-speech. A `.tbx` file is plain text plus one markup form, `[name]` or `[name|content]`. The same script compiles to more than one speech provider.

Grammar version: `timbrex: 1`. Design: [docs/SPEC.md](docs/SPEC.md).

**Status: M0.** The compiler parses the v0.3.4 surface: plain text, `[tag]` and `[tag|content]`, arguments, speaker lines, cast, `//` comments, YAML front matter, scene lines, and raw fences. `timbrex check` applies the standard vocabulary. `--strict-tags` rejects an unregistered tag (the batch rule) and still accepts `[act "…"]`. `timbrex synth` lowers the script and calls a provider. The only provider in this tree is `stub`, which writes a short tone as a WAV file. It does not speak.

## Goals

1. **Small compiler.** The pipeline is lex, parse, IR, then lower. Use a maintained library where one already fits (command line, YAML front matter, and later HTTP or audio muxing). Do not grow a compiler framework.
2. **Audio from scripts.** `timbrex check` parses. `timbrex synth` writes an audio file. Providers are separate packages behind one interface, so more vendors can land without changing the grammar.
3. **Readable Go.** Short packages, short functions, little indirection.

## Quick example

`examples/hello.tbx`:

```tbx
---
timbrex: 1
cast:
  lin: { name: Lin }
  chen: { name: Chen }
---
// A two-line scene.
lin: [laugh] Really? [pause 500ms] That's great!
chen: [sigh] [whisper|I knew it already.]
```

```sh
make build
./timbrex check examples/hello.tbx
./timbrex synth examples/hello.tbx -o hello.wav
```

`hello.wav` from this tree is a tone from the stub provider, not speech. Spoken lines and quoted strings may be any language. Tag names, arguments, and config keys are English ASCII.

## Build

Requires Go 1.27.

```sh
make build   # writes ./timbrex
make test    # builds ./timbrex, then runs the tests
make lint    # golangci-lint v2.14.0, same as CI
make fmt
make tidy    # fails if go.mod or go.sum would change
```

`timbrex version` prints the tool version. `timbrex synth --provider` selects a registered provider. Front matter `provider:` is the fallback when the flag is omitted. The default is `stub`. With `cast: auto`, `timbrex check` lists each recognized speaker once. Front matter `narrator:` sets the speaker of unprefixed paragraphs. The default is `narrator`. A cast display name used as a prefix stays narration and warns W103.

`timbrex check --strict-tags` is the batch rule: an unregistered `[tag]` is error V041, and `[act "…"]` is the explicit free-form form. `--no-strict-tags` forces the single-script rule. With neither flag, front matter `strict_tags` applies, and a normal script defaults to allowing free-form tags. A single unknown word within edit distance 2 of a standard tag is warning W040, and the command still succeeds. Non-ASCII free-form acting is info I050. `[laff]` is distance 3 from `[laugh]`, so it is reported with no suggestion. `[paus]` suggests `[pause]`.

## Editor highlighting

A TextMate grammar covers the same tokens. VS Code and Cursor load it as a language extension. Sublime Text and other TextMate editors can point at the same file. Neovim and Helix need a tree-sitter grammar, which is not in M0.

```sh
cd editors/vscode
cursor --install-extension .    # or: code --install-extension .
```

Details: [editors/vscode/README.md](editors/vscode/README.md). Grammar file: [editors/vscode/syntaxes/tbx.tmLanguage.yaml](editors/vscode/syntaxes/tbx.tmLanguage.yaml). After installing, reload the window and open a `.tbx` file. The language id is `timbrex`. `.timbrex` is recognized as the same language.

## Layout

| Path | Role |
| --- | --- |
| `main.go` | Process entry |
| `internal/cli` | Cobra commands |
| `internal/lex` | Tokens |
| `internal/parse` | AST, including YAML front matter |
| `internal/ast` | Script tree |
| `internal/lower` | AST to IR |
| `internal/ir` | Flat utterances |
| `internal/vocab` | Standard tag list |
| `internal/check` | Strict tags, closing tags, notes |
| `internal/providers` | Provider interface |
| `internal/providers/stub` | Tone WAV used by tests and the default synth path |
| `editors/vscode` | TextMate grammar and editor extension |

## Out of scope for M0

- Real provider APIs and SDKs
- `timbrex fix` / `fmt`, dataset expand (`*.tbxset.yaml`), and `timbrex.yaml` / `timbrex.lock` loading
- Per-tag parameter schemas beyond the value grammar and `note: allow`
- A tree-sitter grammar, audio muxing, streaming, and voice cloning

## Docs

English is the primary text. Chinese companions use the `.zh-CN.md` suffix. [中文说明](README.zh-CN.md).

- [docs/SPEC.md](docs/SPEC.md) — language v0.3.4 ([中文](docs/SPEC.zh-CN.md))
- [docs/CAPABILITIES.md](docs/CAPABILITIES.md) — provider control surface ([中文](docs/CAPABILITIES.zh-CN.md))
- [docs/MARKET_SURVEY.md](docs/MARKET_SURVEY.md) — market and voice features ([中文](docs/MARKET_SURVEY.zh-CN.md))
- [docs/REFLECTION.md](docs/REFLECTION.md) — why the grammar looks like this ([中文](docs/REFLECTION.zh-CN.md))

## License

[MIT](LICENSE)
