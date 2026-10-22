# Tutorial

[简体中文](tutorial.zh-cn.md) · [Language reference](spec.md)

This walkthrough installs Timbrex, writes a `.tbx` script, checks it, inspects settings, synthesizes a local tone with `stub`, then uses one HTTP provider. Later sections add speakers, pauses, emotions, and a batch loop. Every command was run against the current CLI (`timbrex 0.1.0`). There is no `compile` or `inspect` command.

## 1. Install

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

From the repository root, `go run .` runs the same program as `./timbrex`. There is no `cmd/timbrex` package. The rest of this tutorial uses `./timbrex` after `make build`.

## 2. Write a first script

Create `hello.tbx`. Plain text is valid. Tags are optional.

```tbx
Hello. [pause 300ms] Welcome home.
```

The only markup form is `[name]`, `[name args]`, or `[name args|content]`. Tag names are English ASCII. Spoken text may be any language.

A slightly richer script, matching [examples/expressive.tbx](../examples/expressive.tbx):

```tbx
---
timbrex: 1
strict_tags: true
---
[happy|Good to see you!] [pause 300ms] [slow|Take your time.]
```

`timbrex: 1` is the frozen grammar version ([SPEC v0.3.4](spec.md)). `strict_tags: true` rejects unknown tag names. Front matter is optional; if present, the file must start with a line that is exactly `---` (no trailing spaces).

## 3. Check the script

`check` parses the file, resolves tags, and prints diagnostics. It does not need credentials or a network.

```bash
./timbrex check hello.tbx
```

```text
ok
```

Checked-in examples:

```bash
./timbrex check examples/hello.tbx
./timbrex check examples/expressive.tbx
./timbrex check examples/dialogue.tbx
```

Each prints `ok` on stdout. Warnings and info go to stderr and still allow `ok`. Errors fail the command.

Unknown names become free-form acting unless strict mode is on. `--strict-tags` / `--no-strict-tags` override front matter and configuration.

```bash
./timbrex check hello.tbx --strict-tags
```

A strict-mode unknown tag (not a `hello.tbx` snippet; shown as text so doc tests do not parse it as a script):

```text
[thoughtfully] Hello.
```

```text
script.tbx:1:1: error[V041]: unregistered tag "thoughtfully"; for free-form acting write [act "thoughtfully"]
error: check failed
```

Write free-form acting explicitly: `[act "thoughtfully"]`.

## 4. Inspect settings and diagnostics

Timbrex has no `compile` or `inspect` command. Use `check` for the language, and `config` for merged file and environment settings (API keys become `REDACTED`). `config` does not read a script or apply `synth` flags.

```bash
./timbrex config
```

Without provider environment variables, built-in defaults look like this:

```yaml
provider: stub
strict_tags: false
providers:
    azure:
        voice: en-US-JennyNeural
    elevenlabs:
        model: eleven_v3
        endpoint: https://api.elevenlabs.io/v1
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

`cast: auto` is accepted by `check`. Each discovered speaker is listed once on stderr:

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

Coded diagnostics you will see while learning:

| Code | Severity | Meaning |
| --- | --- | --- |
| `V012` | error | Unclosed tag |
| `V013` | error | `[/name]` is invalid; spans use `[name\|...]` |
| `V041` | error | Unregistered tag in strict mode |
| `W040` | warning | Unknown single-word tag close to a registered name |
| `W103` | warning | Cast display name used as a speaker prefix |
| `I050` | info | Non-ASCII free-form acting text |

Parse errors (such as `V012`) print as `file:line:col: …`. Check failures add `error: check failed`.

## 5. Synthesize with stub

`stub` is the default provider. It writes a short 8 kHz tone from the script's rune count. It is not a speech engine. Use it to exercise the pipeline without a key.

```bash
./timbrex synth hello.tbx --provider stub -o tone.wav
```

```text
tone.wav
```

The printed line is the output path. `-o` / `--out` must end in `.wav`. If you omit `-o`, the default is the script path with `.wav` (here `hello.wav`). The source script cannot be overwritten.

The same command works on the repository examples:

```bash
./timbrex synth examples/expressive.tbx --provider stub -o tone.wav
```

```text
tone.wav
```

`file tone.wav` reports `WAVE audio, Microsoft PCM, 16 bit, mono 8000 Hz`. Real providers write 24 kHz WAV instead.

## 6. Synthesize with a real provider

HTTP adapters need credentials. OpenAI is the smallest extra setup: one environment variable. Without it, `synth` fails before any request:

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

On success, stdout is the output path (`speech.wav`). Default model is `gpt-4o-mini-tts`. Live account compatibility and acoustic quality have not been verified in this repository.

`--provider` wins, then front matter `provider:`, then configuration (default `stub`). [examples/dialogue.tbx](../examples/dialogue.tbx) sets `provider: openai`; pass `--provider stub` if you want a local tone from that file.

Other adapters need more than a key: ElevenLabs needs a voice ID; Azure needs a region or endpoint; Qwen needs a workspace endpoint. See [provider setup](providers.md).

## 7. Speakers and cast

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

`cast.<id>.voices.<provider>` selects that speaker's voice. `--voice` is a fallback; it does not replace cast mappings. A display name is not a speaker alias. `Lin:` in the body stays narration and may warn `W103`.

## 8. Pauses

`pause`, `silence`, `break`, and a nonnegative `gap` become local silence during `synth`. They are not sent to the vendor. Default pause is 250 ms. `short`, `medium`, and `long` are 250 ms, 500 ms, and 1 s. Explicit durations use `ms`, `s`, or `min`, from `0ms` to `1min`. `0ms` is accepted and produces no samples.

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

There is no implicit pause between turns. `overlap` is reserved and rejected by real adapters.

## 9. Emotions and styles

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

What a provider does with those tags differs. OpenAI and Qwen send many of them as instructions. MiniMax maps seven emotions. Azure maps styles. ElevenLabs uses audio tags on v3/v4 models. `stub` ignores performance and still writes a tone. See [capabilities](capabilities.md) before targeting a real adapter.

## 10. Batch

There is no batch command. A shell loop is enough. `--provider stub` is required here because [examples/dialogue.tbx](../examples/dialogue.tbx) sets `provider: openai`.

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
out/hello.wav
```

`check` stays offline. `synth` without `stub` needs that provider's credentials.

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
| `stub` | Default provider; 8 kHz test tone, no speech |
| strict mode | Unknown tags are `V041`; write `[act "…"]` instead |
| provider | `stub`, `openai`, `elevenlabs`, `minimax`, `azure`, or `qwen` |

## Next

- [SPEC v0.3.4](spec.md) — grammar, vocabulary, diagnostics
- [Configuration](configuration.md) — flags, files, environment
- [Providers](providers.md) — credentials and request shapes
- [Capabilities](capabilities.md) — what each adapter implements
