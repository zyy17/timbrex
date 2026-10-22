# Timbrex language reference

[简体中文](spec.zh-cn.md) · [Provider capabilities](capabilities.md)

Grammar version: **`timbrex: 1`**. The syntax is frozen. This reference describes language rules and identifies current synthesis support separately.

## 1. Text and tags

A `.tbx` file is UTF-8 text. `.timbrex` is an equivalent extension. A leading BOM is removed and CRLF/CR newlines become LF. Plain text is a valid script.

```tbx
Hello.
[happy|Hello again!] [pause 500ms] Welcome home.
```

The only markup form is `[name args]` or `[name args|content]`. Tags can nest. Names, identifiers, and configuration keys use English ASCII. Spoken text, quoted strings, display names, and free-form acting may use any language. Full-width marks such as `［］` and `｜` are ordinary text, not alternate syntax.

A tag header ends at the first unescaped `|` or `]` outside a quoted argument. Double quotes in content are ordinary spoken punctuation: `[whisper|She said "hello".]`. A span may cross non-empty lines but must close within its paragraph. BBCode closing tags such as `[/whisper]` are invalid.

| Form | Meaning |
| --- | --- |
| `[happy]` within a turn | Set emotion for the remaining turn |
| `[happy|words]` | Set emotion only for these words |
| `[happy]` on its own line | Set scene emotion for subsequent turns |
| `[pause 500ms]` | Insert an event here |
| `[whisper|Hello]` on its own line | Speak the span as narration |

A **scene line** contains only tags without spans, plus optional whitespace. Events on a scene line happen between turns. A scene state persists until replaced or reset.

## 2. Speakers and paragraphs

A blank line ends a paragraph. A declared `speaker_id:` at column zero starts a new turn even without a blank line. Subsequent ordinary lines remain in that turn. Speaker-like text inside a span remains content.

```tbx
---
cast:
  lin: {name: 林}
  chen: {name: 陈}
---
lin: Hello.
Still Lin speaking.
chen: Hello back.

This is narration.
```

Speaker IDs match `[A-Za-z][A-Za-z0-9_.-]*` and are case-sensitive. Only an ID declared in `cast` starts a turn. A display name is not an alias. Without a declaration, `note: hello` is ordinary text. `cast: auto` accepts every leading ASCII ID and `check` reports the discovered IDs. `lin\: hello` escapes a possible speaker prefix.

Narration uses `narrator`, unless front matter `narrator:` supplies another ASCII ID. A cast entry for that ID can assign its voice. Display names are metadata; adapters currently use IDs and voice mappings, not display names as acting instructions.

## 3. Values, escaping, and comments

Arguments are separated by spaces or tabs. Positional values and `key=value` are parsed without imposing a vendor's parameter schema.

| Value kind | Examples |
| --- | --- |
| Number | `0.7`, `-1`, `+2.5` |
| Duration | `500ms`, `1.5s`, `1min` |
| Percent | `120%` |
| Semitones | `+2st` |
| Decibels | `-3dB` |
| Word | `warm`, `en-US` |
| Quoted string | `"holding back tears"`, `"轻轻地"` |
| Named argument | `intensity=0.5` |

A suffix belongs to a number only if it ends the token: `10stone` is a word. Quoted values support `\"` and `\\`. A registered tag may accept trailing words as a note; the current note-enabled tags are `act` and nonverbal events. The checker does not yet enforce per-tag argument types or ranges. Each real adapter validates the parameters it implements before making requests; named arguments are not currently implemented by these adapters.

Numeric synthesis controls require unquoted values of the matching kind. For example, use `speed 1.2`, not `speed "1.2"` or `speed 1e0`. Pauses accept the duration units `ms`, `s`, and `min`; Go-specific forms such as `1m` and `1s250ms` are words, not durations. Write zero duration as `0ms`.

Outside quoted arguments, `[[` and `\[` produce a literal `[`. `]]` and `\]` produce a literal `]`. Backslash can also escape `|`, `\`, and `:`. Other backslashes are literal. Inside two or more open tags, consecutive `]` close nested tags; use `\]` for a literal closing bracket there. Thus `[whisper|say [pause 1s]]` closes both tags.

A line whose first non-space characters are `//` is a comment outside tags. Inline `//`, `#`, and HTML comments are spoken text. Parentheses and angle brackets are plain language text, but some providers reserve them for native markup; see [capabilities](capabilities.md#literal-text).

Raw fences start with three backticks and an info string whose first word is an ASCII identifier, followed by an opaque info tail. The parser currently validates only the first word. They end at a line containing only three backticks and optional whitespace. Their contents are not parsed as Timbrex. All current real adapters reject fences; parsing them does not imply passthrough support.

## 4. Tag resolution and strict mode

Registered tag names are case-insensitive and treat `_` like `-`: `[HAPPY]` and `[clear_throat]` resolve to `happy` and `clear-throat`. Speaker IDs and parameter values do not receive this normalization.

Use `[act "holding back tears"]` for an explicit acting instruction. Its span form is `[act "holding back tears"|I'm fine.]`. Bare note text, `[act holding back tears]`, is also recognized. The chosen provider must support instructions.

By default, an unknown tag is implicit acting. `[thoughtfully]` becomes an `act` instruction with its original text. With `strict_tags: true` or `--strict-tags`, unknown tags are errors. `--no-strict-tags` overrides front matter in the other direction. Both flags together are invalid. For checked-in scripts, explicit `act` and strict mode avoid accidental acting instructions caused by typos.

The parser recognizes the shape `ns:tag`, but no provider namespaces or external vocabularies are registered yet. Such names currently follow the unknown-tag rules. They do not invoke vendor APIs directly.

| Code | Current diagnostic |
| --- | --- |
| V012 | Unclosed tag |
| V013 | Invalid BBCode-style closing tag |
| V041 | Unregistered tag in strict mode |
| W040 | Unknown single-word non-strict tag close to a registered name, with a suggestion |
| W103 | Cast display name used instead of its speaker ID |
| I050 | Non-ASCII free-form acting text |

Suggestions use edit distance at most 2. `[paus]` suggests `[pause]`; `[laff]` has no suggestion for `[laugh]` (distance 3). Warnings and information do not fail `check`; errors do. Other parse and position errors can be reported without a diagnostic code. There is no automatic translation or spelling repair.

## 5. Scope and vocabulary

The vocabulary declares four kinds:

| Kind | Behavior |
| --- | --- |
| State | A point tag lasts to the end of the turn, or persists from a scene line; a span applies locally |
| Event | Occurs at its position; only designated events allow accompanying-speech spans |
| Wrap | Requires content: `[lang en-US|Hello]` |
| Turn | Only at the beginning of a turn, or on a scene line |

Inner controls replace outer controls of the same feature. Different features coexist when the provider can represent them. The shared planner groups `speed/fast/slow`, `voice/as`, `whisper/shout`, and the emotions into their respective features. Leaving a span restores its enclosing state. Each new turn starts from scene state.

`[reset]` in a turn restores scene state. `[reset speed]` restores just scene speed, or removes turn speed if no scene speed exists. A scene-line reset removes scene state. `as` selects another cast ID's voice; `voice` selects a vendor voice ID directly. Both require spans.

The standard vocabulary includes these families. **This is a language inventory, not a provider support promise.** The executable list is [internal/vocab/vocab.go](../internal/vocab/vocab.go).

| Family | Names |
| --- | --- |
| Emotions (state) | `neutral happy sad angry fear surprise disgust calm excited tender serious curious sarcastic nervous tired bored contempt confused proud embarrassed hopeful disappointed panicked crying` |
| Delivery (state) | `emotion act speed fast slow pitch volume whisper shout emph style role accent oral dialect emovec` |
| State reset | `reset` (no span) |
| Pauses and timing (event) | `pause silence duration break rate` |
| Nonverbal (event, optional span) | `laugh sigh breath cough gasp inhale exhale quick-breath pant huff groan grunt moan chuckle giggle cackle snort scoff cry sob sniffle sniff whimper wail clear-throat sneeze hiccup yawn burp snore swallow gulp lip-smack tongue-click kiss shush whistle hum hiss hmm uh um emm` |
| Pronunciation and text (wrap) | `pinyin ipa sub say arpabet jyutping romaji spell verbatim latex lang` |
| Voice (wrap) | `as voice` |
| Turn timing | `overlap gap` |
| Audio and markers (event) | `sfx audio bgm ambience vc cut mark` |

## 6. Front matter

YAML front matter is optional and must begin at the start of the file, between `---` lines.

```yaml
---
timbrex: 1
provider: openai
model: gpt-4o-mini-tts
voice: coral
lang: en-US
strict_tags: true
narrator: guide
cast:
  guide:
    name: Guide
    voices:
      openai: coral
      azure: en-US-JennyNeural
---
```

Supported keys are `timbrex`, `provider`, `model`, `voice`, `lang`, `strict_tags`, `narrator`, and `cast`. `cast` is either `auto` or a map of IDs with optional `name` and `voices`. If supplied, `timbrex` must be `1`. Unknown YAML keys are errors, including misspelled cast fields; future configuration is not silently accepted.

Precedence is documented in [README](../readme.md#commands-and-output). Voice IDs and models are vendor-specific. `lang` is a hint interpreted by the selected adapter; unsupported hints may fail. Azure chooses models through voice names and rejects `model:`. Keep credentials in environment variables, never in scripts.

## 7. Synthesis contract

`check` validates the language and vocabulary. `synth` additionally validates the current adapter's mapping, then renders sequentially. Unsupported controls, raw fences, and invalid parameter values used by a speech segment fail before the first paid request. State that is overridden before any speech has no audible effect.

Pauses (`pause`, `silence`, `break`, and a nonnegative `gap`) are local silence. Default pause is 250 ms; `short`, `medium`, and `long` mean 250 ms, 500 ms, and 1 s. Explicit durations range from 0 to 1 minute per pause. `fast` is 1.2× and `slow` is 0.8×. `speed 120%` means 1.2×, not an increase of 120%. Pitch uses semitones and volume uses dB; supported ranges depend on the adapter. Emotion intensities are interpreted as text only where documented, not as calibrated measurements.

The renderer concatenates compatible raw PCM segments into one WAV. It does not mix, overlap, resample, normalize, or crossfade. There is no implicit pause between turns. Use explicit pauses for timing. [Provider setup](providers.md) and [capabilities](capabilities.md) describe the exact current mappings and limits.

## 8. Future work, not current behavior

The frozen syntax leaves room for external vocabularies, registered provider namespaces, raw passthrough, capability declarations, lock files, and dataset templates outside the speech body. External YAML configuration is already implemented; see [configuration](configuration.md). Pronunciation, mixing, fallback planning, streaming, native dialogue, and voice management need separate implementations and tests. The grammar does not include loops or conditionals.

`fix`, `fmt`, `expand`, `compile`, `caps`, `voices`, `tags`, `migrate`, an LSP, and a public Go SDK are not shipped commands or APIs. Old design sketches of these features are not compatibility promises. New work should preserve the simple authoring model and document real behavior in both languages.
