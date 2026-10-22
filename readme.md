# Timbrex

[简体中文](readme.zh-cn.md)

Timbrex is an English-ASCII tag language for expressive speech. A `.tbx` script compiles to audio through a TTS provider.

```tbx
[happy|Good to see you!] [pause 300ms] [slow|Take your time.]
```

Plain text is a valid script. Tags use one form: `[name]`, `[name args]`, or `[name args|content]`. Spoken text may be any language; tag names stay English ASCII. The grammar is frozen at `timbrex: 1` ([SPEC v0.3.4](docs/spec.md)).

## Install

Requires Go 1.27, as declared in [go.mod](go.mod).

```sh
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex version
```

From the repository root, `go run .` is the same binary as `./timbrex`. There is no `cmd/timbrex` package.

## Quickstart

```sh
./timbrex check examples/expressive.tbx
./timbrex synth examples/expressive.tbx --provider stub -o tone.wav
# Set OPENAI_API_KEY first.
./timbrex synth examples/expressive.tbx --provider openai -o speech.wav
```

`check` prints `ok` or diagnostics. `stub` writes an 8 kHz test tone, not speech, and needs no key. HTTP adapters write one 24 kHz mono 16-bit PCM WAV. They have offline contract tests and have not been validated with live accounts.

## Providers

| ID | Required setup | Default model |
| --- | --- | --- |
| `stub` | None | Test tone (not speech) |
| `openai` | `OPENAI_API_KEY` | `gpt-4o-mini-tts` |
| `elevenlabs` | `ELEVENLABS_API_KEY` and a voice ID | `eleven_v3` |
| `minimax` | `MINIMAX_API_KEY` | `speech-2.8-hd` |
| `azure` | `AZURE_SPEECH_KEY` and region or endpoint | Selected by voice |
| `qwen` | `DASHSCOPE_API_KEY` and `QWEN_ENDPOINT` | `qwen-audio-3.1-tts-flash` |
| `gemini` | `GEMINI_API_KEY` | `gemini-3.8-flash-tts` |
| `fish` | `FISH_API_KEY` and a voice/reference ID | `s2.1-pro` |

Gemini rejects numeric speed; use [examples/gemini.tbx](examples/gemini.tbx). Fish requires a voice reference ID; use [examples/fish.tbx](examples/fish.tbx). A recognized tag is not necessarily synthesizable. Unsupported controls fail before requests rather than disappearing. Setup and mappings: [providers](docs/providers.md), [capabilities](docs/capabilities.md).

## Commands

`version`, `check`, `synth`, `config`. Priority: flags, script front matter, environment, `./timbrex.yaml` or `--config`, then defaults. The default provider is `stub`. See [configuration](docs/configuration.md).

## Documentation

Start with the [tutorial](docs/tutorial.md). Development: [contributing](contributing.md).

| Document | English | 简体中文 |
| --- | --- | --- |
| Tutorial | [Tutorial](docs/tutorial.md) | [教程](docs/tutorial.zh-cn.md) |
| Language reference | [SPEC v0.3.4](docs/spec.md) | [语言参考](docs/spec.zh-cn.md) |
| Configuration | [Configuration](docs/configuration.md) | [配置](docs/configuration.zh-cn.md) |
| Provider setup | [Providers](docs/providers.md) | [厂商接入](docs/providers.zh-cn.md) |
| Implemented controls | [Capabilities](docs/capabilities.md) | [能力表](docs/capabilities.zh-cn.md) |
| Provider research | [Research](docs/market_survey.md) | [厂商调研](docs/market_survey.zh-cn.md) |
| Design rationale | [Design](docs/reflection.md) | [设计思路](docs/reflection.zh-cn.md) |
| Contributing | [Contributing](contributing.md) | [参与贡献](contributing.zh-cn.md) |
| Editor highlighting | [VS Code / Cursor](editors/vscode/readme.md) | [VS Code / Cursor](editors/vscode/readme.zh-cn.md) |

MIT licensed. See [license](license).
