# Tutorial

[简体中文](tutorial.zh-cn.md) · [DSL Specification v1](spec.md)

This tutorial takes you from a short script to a WAV file, then adds speakers and performance directions. The local steps need no account. The speech example requires a TTS provider and may incur charges.

## Build the CLI

Install Go 1.27 and `make`. Run these commands in a terminal:

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex version
```

Run the remaining commands from the repository root, where `./timbrex` was built.

## Write and check a script

Create a file named `hello.tbx` with this content:

```tbx
Hello. [pause 300ms] Welcome home.
```

Ordinary text is spoken; `[pause 300ms]` requests a 300-millisecond pause. Tags are optional, so a file containing only `Hello.` is also valid.

```bash
./timbrex check hello.tbx
```

```text
ok
```

`check` parses the script and checks its tags without contacting a provider. Errors fail the command. Warnings and informational messages go to stderr and do not prevent `ok` on stdout. Passing this check does not establish that a particular provider supports every direction in the script.

## Produce a test file

```bash
./timbrex synth hello.tbx --provider stub -o tone.wav
```

```text
tone.wav
```

Open `tone.wav` in an audio player. You will hear a test tone, not the words from the script: `stub` tests file generation and does not perform speech or insert the requested pauses. It is the default provider and needs no credentials.

`-o` / `--out` selects the output path and format: `.wav`, `.mp3`, `.flac`, `.opus`, or `.aac`. The parent directory must already exist. WAV needs no FFmpeg; other extensions require a trusted `ffmpeg` on `PATH` and pass a local encoding check before any provider request. See [audio output](audio-output.md) for presets and limits. Without `-o`, `hello.tbx` produces `hello.wav` beside the script. Timbrex rejects an output path that would replace the source script.

## Generate speech

Choose a provider from [provider setup](providers.md). For an OpenAI example, supply your key through `OPENAI_API_KEY` **or** the `providers.openai.api_key` configuration field. Then run:

```bash
# Configure your OpenAI credentials before running this command.
./timbrex synth hello.tbx --provider openai --voice coral -o speech.wav
```

A successful run prints `speech.wav`. If neither credential source is set, it fails before sending a request:

```text
error: set providers.openai.api_key or its documented environment variable
```

Keep keys out of scripts and source control. See [configuration](configuration.md) for file and environment settings. The OpenAI adapter's default model is `gpt-4o-mini-tts`; check its [service limits and lifecycle notice](providers.md#openai) before relying on it. Other adapters have different model, voice, and endpoint requirements.

Real adapters compose 24 kHz mono 16-bit PCM WAV before optional final-file export. They synthesize speech segments sequentially by default and add pauses locally. Their request contracts have offline tests; live account compatibility and acoustic quality have not been verified in this repository.

For a script with several speech segments, you can try `--concurrency 2` with a real provider to overlap requests. The default is `1`; the accepted range is `1`–`8`. Keep within your account limits: higher values can increase memory use or trigger rate limits. This does not change the segment text, voice, controls, pauses, or final audio order. `stub` only accepts `1`. See [concurrent generation](providers.md#concurrent-generation) for failure and billing limits.

## Add directions and strict checking

Tags have three forms: `[name]`, `[name args]`, and `[name args|content]`. Tag names use ASCII identifiers; spoken text can use any language.

Create `expressive.tbx` with the following content. It uses the same tag structure as [examples/expressive.tbx](../examples/expressive.tbx):

```tbx
---
timbrex: 1
strict_tags: true
---
[happy|Good to see you!] [pause 300ms] [slow|Take your time.]
```

```bash
./timbrex check expressive.tbx
```

The YAML section at the beginning is called *front matter*. It is optional. When present, it starts at the beginning of the file between two `---` lines. Here, `timbrex: 1` selects the current DSL version and `strict_tags: true` rejects unregistered names. [DSL Specification v1](spec.md) describes the current rules, which may change during development.

Without strict mode, unregistered tags become free-form acting directions. This is convenient while writing, but can hide spelling mistakes. Use strict mode when you want names checked, and write free-form directions explicitly as `[act "thoughtfully"]`.

To see the difference, create `unknown.tbx` containing this line:

```text
[thoughtfully] Hello.
```

```bash
./timbrex check unknown.tbx --strict-tags
```

```text
unknown.tbx:1:1: error[V041]: unregistered tag "thoughtfully"; for free-form acting write [act "thoughtfully"]
error: check failed
```

Replacing `[thoughtfully]` with `[act "thoughtfully"]` makes this strict check pass. `--strict-tags` and `--no-strict-tags` override both front matter and configuration.

## Understand scope

Directions such as `happy` or `slow` can apply at three levels:

- A span, such as `[slow|Take your time.]`, applies only to its enclosed text. Nested spans restore the surrounding state when they end.
- A point tag inside speech, such as `Hello. [sad] I missed you.`, applies from that point to the end of the turn. An unprefixed paragraph is a narration turn; a blank line ends it.
- A line containing only point tags is a scene line. State set there applies to later turns until another scene line changes it. An event such as `[pause 1s]` occurs once rather than becoming persistent state.

Create `emotion.tbx`:

```tbx
---
strict_tags: true
---
[happy]
Hello. [sad] I missed you. [reset] Welcome back.

[slow|Take your time.] [act "quietly relieved"|You made it.]
[reset]
Goodbye.
```

```bash
./timbrex check emotion.tbx
./timbrex synth emotion.tbx --provider stub -o emotion.wav
```

The first `[happy]` sets the scene's emotion. Inside the first paragraph, `[sad]` changes it and `[reset]` restores the scene's `happy`. The second paragraph also inherits `happy`; its spans add local directions. The final standalone `[reset]` clears scene controls, so `Goodbye.` has none of those directions.

Inside speech, `[reset]` restores scene defaults, even inside a nested span. It does not mean “restore the immediately enclosing span.” A named reset, such as `[reset speed]`, affects only that control. See the [scope reference](spec.md) for details.

The stub command above checks file generation only. To hear the directions, use a provider that supports them. Numeric controls and performance instructions are different: for example, Gemini's `slow` is an approximate style request, while its numeric `speed` control is unsupported. Consult the [capability reference](capabilities.md) before changing providers.

## Use more than one speaker

Create `cast.tbx`:

```tbx
---
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
./timbrex check cast.tbx
./timbrex synth cast.tbx --provider stub -o cast.wav
```

For actual dialogue, select `--provider openai` after configuring credentials. Each `voices.openai` value selects that character's voice. `--voice` only supplies a fallback for speakers without a cast mapping; it does not override these two entries.

A leading `id:` starts a speaker turn only if the ID is in `cast`, or if front matter sets `cast: auto`. Otherwise the prefix remains ordinary text: it continues the current speaker’s turn, or starts narration in a new paragraph. A display name is not an alias: use `lin:`, not `Lin:`. Unprefixed narration uses `narrator` unless front matter selects another ID with `narrator:`.

`cast: auto` recognizes speaker prefixes without declaring them and reports each discovered ID during `check`. It does not choose distinct voices. Use explicit cast mappings when voices matter. `[as lin|text]` uses a known speaker's mapping; `[voice "vendor-id"|text]` selects a vendor voice directly.

## Insert pauses

Create `pauses.tbx`:

```tbx
Wait. [pause 500ms] Now go. [pause short] One more beat.
```

```bash
./timbrex check pauses.tbx
./timbrex synth pauses.tbx --provider stub -o pauses.wav
```

With a real provider, `pause`, `silence`, `break`, and nonnegative `gap` insert local silence. `gap` must lead an explicitly recognized speaker turn, before spoken text or state tags, or appear on a scene line. It is not allowed at the start of unprefixed narration; use `pause` there. The default is 250 ms; `short`, `medium`, and `long` mean 250 ms, 500 ms, and 1 s. Explicit durations use `ms`, `s`, or `min`, from `0ms` to `1min`. Write zero as `0ms`: `[pause 0]` is a number rather than a duration and fails `check`.

There is no implicit pause between turns. Use a pause where one is needed. Overlapping speech is not implemented. The stub output above does not reproduce pauses; switch to a real provider to hear them.

## Reuse settings

Provider, model, fallback voice, and strict mode can come from configuration instead of being repeated on each command. Their priority is flags, script front matter, environment, the selected YAML file, then defaults. Cast voice mappings still take priority over the fallback voice.

By default, configuration comes from `timbrex.yaml` in the current working directory. `--config` selects a different file. To inspect the merged file and environment settings:

```bash
./timbrex config
```

Configured API keys appear as `REDACTED`. This command does not read a script, apply `synth` flags, or test credentials. The [configuration reference](configuration.md) covers all fields and overrides; [provider setup](providers.md) lists each adapter's defaults and environment variables.

Some repository examples select a real provider in their front matter. To check all examples and produce local test files without making speech requests:

```bash
mkdir -p out
for f in examples/*.tbx; do
  ./timbrex check "$f"
  ./timbrex synth "$f" --provider stub -o "out/$(basename "${f%.tbx}").wav"
done
```

## Troubleshoot a result

| Symptom | What to check |
| --- | --- |
| `check` prints `ok`, but `synth` reports an unsupported control | Language validation and adapter support are separate. Check the selected model in the [capability reference](capabilities.md). |
| The output is just a tone | You used `stub`, which is the default. Configure and select a real provider. |
| `--voice` does not change a character's voice | The character has a cast mapping, or a `voice`/`as` direction is active. `--voice` is only the fallback. |
| A name followed by `:` is spoken aloud | Declare that exact ID in `cast`, or use `cast: auto`. Display names are not aliases. |
| Synthesis asks for a key despite an exported variable | Check the selected provider and its documented variable name; inspect merged settings with `config`. A key for one provider cannot configure another. |
| No output file is created | Read stderr. Check the output directory, script diagnostics, and provider error. A failed real synthesis preserves an existing destination; earlier requests may already have incurred charges. |

Diagnostics include a file, line, and column where available. `V041` means an unregistered strict-mode tag, `V012` an unclosed tag, and `W103` a display name used as a speaker prefix. The [DSL reference](spec.md) lists diagnostics; [provider setup](providers.md) explains service failures and limits.
