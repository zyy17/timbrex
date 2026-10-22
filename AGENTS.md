# AGENTS.md

Rules for humans and coding agents working on Timbrex. Prefer this file over history when they disagree. Design and usage live in [readme.md](readme.md), [docs/tutorial.md](docs/tutorial.md), and [docs/spec.md](docs/spec.md) (SPEC v0.3.4). Do not paste chat logs into the docs. This file is English-only for tool discovery.

## What this project is

Timbrex is an English-ASCII tag language for expressive speech, plus a compiler that turns a `.tbx` script into audio through more than one provider. The grammar is frozen. Growth belongs in the tag vocabulary, capability declarations, and provider packages.

Three goals:

1. **Small compiler.** Pipeline is lex, then parse, then check, then lower to IR, then provider. If a maintained library already covers a piece (YAML front matter, the CLI, HTTP, audio muxing), use it. Do not build a compiler framework.
2. **Audio from scripts.** The end state is `timbrex synth script.tbx` and `timbrex check`. Support as many TTS providers as practical. When adding a provider, read that vendor's official docs and implement the real API, tags, and parameters. No guessed request shapes.
3. **Readable code.** Short packages, short functions, little indirection. A reader should follow one script from source to a provider call without a pile of wrappers.

## Layering

| Package | May know | Must not know |
| --- | --- | --- |
| `internal/lex` | Source text, tokens | AST, files, providers |
| `internal/parse` | Tokens, AST, YAML front matter | IR, providers, the CLI |
| `internal/ast` | The script tree | I/O, providers |
| `internal/ir` | Utterances, tag names as strings | Source text, the CLI |
| `internal/lower` | AST, IR, the standard tag kinds | I/O, providers, the CLI |
| `internal/vocab` | The standard tag list | Files, the CLI |
| `internal/check` | AST, vocabulary, diagnostics | Lexing, providers, the CLI |
| `internal/providers` | IR, the provider interface | Lexing, Cobra |
| `internal/providers/<name>` | That vendor's API and one `Provider` | The lexer, the parser, Cobra |
| `internal/config` | koanf, YAML configuration, environment variables, typed settings | AST, IR, provider execution, Cobra |
| `internal/cli` | Cobra, configuration, wiring parse → lower → provider | Tag grammar details |

Only `internal/cli` may import Cobra. A provider registers itself from `init`. The CLI blank-imports the providers it ships.

Later packages, not in this tree yet: `internal/expand` (dataset templates), `internal/caps` (provider capability files). Do not add them as empty shells.

Load configuration once through the Cobra entry point. Keep environment bindings and defaults in `internal/config`; adapters receive resolved options. Configuration priority is flags, script front matter, environment, the selected YAML file, then defaults. See [configuration](docs/configuration.md).

## Code style

- Idiomatic Go: community naming, short functions, flat structs. Avoid deep nesting.
- Teachability first. Prefer the smallest clear code. Do not pad scope.
- Comments only on key paths (what the grammar freezes, what a provider must take from vendor docs). No narrating comments.
- Pass `make test` and `make lint` before you claim done. `make test` builds `./timbrex` and runs `go test ./...`. `make lint` runs golangci-lint v2.14.0 and `golangci-lint fmt --diff`. CI uses the Go version in `go.mod` and that same linter.
- Prefer a pattern that is already in the tree over a new abstraction.

## CLI

- Binary name: `timbrex`. Script extension: `.tbx`. `.timbrex` is an alias for the same language. The CLI reads the path it is given. Entry point is repository-root `main.go`: `go run .` or `make build` then `./timbrex`. There is no `cmd/timbrex` package.
- With `cast: auto`, `check` lists each recognized speaker once.
- Front matter `narrator:` names the narration speaker. The default is `narrator`.
- Commands: `version`, `check`, `synth`, `config`. There is no `compile` or `inspect` command.
- `check` parses, resolves tags, and prints `ok` or diagnostics. `--strict-tags` / `--no-strict-tags` override front matter `strict_tags`.
- `synth` writes audio after the same check. `--provider` wins, then front matter `provider:`, then configuration (default `stub`).
- `--config` selects a YAML file; otherwise load `./timbrex.yaml` if present. `config` displays merged settings with credentials redacted. Help and version do not load configuration.
- `stub` is the fake provider. It writes a WAV tone so the pipeline can be tested. It is not a speech engine.
- Do not add a real provider until its official documentation has been read and the request matches that documentation.

## Current scope

In:

- Plain text, `[tag]`, `[tag args|content]`, nested spans, speaker lines, `//` comments, YAML front matter, scene lines, raw fences
- Cast: an `ident:` line is a turn only when that id is in `cast`, or when `cast: auto`. Otherwise it is narration
- Value grammar: number, duration, percent, semitone, decibel, quoted string, word, and `key=value`
- Standard vocabulary, V041 in strict mode, W040 for a near miss in a single script, W103 when a display name is used as a speaker prefix, I050 for non-ASCII acting text, V013 for `[/name]`, V012 for an unclosed tag
- Explicit `[act "…"]` in strict mode. Implicit free-form tags become `act` when strict mode is off
- Lowering to IR with typed arguments. Stub synthesis and real HTTP providers: ElevenLabs, OpenAI, MiniMax, Azure, Qwen, Gemini, and Fish Audio
- Cast voice mappings, scene/turn/span planning, local pauses, sequential PCM synthesis into WAV
- Centralized koanf configuration, environment bindings, and Cobra command execution
- TextMate grammar in `editors/vscode` for `.tbx`

Out:

- Vendor SDK dependencies, `timbrex fix` / `fmt`, dataset expand, `timbrex.lock`
- Per-tag parameter schemas, registered provider namespaces, tree-sitter
- General audio mixing, overlap, raw-provider passthrough, and streaming

Provider request contracts are tested offline. Do not claim live compatibility or acoustic quality without recording a real account validation. Unsupported controls must fail before synthesis requests rather than silently disappear. Provider details and setup live in [docs/providers.md](docs/providers.md); update its Chinese companion with it.

## Language

- Docs may be bilingual. English is the primary text (`readme.md`, `docs/spec.md`). A Chinese companion uses the `.zh-cn.md` suffix and must say the same thing.
- Use lowercase documentation filenames. Keep `AGENTS.md` as named for tool discovery.
- Identifiers, package names, comments, CLI help, and CLI errors are English.
- Commit messages are English Conventional Commits.

## Docs

- Design and usage. No conversation transcripts.
- Keep README honest when behavior changes. Update the Chinese companion in the same change when the English text changes.
- [docs/spec.md](docs/spec.md) is SPEC v0.3.4, the language target. The code may lag it; say so in the README rather than pretending a command exists. Keep [docs/tutorial.md](docs/tutorial.md) runnable: every command and `ok`/error line must match the current CLI.

## Git

- Atomic Conventional Commits (`feat`, `fix`, `docs`, `refactor`, `test`, `chore`). Clean messages. No emoji.
- Self-test before push: `make test` and `make lint`.
- Do not commit secrets, `.env`, or API keys.

## When unsure

Prefer the smallest change that keeps the pipeline readable and the grammar frozen. If a request wants a new syntax, say so and point at a new tag, a provider package, or a capability file instead.
