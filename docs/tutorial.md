# Tutorial

[简体中文](tutorial.zh-cn.md) · [Language reference](spec.md)

Timbrex compiles a `.tbx` script to a WAV file. This page runs `check` and `synth` against the current CLI (`timbrex 0.1.0`). The CLI has no `compile` or `inspect` command.

## Prerequisites

Requires Go 1.27 ([go.mod](../go.mod)).

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex version
```

```text
timbrex 0.1.0
```

From the repository root, `go run .` is the same program as `./timbrex`. There is no `cmd/timbrex` package. Later steps use `./timbrex` after `make build`.

## Write a script

Create `hello.tbx`. Plain text is a valid script. Tags are optional.

```tbx
Hello. [pause 300ms] Welcome home.
```

Markup is `[name]`, `[name args]`, or `[name args|content]`. Tag names are English ASCII. Spoken text may be any language.

The following script matches [examples/expressive.tbx](../examples/expressive.tbx):

```tbx
---
timbrex: 1
strict_tags: true
---
[happy|Good to see you!] [pause 300ms] [slow|Take your time.]
```

`timbrex: 1` is the frozen grammar version ([SPEC v0.3.4](spec.md)). `strict_tags: true` rejects unknown tag names. Front matter is optional. If present, it must begin at the start of the file between `---` lines. The opening and closing `---` lines may have trailing spaces or tabs.

## Check a script

`check` parses the file and resolves tags. It does not use credentials or the network.

```bash
./timbrex check hello.tbx
```

```text
ok
```

Repository examples:

```bash
./timbrex check examples/hello.tbx
./timbrex check examples/expressive.tbx
./timbrex check examples/dialogue.tbx
./timbrex check examples/gemini.tbx
./timbrex check examples/fish.tbx
```

Success prints `ok` on stdout. Warnings and info go to stderr and still print `ok`. An error fails the command.

Unknown names become free-form acting unless strict mode is on. `--strict-tags` and `--no-strict-tags` override front matter and configuration.

```bash
./timbrex check hello.tbx --strict-tags
```

The following line is invalid in strict mode. The block is fenced as `text` so `go test` does not parse it as a script:

```text
[thoughtfully] Hello.
```

```text
script.tbx:1:1: error[V041]: unregistered tag "thoughtfully"; for free-form acting write [act "thoughtfully"]
error: check failed
```

Write free-form acting as `[act "thoughtfully"]`.

## Inspect configuration

`check` validates language. `config` prints merged file and environment settings. Configured API keys are shown as `REDACTED`. `config` does not read a script or apply `synth` flags.

```bash
./timbrex config
```

Without provider environment variables, built-in defaults are:

```yaml
provider: stub
strict_tags: false
providers:
    azure:
        voice: en-US-JennyNeural
    elevenlabs:
        model: eleven_v3
        endpoint: https://api.elevenlabs.io/v1
    fish:
        model: s2.1-pro
        endpoint: https://api.fish.audio/v1
    gemini:
        model: gemini-3.8-flash-tts
        voice: Kore
        endpoint: https://generativelanguage.googleapis.com/v1beta
    minimax:
        model: speech-2.8-hd
        voice: English_expressive_narrator
        endpoint: https://api.minimax.io/v1
    openai:
        model: gpt-4o-mini-tts
        voice: coral
        endpoint: https://api.openai.com/v1
    qwen:
        model: qwen-audio-3.1-tts-flash
        voice: longanhuan_v3.1
    stub: {}
```

`check` accepts `cast: auto` and lists each discovered speaker once on stderr:

```tbx
---
timbrex: 1
cast: auto
---
lin: Hello.
chen: Hi.
```

```text
script.tbx:5:1: info: cast auto recognized speaker "lin"
script.tbx:6:1: info: cast auto recognized speaker "chen"
ok
```

## Synthesize with stub

`stub` is the default provider. It writes an 8 kHz tone from the script's rune count. It is not a speech engine and does not require a key.

```bash
./timbrex synth hello.tbx --provider stub -o tone.wav
```

```text
tone.wav
```

Stdout is the output path. `-o` / `--out` must end in `.wav`. If `-o` is omitted, the default is the script path with `.wav` (here `hello.wav`). The source script cannot be overwritten.

```bash
./timbrex synth examples/expressive.tbx --provider stub -o tone.wav
```

```text
tone.wav
```

`file tone.wav` reports `WAVE audio, Microsoft PCM, 16 bit, mono 8000 Hz`. HTTP adapters write 24 kHz WAV.

## Synthesize with OpenAI

HTTP adapters require credentials. OpenAI requires `OPENAI_API_KEY`. Without it, `synth` fails before any request:

```bash
./timbrex synth examples/expressive.tbx --provider openai -o speech.wav
```

```text
error: set providers.openai.api_key or its documented environment variable
```

```bash
export OPENAI_API_KEY=your-key
./timbrex synth examples/expressive.tbx --provider openai --voice coral -o speech.wav
```

On success, stdout is the output path (`speech.wav`). The default model is `gpt-4o-mini-tts`. Live account compatibility and acoustic quality have not been verified in this repository.

Provider selection: `--provider`, then front matter `provider:`, then configuration (default `stub`). [examples/dialogue.tbx](../examples/dialogue.tbx) sets `provider: openai`. Pass `--provider stub` to write a local tone from that file.

Other adapters require additional settings: ElevenLabs and Fish need a voice ID; Azure needs a region or endpoint; Qwen needs a workspace endpoint; Gemini rejects numeric speed. See [provider setup](providers.md).

## Declare speakers

A line that starts with `id:` is a turn only when that id is in `cast`, or when `cast: auto`. Otherwise it is narration. Narration uses speaker `narrator` unless front matter sets `narrator:`.

```tbx
---
timbrex: 1
cast:
  lin:
    name: Lin
    voices: {openai: coral}
  chen:
    name: Chen
    voices: {openai: onyx}
---
lin: You came! [pause 300ms] [whisper|Come inside.]
chen: [slow|I've got something to tell you.]
```

```bash
./timbrex check examples/dialogue.tbx
./timbrex synth examples/dialogue.tbx --provider stub -o dialogue.wav
```

```text
ok
dialogue.wav
```

`cast.<id>.voices.<provider>` selects that speaker's voice. `--voice` is a fallback; it does not replace cast mappings. A display name is not a speaker alias. `Lin:` in the body remains narration and may emit `W103`.

## Insert pauses

`pause`, `silence`, `break`, and a nonnegative `gap` become local silence in the shared PCM renderer used by HTTP adapters. They are not sent to the vendor. `stub` does not insert silence; it writes one tone from rune count. Default pause is 250 ms. `short`, `medium`, and `long` are 250 ms, 500 ms, and 1 s. Explicit durations use `ms`, `s`, or `min`, from `0ms` to `1min`. Write zero as `0ms`. `[pause 0]` is a number, not a duration, and `check` rejects it.

```tbx
---
timbrex: 1
strict_tags: true
---
Wait. [pause 500ms] Now go. [pause short] One more beat.
```

```bash
./timbrex check pauses.tbx
./timbrex synth pauses.tbx --provider stub -o pauses.wav
```

```text
ok
pauses.wav
```

There is no implicit pause between turns. `overlap` is reserved. HTTP adapters reject it.

## Set emotion and style

A point tag lasts to the end of the turn. A span applies only to its content. A line that contains only tags without spans sets scene state for later turns. `[reset]` restores scene state.

```tbx
---
timbrex: 1
strict_tags: true
---
[happy]
[happy|Good to see you!] [pause 300ms] [slow|Take your time.]
[act "holding back tears"|I'm fine.]
```

```bash
./timbrex check emotion.tbx --strict-tags
./timbrex synth emotion.tbx --provider stub -o emotion.wav
```

```text
ok
emotion.wav
```

`[happy]` on its own line is scene state. The next narration line inherits it. `[happy|…]` is a local span. `[act "…"]` is the explicit acting form required in strict mode.

Adapters map these tags differently. OpenAI, Qwen, and Gemini send many of them as instructions. MiniMax maps seven emotions. Azure maps styles. ElevenLabs uses audio tags on v3/v4 models. Fish maps a subset of S2 cues. `stub` ignores performance and writes a tone. See [capabilities](capabilities.md) before targeting an HTTP adapter.

## Synthesize multiple scripts

There is no batch command. Use a shell loop. `--provider stub` is required here because [examples/dialogue.tbx](../examples/dialogue.tbx), [examples/gemini.tbx](../examples/gemini.tbx), and [examples/fish.tbx](../examples/fish.tbx) set a real provider.

```bash
mkdir -p out
for f in examples/*.tbx; do
  ./timbrex check "$f"
  ./timbrex synth "$f" --provider stub -o "out/$(basename "${f%.tbx}").wav"
done
```

```text
ok
out/dialogue.wav
ok
out/expressive.wav
ok
out/fish.wav
ok
out/gemini.wav
ok
out/hello.wav
```

`check` does not use the network. `synth` without `stub` requires that provider's credentials.

## Diagnostics

| Code | Severity | Meaning |
| --- | --- | --- |
| `V012` | error | Unclosed tag |
| `V013` | error | `[/name]` is invalid; spans use `[name\|...]` |
| `V041` | error | Unregistered tag in strict mode |
| `W040` | warning | Unknown single-word tag close to a registered name |
| `W103` | warning | Cast display name used as a speaker prefix |
| `I050` | info | Non-ASCII free-form acting text |

Parse errors (such as `V012`) print as `file:line:col: …`. Check failures add `error: check failed`.

## Glossary

| Term | Meaning |
| --- | --- |
| `.tbx` / `.timbrex` | Script extensions for the same language |
| tag | `[name]`, `[name args]`, or `[name args\|content]` |
| front matter | Optional YAML between opening and closing `---` lines |
| cast | Declared speaker ids; `cast: auto` accepts every leading ASCII id |
| turn | A line (and following lines) spoken by a declared speaker |
| narration | Unprefixed paragraphs; speaker `narrator` by default |
| scene line | A line of tags without spans; sets or emits between turns |
| span | `[name\|content]`; local scope |
| `check` | Parse and resolve tags; print `ok` or diagnostics |
| `synth` | Same check, then write a WAV |
| `config` | Print merged settings with keys redacted |
| `stub` | Default provider; 8 kHz test tone, not speech |
| strict mode | Unknown tags are `V041`; write `[act "…"]` instead |
| provider | `stub`, `openai`, `elevenlabs`, `minimax`, `azure`, `qwen`, `gemini`, or `fish` |

## See also

- [SPEC v0.3.4](spec.md) — grammar, vocabulary, diagnostics
- [Configuration](configuration.md) — flags, files, environment
- [Providers](providers.md) — credentials and request shapes
- [Capabilities](capabilities.md) — adapter mappings
