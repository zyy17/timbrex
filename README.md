# Timbrex

[简体中文](README.zh-CN.md)

**A small DSL for expressive text-to-speech.** Write what to say and how to say it, then generate audio through ElevenLabs, OpenAI, MiniMax, Azure Speech, or Qwen.

```tbx
[happy|Good to see you!] [pause 300ms] [slow|Take your time.]
```

Plain text is valid. Tags use one shape: `[name]` or `[name|content]`. Spoken text can be in any language; tag names are English ASCII. The grammar is frozen at `timbrex: 1`. Providers and vocabulary can grow without adding syntax.

Timbrex is in early development. The compiler and five HTTP adapters are implemented and covered by offline contract tests. These adapters have **not yet been validated with live provider accounts**. Model access, voices, regional availability, and the quality of acting instructions require live verification. Different providers do not promise identical performances.

## Generate your first speech

Requires Go 1.27, as declared in [go.mod](go.mod).

```sh
make build
./timbrex check examples/expressive.tbx --strict-tags

# Set OPENAI_API_KEY in your environment first.
./timbrex synth examples/expressive.tbx --provider openai -o speech.wav
```

The example above works with each adapter's default model and a suitable voice. ElevenLabs additionally requires your voice ID; Qwen requires a workspace endpoint. See [provider setup](docs/PROVIDERS.md) for all five services.

To try the compiler without credentials or a network request:

```sh
./timbrex synth examples/expressive.tbx --provider stub -o tone.wav
```

`stub` creates a test tone, not speech. It remains the default when no provider is selected.

## Learn the language

```tbx
---
timbrex: 1
provider: openai
strict_tags: true
cast:
  lin:
    name: Lin
    voices: {openai: coral}
  chen:
    name: Chen
    voices: {openai: onyx}
---
lin: [happy] You came! [pause 300ms] [whisper|Come inside.]
chen: [slow|I've got something to tell you.]
```

- `[happy]` changes delivery for the rest of the turn; `[happy|words]` changes only those words.
- A line containing only tags without spans sets scene state. `[reset]` on its own line clears it.
- `lin:` starts a turn only if `lin` is declared in `cast`. Without a cast, the prefix is ordinary spoken text. `cast: auto` opts into automatic speaker detection.
- Unprefixed paragraphs use `narrator`, or the ID selected by front matter `narrator:`.
- `//` at the start of a line is a comment. Blank lines separate turns.
- Prefer `[act "holding back tears"]` for free-form acting. `--strict-tags` rejects unknown names; `--no-strict-tags` allows implicit acting such as `[thoughtfully]`.

Read the [language reference](docs/SPEC.md) and [implemented capability table](docs/CAPABILITIES.md). A recognized tag is not necessarily synthesizable by every provider. Unsupported controls fail instead of silently disappearing.

## Commands and output

```sh
./timbrex version
./timbrex check script.tbx
./timbrex synth script.tbx --provider openai --model gpt-4o-mini-tts --voice coral -o script.wav
```

Provider selection: `--provider`, then front matter `provider:`, then `stub`. Model and default voice: CLI flag, front matter, provider environment variable, built-in default. A speaker's `cast.<id>.voices.<provider>` takes priority over the default voice. `--voice` supplies a fallback; it does not replace cast mappings.

`check` validates syntax and vocabulary without credentials or network access. `synth` runs the same check, validates the selected adapter's supported controls, and then makes requests. Strict-mode flags override front matter `strict_tags`; the default is non-strict. With `cast: auto`, `check` lists each recognized speaker once.

Real providers produce one 24 kHz, mono, 16-bit PCM WAV file. Turns and control changes become sequential requests; explicit pauses become local silence. Requests across span boundaries may sound discontinuous. Output is replaced only after the full render succeeds. Existing audio survives a failed render, although completed API calls may still be billed. There are no automatic retries. `-o` defaults to the source path with `.wav`; the source file cannot be overwritten.

## Development

```sh
make test    # builds ./timbrex and runs all Go tests
make lint    # requires golangci-lint v2.14.0, matching CI
make tidy    # verifies module files without changing them
make fmt
```

Tests cover parsing, diagnostics, nested scopes, request contracts for every adapter, exact pauses, failure handling, and output preservation. Tests use local HTTP servers, never vendor credentials. See [contributing](CONTRIBUTING.md) and [AGENTS.md](AGENTS.md).

The pipeline is source → lexer → parser/AST → check → lower/IR → provider → WAV. Packages follow those boundaries under `internal/`. Only `internal/cli` knows Cobra; each provider lives in `internal/providers/<name>`. [Editor support](editors/vscode/README.md) includes a TextMate grammar for VS Code and Cursor.

## Scope and documentation

Implemented: script parsing and checking, speaker voice mappings, scoped controls, sequential synthesis, local pauses, and WAV output. Not implemented: streaming, concurrent dialogue, audio mixing, pronunciation wrappers, raw-provider passthrough, voice cloning, dataset expansion, config/lock files, or `fmt`/`fix` CLI commands. Some corresponding tags are reserved in the vocabulary; synthesis rejects them.

English is the primary documentation. Chinese companions carry the same behavior and limitations.

| Document | English | 简体中文 |
| --- | --- | --- |
| Language reference | [SPEC](docs/SPEC.md) | [语言参考](docs/SPEC.zh-CN.md) |
| Credentials and adapters | [PROVIDERS](docs/PROVIDERS.md) | [厂商接入](docs/PROVIDERS.zh-CN.md) |
| Implemented controls | [CAPABILITIES](docs/CAPABILITIES.md) | [能力表](docs/CAPABILITIES.zh-CN.md) |
| Provider research | [MARKET_SURVEY](docs/MARKET_SURVEY.md) | [厂商调研](docs/MARKET_SURVEY.zh-CN.md) |
| Design rationale | [REFLECTION](docs/REFLECTION.md) | [设计思路](docs/REFLECTION.zh-CN.md) |
| Contributing | [CONTRIBUTING](CONTRIBUTING.md) | [参与贡献](CONTRIBUTING.zh-CN.md) |

MIT licensed. See [LICENSE](LICENSE).
