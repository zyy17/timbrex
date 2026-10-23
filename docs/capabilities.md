# Implemented provider capabilities

[简体中文](capabilities.zh-cn.md) · [Setup and official references](providers.md) · [Tutorial](tutorial.md)

The tables below describe Timbrex's current adapters, not everything the vendors offer. **Parameter** means a numeric API field or SSML control. **Instruction** means a requested performance that the model may interpret approximately. **Unsupported** means a local error when used by a speech segment. Voice/model restrictions still apply; these mappings have offline contract coverage, not live acoustic verification.

OpenAI instructions require a model that supports `instructions`; `tts-1` and `tts-1-hd` reject them. ElevenLabs acting and events require `eleven_v3`, `eleven_v4`, or `eleven_v4_turbo`. Explicit speed controls are rejected on `eleven_v4` and `eleven_v4_turbo`, which do not support the speed setting. For other models, the adapter's speed range is conservatively limited to the range in the linked product guide. For other ranges and language restrictions, see [provider setup](providers.md).

## Controls by provider

[OpenAI](#openai) · [ElevenLabs](#elevenlabs) · [MiniMax](#minimax) · [Azure](#azure) · [Qwen](#qwen) · [Gemini](#gemini) · [Fish](#fish)

### OpenAI

| Control | Support |
| --- | --- |
| Text and cast voices | Yes |
| `pause`, `silence`, `break`, nonnegative `gap` | Local silence |
| `speed`, `fast`, `slow` | Parameter: 0.25–4× |
| `pitch` | Unsupported |
| `volume` | Unsupported |
| Standard emotions | Instruction |
| Emotion intensity, e.g. `happy 0.8` | Text instruction |
| `whisper` / `shout` | Instruction |
| `act`, `accent`, `dialect`, custom `emotion` | Instruction |
| `style`, `role` | Instruction |
| `emph` | Instruction |
| `laugh`, `cry`, `sigh` **spans** | Instruction |
| Standalone nonverbal events | Unsupported |
| `lang` span / front matter | Instruction |
| `voice`, `as` spans; `reset` | Shared planner |

### ElevenLabs

| Control | Support |
| --- | --- |
| Text and cast voices | Yes |
| `pause`, `silence`, `break`, nonnegative `gap` | Local silence |
| `speed`, `fast`, `slow` | Parameter: 0.7–1.2×; unsupported on v4/v4 Turbo |
| `pitch` | Unsupported |
| `volume` | Unsupported |
| Standard emotions | Audio-tag instruction |
| Emotion intensity, e.g. `happy 0.8` | Text instruction |
| `whisper` / `shout` | Audio tags |
| `act`, `accent`, `dialect`, custom `emotion` | Audio-tag instruction |
| `style`, `role` | Audio-tag instruction |
| `emph` | Audio-tag instruction |
| `laugh`, `cry`, `sigh` **spans** | Audio-tag instruction |
| Standalone nonverbal events | Selected events |
| `lang` span / front matter | Language code |
| `voice`, `as` spans; `reset` | Shared planner |

### MiniMax

| Control | Support |
| --- | --- |
| Text and cast voices | Yes |
| `pause`, `silence`, `break`, nonnegative `gap` | Local silence |
| `speed`, `fast`, `slow` | Parameter: 0.5–2× |
| `pitch` | Integer −12…+12 st |
| `volume` | Parameter −60…+20 dB |
| Standard emotions | Seven mapped emotions |
| Emotion intensity, e.g. `happy 0.8` | Unsupported |
| `whisper` / `shout` | Whisper on 2.6 only; no shout |
| `act`, `accent`, `dialect`, custom `emotion` | Unsupported |
| `style`, `role` | Unsupported |
| `emph` | Unsupported |
| `laugh`, `cry`, `sigh` **spans** | Unsupported |
| Standalone nonverbal events | Selected events on 2.8 |
| `lang` span / front matter | Language boost |
| `voice`, `as` spans; `reset` | Shared planner |

### Azure

| Control | Support |
| --- | --- |
| Text and cast voices | Yes |
| `pause`, `silence`, `break`, nonnegative `gap` | Local silence |
| `speed`, `fast`, `slow` | SSML: 0.5–2× |
| `pitch` | SSML −12…about +7.01955 st |
| `volume` | SSML −60…+20 dB |
| Standard emotions | Mapped styles |
| Emotion intensity, e.g. `happy 0.8` | Unsupported |
| `whisper` / `shout` | Voice-dependent styles |
| `act`, `accent`, `dialect`, custom `emotion` | Unsupported |
| `style`, `role` | SSML; role requires style |
| `emph` | Voice-dependent emphasis |
| `laugh`, `cry`, `sigh` **spans** | Unsupported |
| Standalone nonverbal events | Unsupported |
| `lang` span / front matter | SSML locale |
| `voice`, `as` spans; `reset` | Shared planner |

### Qwen

| Control | Support |
| --- | --- |
| Text and cast voices | Yes |
| `pause`, `silence`, `break`, nonnegative `gap` | Local silence |
| `speed`, `fast`, `slow` | Parameter: 0.5–2× |
| `pitch` | Parameter −12…+12 st |
| `volume` | Parameter −60…about +6.02 dB |
| Standard emotions | Instruction |
| Emotion intensity, e.g. `happy 0.8` | Text instruction |
| `whisper` / `shout` | Instruction |
| `act`, `accent`, `dialect`, custom `emotion` | Instruction |
| `style`, `role` | Instruction |
| `emph` | Instruction |
| `laugh`, `cry`, `sigh` **spans** | Instruction |
| Standalone nonverbal events | Selected events |
| `lang` span / front matter | Language hint |
| `voice`, `as` spans; `reset` | Shared planner |

### Gemini

| Control | Support |
| --- | --- |
| Text and cast voices | Yes |
| `pause`, `silence`, `break`, nonnegative `gap` | Local silence |
| `speed`, `fast`, `slow` | Instruction for `fast`/`slow`; numeric unsupported |
| `pitch` | Unsupported |
| `volume` | Unsupported |
| Standard emotions | Instruction |
| Emotion intensity, e.g. `happy 0.8` | Text instruction |
| `whisper` / `shout` | Instruction |
| `act`, `accent`, `dialect`, custom `emotion` | Instruction |
| `style`, `role` | Instruction |
| `emph` | Instruction |
| `laugh`, `cry`, `sigh` **spans** | Instruction |
| Standalone nonverbal events | Selected events |
| `lang` span / front matter | Native language-code |
| `voice`, `as` spans; `reset` | Shared planner |

### Fish

| Control | Support |
| --- | --- |
| Text and cast voices | Yes |
| `pause`, `silence`, `break`, nonnegative `gap` | Local silence |
| `speed`, `fast`, `slow` | Parameter: 0.5–2× |
| `pitch` | Unsupported |
| `volume` | Parameter −60…+20 dB (local bound) |
| Standard emotions | Selected S2 cues |
| Emotion intensity, e.g. `happy 0.8` | Unsupported |
| `whisper` / `shout` | S2 cues |
| `act`, `accent`, `dialect`, custom `emotion` | Unsupported |
| `style`, `role` | Unsupported |
| `emph` | S2 cue |
| `laugh`, `cry`, `sigh` **spans** | Unsupported |
| Standalone nonverbal events | Selected events |
| `lang` span / front matter | Unsupported |
| `voice`, `as` spans; `reset` | Shared planner |


## Exact mappings

MiniMax emotion mappings: `happy→happy`, `sad→sad`, `angry→angry`, `fear→fearful`, `disgust→disgusted`, `surprise→surprised`, `calm→calm`. `whisper` occupies the same API field as emotion and cannot be combined with one. Supported events on `speech-2.8-hd` and `speech-2.8-turbo`:

```text
laugh chuckle cough clear-throat groan breath pant inhale exhale gasp
sniff sigh snort burp lip-smack hum hiss emm sneeze
```

Azure style mappings: `happy→cheerful`, `sad→sad`, `angry→angry`, `fear→fearful`, `excited→excited`, `calm→calm`, `tender→gentle`, `serious→serious`, `whisper→whispering`, `shout→shouting`, `crying→crying`, `neutral→neutral`. `[style "vendor-style"]` passes a style name. Only one style is allowed per segment. The selected voice must support it; Azure can ignore unsupported styles. Timbrex currently does not probe voices.

ElevenLabs events: `laugh`, `sigh`, `gasp`, `clear-throat`, `giggle`, `cough`. Qwen events: `laugh`, `sigh`, `gasp`, `clear-throat`, `giggle`, `cough`, `snort`. Event arguments and notes are unsupported, including `[laugh nervously]`; use a supported accompanying-speech span or an acting instruction instead.

`[laugh|Hello]` asks for speech while laughing. `[laugh]` requests a standalone event. Those are different capabilities. An instruction does not guarantee a precise nonverbal duration or an isolated sound.

Volume converts dB to a linear amplitude ratio with `10^(dB/20)`. MiniMax receives that ratio; Azure receives a relative percentage; Qwen receives a rounded integer scaled from its default volume of 50. Qwen pitch converts semitones to a ratio with `2^(st/12)`. Azure pitch is limited to its documented frequency ratio of 0.5–1.5, whose upper bound is `12*log2(1.5)` semitones, to avoid requesting values the service may clamp or substitute. These are parameter mappings, not measurements of rendered loudness or pitch accuracy.

## Literal text

Escaping protects Timbrex syntax, but does not disable a vendor's native markup. To avoid turning literal text into unintended instructions, ElevenLabs, Qwen, and Fish reject literal ASCII `[` or `]` in spoken text. Fish also rejects the native speaker-token prefix `<|`. ElevenLabs also rejects brackets inside acting instructions, literal `<break>` tags on older models, and `<phoneme>` tags on `eleven_flash_v2` and `eleven_turbo_v2`. Mathematical comparisons and unrelated tag-like words remain accepted.

MiniMax rejects its recognized native event strings, such as `(laughs)`, and its pause prefix `<#`. Other parentheses are passed through, but the service can interpret parenthesized Pinyin, IPA, or Jyutping as pronunciation controls; the adapter does not recognize every such form. Use Unicode alternatives such as `（ ）` and `［ ］`, or another provider, when those marks must remain text.

Eleven v4 also interprets [slash-delimited IPA](https://elevenlabs.io/docs/overview/capabilities/text-to-speech/best-practices#ipa-with-eleven-v4) as pronunciation guidance. Slash text is passed through; the adapter does not broadly reject slashes or guarantee that every native pronunciation form stays literal.

Azure XML-escapes text and attributes. OpenAI sends speech and instructions in separate fields. This prevents compiler-generated structural injection; it does not guarantee that a generative model will pronounce every symbol literally.

## Reserved, not implemented

All real adapters currently reject pronunciation wrappers (`ipa`, `pinyin`, `sub`, `say`, and related tags), audio files/effects/music, `overlap`, `cut`, `duration`, `rate`, `oral`, `emovec`, markers, voice conversion, and raw fences. There is no automatic fallback or silent dropping of unsupported controls. Named tag arguments are parsed but have no adapter mappings yet.

The stub ignores performance semantics and writes an 8 kHz test tone. It cannot validate provider capabilities or speech quality, and it does not insert local pauses. `check` recognizes the full vocabulary and validates pause/silence/break/gap duration kinds used by the shared planner. It does not select an adapter, validate adapter-specific ranges, or make requests.

## Gemini TTS

The `gemini-3.8-flash-tts` adapter supports text, prebuilt cast voices, shared voice/as/reset scopes and local pauses. Standard emotions (including numeric intensity), `act`, `style`, `role`, `accent`, `dialect`, custom `emotion`, `whisper`, `shout`, `emph`, and laugh/cry/sigh spans become **instructions** in `speechMetadata.style`. `fast` and `slow` request faster/slower speech without an exact multiplier. Numeric speed, pitch and volume are **unsupported**. `lang` is a native language-code parameter with the restricted list documented in setup.

Standalone events map `laugh→<laugh>`, `sigh→<sigh>`, `cough→<cough>`, `breath→<breath>`, and `gasp→<gasp>`; arguments/notes and other events are unsupported. Native cues request a performance, not an exact isolated sound. Literal ASCII `<` and `>` in text are conservatively rejected to avoid native cue interpretation; use Unicode alternatives when they must remain text. Instructions stay separate from the transcript. The reserved unsupported features above also apply.

## Fish Audio

The `s2.1-pro` adapter supports text, explicit voice reference IDs, shared voice/as/reset scopes and local pauses. Numeric speed is a **parameter** (0.5–2×); `fast`/`slow` use 1.2×/0.8×. Volume is a **dB parameter** within the adapter's −60…+20 bound. Pitch, explicit language/accents, acting/style/role instructions, numeric emotion intensity and laugh/cry/sigh spans are **unsupported**.

Native S2 cues map `happy→[happy]`, `sad→[sad]`, `angry→[angry]`, `fear→[scared]`, `surprise→[surprised]`, `disgust→[disgusted]`, `calm→[calm]`, `excited→[excited]`, `whisper→[whispering]`, `shout→[shouting]`, and `emph→[emphasis]`. Other emotions are unsupported. Events map `laugh→[laughing]`, `chuckle→[chuckling]`, `sigh→[sighing]`, `groan→[groaning]`, `pant→[panting]`, `gasp→[gasping]`, `yawn→[yawning]`, `snore→[snoring]`, and `clear-throat→[clear throat]`. Cue/event arguments and notes are unsupported. These are requested performances, not guaranteed isolated sounds or measured acoustic values.

Literal ASCII `[`/`]` and the native speaker-token prefix `<|` are rejected to prevent accidental controls. Ordinary parentheses remain text; the legacy S1 parenthesized cue syntax is not generated. The reserved unsupported features above also apply.
