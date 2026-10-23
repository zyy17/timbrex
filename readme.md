# Timbrex

[简体中文](readme.zh-cn.md)

Timbrex turns scripts into speech. Write dialogue as plain text, add directions for delivery and pauses, and choose a text-to-speech provider to produce a WAV file.

```tbx
[happy|Good to see you!] [pause 300ms] [slow|Take your time.]
```

A script can contain several speakers with different voices. Tags apply to a passage, a speaker's turn, or later turns, so you can keep performance directions beside the words they describe. Plain text needs no markup.

## Try it locally

Install Go 1.27 and `make`, then run:

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex check examples/expressive.tbx
./timbrex synth examples/expressive.tbx --provider stub -o tone.wav
```

The check prints `ok`; synthesis prints `tone.wav`. The `stub` provider needs no account and writes an 8 kHz test tone. It lets you test file generation, but does not speak the text or reproduce its pauses and directions.

For speech, configure a provider's credentials and any required voice or endpoint, then select it with `--provider`. Follow the [tutorial](docs/tutorial.md) for a complete example, or go directly to [provider setup](docs/providers.md).

## What it supports

Timbrex includes adapters for OpenAI, ElevenLabs, MiniMax, Azure Speech, Qwen, Gemini TTS, and Fish Audio. Each maps a supported subset of directions to that provider's parameters or performance instructions. The [capability reference](docs/capabilities.md) distinguishes numeric controls from approximate instructions and unsupported features.

`check` validates the script's language. `synth` also checks the selected adapter before sending requests. A script can pass `check` and still use a control that its provider does not support. Unsupported controls are reported rather than silently removed from speech.

Real adapters synthesize segments sequentially, insert requested pauses locally, and write a 24 kHz mono 16-bit PCM WAV. There is no audio mixing, overlapping dialogue, streaming output, or automatic fallback between providers. This is a script-to-audio tool; it does not create or clone voices.

The adapters have offline request and response tests. Live account compatibility and acoustic quality remain unverified. Model availability, supported voices, and service limits are documented in [provider setup](docs/providers.md).

## Learn the language

Scripts use `.tbx` or `.timbrex`. Spoken text can use any language; tag names use ASCII identifiers. For example, `[slow|text]` applies a direction to a span, and `[pause 300ms]` inserts a pause.

[DSL Specification v1](docs/spec.md) describes the current language. Its rules may change during development. The `timbrex: 1` front-matter field selects the supported DSL version; it is separate from the CLI's software version.

## Find the right guide

| Goal | English | 简体中文 |
| --- | --- | --- |
| Write and synthesize a first script | [Tutorial](docs/tutorial.md) | [教程](docs/tutorial.zh-cn.md) |
| Look up syntax, scope, or diagnostics | [DSL Specification v1](docs/spec.md) | [DSL 规范 v1](docs/spec.zh-cn.md) |
| Set defaults, credentials, and overrides | [Configuration](docs/configuration.md) | [配置](docs/configuration.zh-cn.md) |
| Connect a speech provider | [Provider setup](docs/providers.md) | [厂商接入](docs/providers.zh-cn.md) |
| Check which controls an adapter supports | [Capabilities](docs/capabilities.md) | [能力表](docs/capabilities.zh-cn.md) |
| Develop or contribute | [Contributing](contributing.md) | [参与贡献](contributing.zh-cn.md) |
| Enable editor highlighting | [VS Code / Cursor](editors/vscode/readme.md) | [VS Code / Cursor](editors/vscode/readme.zh-cn.md) |

For background, see the [design rationale](docs/reflection.md) and [provider research](docs/market_survey.md).

MIT licensed. See [license](license).
