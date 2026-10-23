# Timbrex

[简体中文](readme.zh-cn.md)

Timbrex is a scripting language and CLI for turning written dialogue into speech. Put the words, speakers, delivery directions, and pauses in a text file, then render it to WAV through a text-to-speech (TTS) provider.

A TTS API handles speech generation. Timbrex handles the script around those calls: which voice speaks each line, which directions apply to a passage, and where to insert silence. Those choices stay beside the dialogue in a file you can edit, review, and version in Git; the adapter builds the requests and assembles the audio.

Use it for narrated tutorials, voiceover drafts, or scripted dialogue prototypes where you want to revise the performance along with the words.

## A script at a glance

This dialogue assigns two OpenAI voices and adds a pause and delivery directions:

```tbx
---
strict_tags: true
cast:
  lin: {voices: {openai: coral}}
  chen: {voices: {openai: onyx}}
---
lin: [happy] You came! [pause 300ms] [whisper|Come inside.]
chen: [slow|I've got something to tell you.]
```

The `cast` section makes `lin:` and `chen:` speaker turns and maps them to `coral` and `onyx`; those voice IDs are specific to OpenAI. `[happy]` applies through Lin's turn, `[whisper|…]` and `[slow|…]` apply to their enclosed words, and `[pause 300ms]` inserts local silence between phrases.

The runnable version is [examples/dialogue.tbx](examples/dialogue.tbx). Scripts use `.tbx` or `.timbrex`; plain text works too, and spoken text can use any language. Supported languages, voices, and directions depend on the selected provider and model.

## Quick start

Install Git, Go 1.27, and `make`. In a terminal, run:

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex check examples/dialogue.tbx
./timbrex synth examples/dialogue.tbx --provider stub -o tone.wav
```

`check` prints `ok` without contacting a speech service. The last command prints `tone.wav` and creates a file you can open in an audio player. The `stub` provider needs no credentials and makes no speech requests: it produces a test tone, without speaking the dialogue or reproducing its pauses and directions.

### Generate speech

For the OpenAI voices in this example, configure `OPENAI_API_KEY` in your environment, or set `providers.openai.api_key` in your [configuration file](docs/configuration.md). Keep the key out of the script and source control. Then, from the same repository directory, run:

```bash
./timbrex synth examples/dialogue.tbx --provider openai --model gpt-4o-mini-tts -o dialogue.wav
```

This command sends requests to OpenAI and may incur charges. On success, it prints `dialogue.wav`. Check the [OpenAI setup and model lifecycle notice](docs/providers.md#openai) before using the service. To use another provider, follow its [setup guide](docs/providers.md) and supply compatible voice IDs and controls.

The [tutorial](docs/tutorial.md) walks through writing your own script, adding speakers, and changing delivery within a passage.

## Providers and current scope

Adapters are included for **OpenAI, ElevenLabs, MiniMax, Azure Speech, Qwen, Gemini TTS, and Fish Audio**. Each supports a different subset of controls: some become numeric API parameters, while others become approximate performance instructions. Changing providers does not promise the same voice or delivery. See the [capability reference](docs/capabilities.md) for the supported mappings.

Real adapters validate all planned speech segments before sending synthesis requests. Passing `check` establishes language validity, not provider support; an unsupported control used by a speech segment fails during synthesis preparation.

Speech is synthesized sequentially, with pauses added locally, into a **24 kHz mono 16-bit PCM WAV**. Mixing, overlapping dialogue, streaming output, automatic provider fallback, and voice creation or cloning are not implemented.

The project is under development. [DSL Specification v1](docs/spec.md) describes the current language and may evolve. Adapters have offline request and response tests; live account compatibility and acoustic quality remain unverified.

## Documentation

| Start here when you want to… | Guide |
| --- | --- |
| Write and synthesize your first script | [Tutorial](docs/tutorial.md) |
| Look up syntax, scope, or diagnostics | [DSL Specification v1](docs/spec.md) |
| Set defaults, credentials, and overrides | [Configuration](docs/configuration.md) |
| Choose a provider and supported controls | [Provider setup](docs/providers.md) · [Capabilities](docs/capabilities.md) |
| Enable syntax highlighting | [VS Code / Cursor](editors/vscode/readme.md) |

## Contributing

Bug reports, documentation improvements, and provider contributions are welcome. A useful bug report includes a minimal script, the command, and the expected and actual results, with credentials and private text removed.

The [contributor guide](contributing.md) covers development setup, package boundaries, offline tests, and pull requests. For the reasoning behind the language and implementation, see the [design notes](docs/reflection.md).

## License

[MIT](license).
