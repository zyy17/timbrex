# Timbrex specification v0.3.4

Grammar version: `timbrex: 1`.

> One pair of brackets, one rule, one open registry. **The DSL itself is English ASCII. Spoken lines may be any language.**
> A `.tbx` file (`.timbrex` is an alias) becomes performed speech, a multi-speaker scene, or a training row, and compiles to any TTS provider.
> Background: [REFLECTION.md](REFLECTION.md). Capability notes: [CAPABILITIES.md](CAPABILITIES.md), [MARKET_SURVEY.md](MARKET_SURVEY.md).
> Chinese text: [SPEC.zh-CN.md](SPEC.zh-CN.md).

## Changelog

### v0.3.4 (2026-10-21)

The project name is Timbrex, from timbre plus the suffix -x.

| Thing | Name |
| --- | --- |
| CLI, Go module, config, lock, grammar key, JSON field, adapter prefix, Go API | `timbrex` |
| Script extension | `.tbx` (preferred). `.timbrex` is an equivalent alias. `timbrex fmt` writes `.tbx` |
| Batch dataset | `*.tbxset.yaml` |
| Module | `github.com/zyy17/timbrex` |

Diagnostic prefixes stay `V` / `W` / `I`. `voices.lock.yaml` is unchanged. `timbrex migrate` rewrites older file names and the grammar key.

### v0.3.2 (2026-10-20)

- tbxset defaults to `strict_tags: true`. An unregistered `[…]` is error **V041**, with a near-name suggestion. It is not silent free-form acting.
- Explicit free-form acting is `[act "…"]`, or `[act "…"|content]` for a span. `act` is a registered standard tag with one string argument. It adds no syntax.
- A single script still treats an unregistered `[…]` as implicit free-form acting, plus warning W040 when the name is close to a registered tag. `strict_tags: true`, `--strict-tags`, and `--no-strict-tags` override the mode.
- `timbrex fix --explicit-acts` rewrites implicit `[x]` to `[act "x"]`. `timbrex fmt` quotes `[act holding back tears]` as `[act "holding back tears"]`.

### v0.3.1 (2026-10-20)

- Tag names, namespaces, parameter names, keywords, YAML keys, the emotion set, and the standard vocabulary are English ASCII.
- Full-width brackets and punctuation are ordinary characters everywhere. `timbrex fix` rewrites them. `timbrex check` warns (section 3.5).
- A speaker id is an ASCII identifier declared in `cast`. Display names live on `name:`. Only an ASCII `:` introduces a speaker.
- Free-form acting may be any language. English is recommended. Providers record a language preference. `act_translate` controls translation.
- Quoted strings may be any Unicode.
- `[x|content]` is the only span form. BBCode `[x]…[/x]` is rewritten by `timbrex fix`.

### v0.3.0 (2026-10-20)

First frozen design: one tag syntax, scope by position, an open IR, five levels of capability, tbxset datasets, and three version axes.

## 0. Principles

| Principle | What it means |
| --- | --- |
| P1 Easy | Plain text is valid. Authors learn one mark, `[ ]`. Scope comes only from position. The shape matches ElevenLabs v3, so a model already writes it. One language for the vocabulary, so aliases do not collide. |
| P2 Extensible | The grammar is frozen. Growth is vocabulary packages, namespaces, capability files, and adapters. IR feature ids are strings. |
| P3 Coverage | Five levels: standard tag, package tag, namespace tag, namespace raw text or parameters, raw fence (section 8). |
| P4 Reproducible | Parsing is deterministic. Forgiving rewrites happen only in an explicit `timbrex fix`. Vocabulary and caps versions are locked. |

**Language rule.** Code (marks, identifiers, config keys, enums) is English ASCII. Content (lines, narration, free-form acting, quoted strings, display names) may be any language.

| Layer | File | Holds |
| --- | --- | --- |
| Performance | `*.tbx` body | Who speaks, what they say, how, and when. **No control flow.** |
| Config | YAML front matter, or `timbrex.yaml` / `cast.yaml` | Speakers and voices, provider, compliance, output, request parameters |
| Data generation | `*.tbxset.yaml`, or `items.jsonl` from any program | Sources, sampling, seed, splits, templates with only `{{var}}` substitution |

## 1. Core rules

```
1. Plain text is a valid script. Markup is one form only:
     [name args…]  or  [name args…|content]        (ASCII [ ] | only)
2. Position decides scope:
     [x] inside a line     → event happens here / state lasts to end of this turn
     [x|content]           → applies to content only
     [x] alone on a line   → whole scene from here, until overridden or [reset]
3. name = registered ASCII tag, or ns:tag. Args: positional or key=value;
   values: 0.7  500ms  1.5s  120%  +2st  -3dB  word  "any unicode string".
   Unknown trailing words = note.
   Free-form acting: explicit [act "any language"] (always valid);
   implicit [anything unregistered] is allowed in single scripts only (tbxset = error V041).
4. Turns: `speaker_id: text` (speaker_id is an ASCII id declared in cast).
   Unprefixed paragraphs = narrator. A blank line ends a turn.
5. `//` at line start = comment. ``` fence = raw passthrough (e.g. ```ssml azure).
   Literal brackets: [[ ]] or \[ \] \|. Other brackets in text — () <> 【】 （） — are plain
   text, escaped per provider by the compiler.
```

Minimal example:

```tbx
---
cast:
  lin:  { name: Lin }
  chen: { name: Chen }
---
lin: [laugh] Really? [pause 500ms] Great!
chen: [sigh] [whisper|I already knew.]
```

## 2. Lexical grammar

### 2.1 Preprocess

Normalize newlines to `\n`. Strip a leading BOM. There is no full-width folding. Full-width characters stay as text. `timbrex fix` may rewrite them (section 3.5).

### 2.2 EBNF

```ebnf
script        = [ front_matter ] , { block } ;
front_matter  = "---" , NL , yaml_text , "---" , NL ;          (* file start only *)

block         = blank_line | comment_line | fence | scene_line | turn | narration ;
blank_line    = { WS } , NL ;
comment_line  = { WS } , "//" , { ANY } , NL ;
fence         = "```" , fence_info , NL , { raw_line } , "```" , NL ;
fence_info    = ident , { WS , ( ident | kv ) } ;

scene_line    = { WS } , tag , { { WS } , tag } , { WS } , NL ;
turn          = speaker_id , ":" , content , NL , { cont_line } ;
narration     = content , NL , { cont_line } ;
cont_line     = content , NL ;                                 (* non-empty, no speaker, not a comment *)

speaker_id    = ident ;                                        (* must be declared in cast, section 3.4 *)

content       = { text_run | tag | escape } ;
text_run      = TEXT_CHAR , { TEXT_CHAR } ;
escape        = "[[" | "]]" | "\" , ( "[" | "]" | "|" | "\" | ":" ) ;

tag           = "[" , { WS } , tag_body , { WS } , [ "|" , content ] , "]" ;
tag_body      = structured | free_act ;
structured    = head , { WS , arg } ;
head          = [ ns , ":" ] , ident ;
arg           = ident , "=" , value | value ;
value         = number | duration | percent | semitone | decibel | quoted | word ;
number        = [ "+" | "-" ] , DIGIT , { DIGIT } , [ "." , DIGIT , { DIGIT } ] ;
duration      = number , ( "ms" | "s" | "min" ) ;
percent       = number , "%" ;
semitone      = number , "st" ;
decibel       = number , "dB" ;
quoted        = '"' , { UNICODE - '"' | '\"' } , '"' ;
word          = WORD_CHAR , { WORD_CHAR } ;
free_act      = ACT_CHAR , { ACT_CHAR } ;

ident         = ASCII_LETTER , { ASCII_LETTER | DIGIT | "_" | "-" | "." } ;
```

### 2.3 How a tag is classified

1. *body* is the text after `[` and before the first unescaped `|` or `]`.
2. If the first word is `ns:ident` and `ns` is a registered namespace, it is a namespace tag. A registered `ident` uses that TagSpec. Otherwise the text after `:` is **namespace raw** and goes to that provider unchanged.
3. Otherwise, if the first word is an ASCII ident registered in the loaded vocabulary (case-insensitive, `-` and `_` equivalent), match arguments: `key=value` first, then positional values by type. Remaining words are a `note` when the tag says `note: allow`, and an error otherwise.
4. Otherwise the tag is **implicit free-form acting**. `note` is the raw body, in any language. In strict mode (tbxset default, or `strict_tags: true`) this is error V041, with registered names at edit distance 2 or less, and the hint to write `[act "…"]`. Namespace raw text and `[act "…"]` are not affected.
5. `|content` is parsed recursively and must close inside the same paragraph.

### 2.4 Why spans are `[x|content]`

| | `[x|content]` | BBCode `[x]…[/x]` |
| --- | --- | --- |
| Versus a state tag | A pipe means a span | The opening tag looks exactly like a state tag. A missing close silently changes meaning |
| Crossing | Brackets cannot cross | `[a][b][/a][/b]` needs an extra error |
| Wrong closer | Does not exist | Common |
| Locality | Closes inside the paragraph | Matching can be far away |
| Long span | Heavier | Natural |
| Model familiarity | High | Higher |

**`[x|content]` is the only normative span.** A long region is a state tag plus position scope (rest of the turn, or a line of tags for the scene). `timbrex fix` rewrites a paired `[x]…[/x]` in the same paragraph to `[x|…]`. `timbrex check` reports `[/x]` as V013.

## 3. Meaning

### 3.1 Document

- A **paragraph** is consecutive lines split by a blank line.
- A **turn** is a paragraph whose first line is `speaker_id:`. Later lines without a speaker stay in that turn.
- A **narration** paragraph has no speaker. The speaker is `narrator` unless `narrator:` points elsewhere. A one-voice script is all narration and needs no config.
- A **scene line** contains only tags. It is not spoken. It changes scene state or places a timeline event (pause, effect, music).

### 3.2 Tag kinds

The registry declares the kind. Authors do not memorize it.

| kind | Mid-line `[x]` | `[x|…]` | Alone on a line `[x]` |
| --- | --- | --- | --- |
| `event` (laugh, pause, sfx) | Happens here | Accompanying sense, if TagSpec defines it. Otherwise an error | Happens on the timeline between turns |
| `state` (emotion, speed, whisper) | From here to the end of the turn | Applies to the content | Until the scene ends, it is overridden, or `[reset]` |
| `wrap` (pinyin, say, sub, lang) | Error: content is required | Applies to the content | Error |
| `turn` (overlap, gap) | Only at the start of a turn | — | Applies to the next turn |

Inner values of the same feature replace outer ones. Different features stack. `[reset]` clears turn state. `[reset speed]` clears that feature. A scene-line `[reset]` clears scene state.

### 3.3 Free-form acting

- Explicit, valid in every mode: `[act "holding back tears"]`, `[act "…"|content]`. `act` is a standard state tag. Its parameter is a string. Bare words are a note (`[act holding back tears]`). `timbrex fmt` adds quotes.
- Implicit, single scripts only: an unregistered bracket is free-form acting in any language. A near miss warns W040.
- Strict mode (tbxset default, or `strict_tags: true`, or `--strict-tags`): the implicit form is V041. A typo such as `[laff]` must not silently label thousands of rows. Explicit acts can be searched.
- English is recommended. Non-ASCII acting gets info I050, not an error.
- `act_translate: off | lexicon | llm` (default `lexicon`). When the acting language and the provider preference differ, `lexicon` uses a built-in phrase table and otherwise sends the original text with an info. `llm` calls a configured translator and caches the result in `timbrex.lock`. `off` always sends the original text.
- If the provider accepts free text, send it (after translation). Otherwise map to the nearest standard emotion or nonverbal, then to an instruction, then drop with a warning.
- Precedence: CLI `--strict-tags` / `--no-strict-tags`, then tbxset or config `strict_tags`, then the mode default (tbxset true, single script false).

### 3.4 Speakers

- The prefix is `speaker_id:` with an ASCII identifier, an ASCII colon, and an id declared in cast. Anything else is ordinary text, so a display name or a note with a colon is not a speaker.
- The display name is `lin: { name: Lin }`. It is used in manifests, reports, captions, and instruction-style providers.
- `cast: auto` treats every leading `ident:` as a speaker. `timbrex check` lists what it recognized.
- A literal colon is `lin\: not a speaker`.

### 3.5 Input cleanup and `timbrex fix`

The grammar accepts ASCII only. Full-width marks and non-English tag words are a tool concern, not a grammar concern.

| Author wrote | Grammar | `timbrex check` | `timbrex fix` |
| --- | --- | --- | --- |
| Full-width brackets around a word | Plain text | warning W101 | The ASCII tag, when the lexicon knows the word |
| A full-width pipe inside those brackets | Plain text | W101 | `[tag|content]` |
| An unregistered word that matches a standard tag | Free-form acting | warning W102 | The English tag |
| A non-English free-form phrase | Free-form acting | info I050 | Unchanged, unless `--translate-acts` |
| A display name used as a prefix | Narration | warning W103 | The declared speaker id |
| A full-width pipe or quote inside a tag | Part of the body, so free-form acting | W104 | ASCII `|` and `"` |
| Parenthetical stage directions | Plain text | info I060 | `[tag]` only with `--parens` |
| `[whisper]…[/whisper]` | `[/whisper]` is not a closer | error V013 | `[whisper|…]` |
| `[thoughtfully]` inside tbxset | Strict error | error V041 | `--explicit-acts` writes `[act "thoughtfully"]` |

The non-English to English map is tool data (`fix/zh.yaml`), not part of the vocabulary. `timbrex fix --check` exits non-zero in CI when a rewrite is available.

### 3.6 Arguments

- Emotion intensity is 0–1, default 0.6: `[happy 0.8]`.
- Speed is a ratio or a word: `[speed 1.2]`, `[fast]`. Pitch is `+2st` or `high` / `low`. Volume is `+3dB` or `loud` / `soft`.
- Degree words (`slightly`, `very`) are notes. An adapter may map them to numbers.

## 4. Standard vocabulary

`std` loads by default. `timbrex tags --search laugh` queries it.

### 4.1 Core

| tag | kind | example |
| --- | --- | --- |
| `pause` | event | `[pause 500ms]`, `[pause short]`, `[pause]` |
| 24 emotions: `neutral happy sad angry fear surprise disgust calm excited tender serious curious sarcastic nervous tired bored contempt confused proud embarrassed hopeful disappointed panicked crying` | state | `[sad 0.5]`, `[happy|Great]` |
| `emotion` | state | `[emotion wistful 0.4]` |
| `act` | state | `[act "holding back tears"]` |
| `speed`, `fast`, `slow` | state | `[slow|Nobody spoke]` |
| `pitch`, `volume` | state | `[pitch +2st]`, `[volume -3dB|…]` |
| `whisper`, `shout` | state | `[whisper|I came to see you]` |
| `emph` | state | `[emph|Come in!]` |
| `laugh`, `sigh`, `breath`, `cough`, `gasp` | event | `[laugh]`, `[laugh nervously]` |
| `pinyin`, `ipa`, `sub`, `say` | wrap | `[pinyin chong2|重]`, `[sub "World Health Organization"|WHO]`, `[say digits|2026]` |
| `sfx` | event | `[sfx door_creak.wav]`, `[sfx "distant thunder" dur=3s]` |
| `reset` | state | `[reset]` |

### 4.2 Extensions inside std

- Prosody: `silence`, `duration`, `break`, `rate`.
- Style: `style`, `role`, `accent`, `oral`, `dialect`, `emovec`. Dialect values are English ids. A display string uses quotes: `dialect "…"`.
- Nonverbal set: breath family, laugh family, cry family, cough and mouth sounds, and fillers `hmm uh um emm`. Arguments include a duration, `intensity=`, and a note. Content means accompanying speech: `[laugh|while laughing]`.
- Pronunciation and text: `arpabet`, `jyutping`, `romaji`, `spell`, `verbatim`, `latex`, `lang`, `mark`.
- Several speakers and time: `overlap`, `gap`, `cut`, `as`, `voice`.
- Audio: `sfx`, `audio`, `bgm`, `ambience`.
- Voice conversion: `vc`.

### 4.3 Governance

1. A new name must mean the same thing when it falls back to free text. Otherwise it belongs in a namespace.
2. Names are append-only. A rename is a new name plus `deprecated` on the old one. Aliases are English too.
3. A new tag has `since:`. A script locked to an older vocabulary still treats the new name as free-form acting and may suggest an upgrade.

## 5. Configuration

YAML, in front matter or `timbrex.yaml`. Keys are English.

```yaml
---
timbrex: 1
use: [std@1, film@0.3]
lang: zh-CN
provider: azure
fallback: [qwen, minimax]
dialogue: auto             # native | stitch | auto
narrator: narrator
act_translate: lexicon     # off | lexicon | llm
strict_tags: false         # single script default; tbxset default is true
parens: text               # text | act
seed: 42
cast:
  narrator: { name: Narrator, voices: { azure: zh-CN-YunjianNeural } }
  lin:
    name: Lin
    desc: "a lively woman in her twenties"
    gender: f
    age: 25
    voices: { azure: zh-CN-XiaoxiaoNeural, fish: { reference_id: "<id>" } }
    clone:  { ref: voices/lin.wav, consent: consents/lin.pdf, cache: true }
    design: { prompt: "a gentle voice, slightly nasal", seed: 7 }
    remix:  { base: clone, prompt: "more mature", strength: medium }
    mix:    [ { voice: "<a>", weight: 0.7 }, { voice: "<b>", weight: 0.3 } ]
    match:  { gender: f, age: 20-30, style: lively }
lexicon: { GPT: "G P T" }
params:
  el: { stability: 0.3 }
compliance: { require_consent_for_clone: true, watermark: provider, disclose_ai: true }
output: { format: wav, sample_rate: 24000, mode: merge }
---
```

Voice resolution order: `voices.<provider>`, then `clone`, `design`, `remix`, `mix`, `match`, then the default voice plus a warning. `timbrex voices clone|design|remix` builds a voice and writes `voices.lock.yaml`. A clone without consent is an error. `timbrex check` tells a person which vendor checks remain. It does not bypass them.

## 6. Datasets: `*.tbxset.yaml`

```yaml
timbrex: 1
seed: 20261020
provider: minimax
fallback: [qwen]
config: cast.yaml
strict_tags: true
inputs:
  row: csv(sentences.csv)
grid: [row, k]
vars:
  k:   range(3)
  spk: pick(cast)
  emo: weighted({neutral: 2, happy: 1, sad: 1, angry: 1, surprise: 1})
  spd: uniform(0.9, 1.15, step=0.05)
  nv:  pick(["", "[laugh] ", "[sigh] ", "[breath] "])
  act: pick(["", "[act \"holding back a smile\"] "])
id: "{{row.id}}_{{k}}"
split: { train: 0.9, dev: 0.05, test: 0.05, by: spk }
template: "{{spk}}: {{{nv}}}{{{act}}}[speed {{spd}}][{{emo}} 0.7]{{row.text}}"
output: { mode: per-item, manifest: manifest.jsonl, export: [hf] }
qc: { asr_cer_max: 0.1, max_silence: 1.5s }
```

- Strict checking runs after `timbrex expand` and before synthesis, on every expanded item, including `{{{x}}}` inserts and the `timbrex` field of `items.jsonl`. Reports group by template site plus variable values.
- `{{x}}` is escaped by default, so `[` `]` `|` in data do not become tags. `{{{x}}}` inserts raw text. Templates have no conditions and no loops.
- Each variable draws from `hash(seed, item id, variable name)`. Adding a variable does not reshuffle the others.
- `timbrex expand` writes `items.jsonl`. A program in any language may write the same file. Each line is `{id, timbrex, meta}`.
- The manifest matches the earlier field list, plus `vars`, `vocab`, `caps_verified`, and `speaker_name`.

## 7. Extension shape

```
.tbx → parse (frozen grammar) → AST
     → resolve (registry / namespace / lock) → lower → open IR
     → plan (caps + fallback chain) → requests → synth → render → audio + manifest
```

### 7.1 TagSpec

A package may be YAML or Go. `name`, `namespace`, parameter names, and aliases must be ASCII idents. `describe` and quoted text inside `expand` may be any language.

```yaml
package: film
version: 0.3.0
namespace: film
tags:
  - name: door-slam
    kind: event
    feature: audio.sfx
    lower: { sfx: assets/door_slam.wav }
    describe: "a door slamming shut"
  - name: hold-back-tears
    kind: state
    expand: '[act "holding back tears, voice tight"]'
  - name: breathy
    since: 0.3.0
    kind: state
    feature: voice.breathiness
    params: [ { name: level, type: number, range: [0, 1], default: 0.5 } ]
    note: allow
    fallback: [ native, { instruction: "more breathy, {level|degree}" }, drop ]
```

### 7.2 Open IR

```go
type Node struct {
    Kind     NodeKind // Text | Event | StateSet | StateUnset | SpanBegin | SpanEnd | TurnAttr | Raw
    Feature  string   // "prosody.speed", "emotion", "nv.laugh", "act", "pass.el", ...
    Args     map[string]Value
    Note     string
    NoteLang string
    Text     string
    Src      Pos
}
```

The core understands timeline, speaker, text, and raw. Every other feature is interpreted by caps and adapters.

### 7.3 Capability files

Strategy chain: `native → param → ssml → instruction → postprocess → asset → map-nearest → drop`. A `scope: request` feature splits the request. `--strict` and `--report` keep their earlier meaning.

### 7.4 Five levels, one syntax

| Level | Writing | Portable |
| --- | --- | --- |
| 1 Standard | `[sad 0.6]`, `[laugh]` | Yes |
| 2 Package | `[door-slam]`, `[film:door-slam]` | Yes, with the package fallback |
| 3 Namespace tag | `[az:express-as style=cheerful degree=1.5|…]` | That provider |
| 4 Namespace raw or parameters | `[el:strong French accent]`, `[mm:set timbre_weights=…]`, or config `params:` | That provider |
| 5 Raw fence | `` ```ssml azure `` | That provider |

Another provider seeing levels 3–5 uses `on_other: drop | warn | act | error` (default warn and drop).

### 7.5 Adapters

A Go adapter exposes `ID`, `Aliases`, `Namespace`, `Caps`, `Voices`, `Tags`, `Plan`, and `Synthesize`. A declarative adapter is caps plus a `request:` template and no Go. An external adapter is the executable `timbrex-provider-<name>`, JSON-RPC on stdin and stdout, in any language.

## 8. Coverage

Section 8 of [SPEC.zh-CN.md](SPEC.zh-CN.md) lists each vendor control and the Timbrex spelling that reaches it. The rule is the five levels above: portable tags first, a namespace tag or fence only when the portable tag cannot say the thing.

## 9. Versions

| Axis | Id | Rule |
| --- | --- | --- |
| Grammar | `timbrex: 1` | Changes only when the EBNF changes, with `timbrex migrate` |
| Vocabulary | `use: [std@1]` locked in `timbrex.lock` | Semver. Append-only meaning. A rename is a new name plus a deprecated alias |
| Capabilities | caps `verified` / `deprecated` / `sunset` | Ships with the adapter. `timbrex caps --probe` measures a live API |

`timbrex.lock` records vocabulary versions, caps versions, the voice lock, and the `act_translate=llm` cache.

## 10. Extensions that do not change the grammar

- A `music` package can add `sing` and an `abc` fence. Other providers fall back to an instruction, which on ElevenLabs is `[sings]`.
- A new provider parameter starts as namespace raw text, then a checked namespace tag, then a standard tag. Scripts locked to the older vocabulary still treat the new name as free-form acting.
- `sfx` accepts a file or a description. A spatial package can add `pos`. A provider that later speaks it natively only adds a caps line.
- `timbrex serve --ws` parses and synthesizes a paragraph at a time. A realtime package may add `[await user]`, which offline compilation drops with a warning.

## 11. Tools

| Tool | Role |
| --- | --- |
| `timbrex prompt` | A model card: the rules, the English vocabulary, and what the chosen provider supports |
| `timbrex check` | Grammar, unknown tags, speakers, fallback preview, compliance, and the section 3.5 lints |
| `timbrex fix` | Full-width marks, non-English tag words, `[x]…[/x]`, angle brackets, display-name prefixes, `--parens`, `--translate-acts`, `--explicit-acts` |
| `timbrex fmt` | Whitespace, argument order, aliases to canonical names |
| `timbrex expand` / `compile` / `synth` / `diff` / `caps` / `voices` / `tags` / `migrate` | Dataset rows, audio, and locks |
| LSP, MCP, HTTP, Go library | Completion and hover. `timbrex.Parse`, `timbrex.Builder` |

```
scene.tbx:12:9  error[V012]   unclosed span
scene.tbx:13:5  error[V013]   [/whisper] is not valid; spans use [whisper|…]
scene.tbx:18:14 warning[W040] unknown [laff] used as free-form acting; did you mean [laugh]?
emo.tbxset.yaml:template:21 error[V041] unregistered tag [laff] (strict_tags); did you mean [laugh]?
                              for free-form acting write [act "laff"]
```

## 12. Shape of a scene

```tbx
[bgm rain.wav vol=-22dB loop]
It rained all night. [pause 600ms] When the door opened, [slow|nobody spoke].
[sfx door_creak.wav]
lin: [surprise 0.6] Chen? [pause 300ms] Why are you… [breath] here so late.
chen: [sad 0.5][sigh] It's nothing. [whisper|I wanted to see you.]
lin: [tender 0.6] Come in. It's cold.
chen: No, I should go. I was just about to [cut]
lin: [overlap -200ms][angry 0.7][emph|Come in!]
[bgm stop fade=3s]
```

`[thoughtfully]` is implicit free-form acting. It is legal in a single script. A tbxset template must write `[act "thoughtfully"]` or `timbrex expand` reports V041.

## 13. Migration into v0.3.1

| Older form | v0.3.1 |
| --- | --- |
| `<x args>text</x>` | `[x args|text]` |
| `speaker(emotion intensity):` | `id: [emotion 0.x]` |
| A narration marker prefix | An unprefixed paragraph |
| Non-English tag words | English names |
| Full-width brackets, pipes, colons, quotes | ASCII `[]` `|` `:` `"` |
| A display name as the speaker prefix | `id: { name: … }` plus `id:` |
| Directive overlap or gap | `[overlap …]` or `[gap …]` at the start of the turn |
| Background-music directive | Scene line `[bgm …]` |
| An interrupt span | `[cut]` at the break |
| Provider and parameter directives | Config `dialogue:` and `params:`, or `[ns:set …]` |
| A raw SSML element | `` ```ssml azure `` |
| Loops, picks, and `${x}` | `*.tbxset.yaml` and `{{x}}` |
| `#` and HTML comments | `//` |

## 14. Implementation stages

| Stage | Contents |
| --- | --- |
| M0 | Frozen-grammar parser, ASCII registry checks, std, `check`, golden tests. `fmt`, `fix`, and `prompt` follow. |
| M1 | Open IR, caps, plan and fallback, basic render. Providers: azure, elevenlabs, minimax, gemini, qwen |
| M2 | tbxset including strict tags and V041, manifest, cache, concurrency, fallback, voices.lock, act_translate |
| M3 | More providers, declarative adapters, external process adapters, local models |
| M4 | Native dialogue, LSP, MCP or HTTP, realtime `serve --ws`, `caps --probe` |

M0 in this repository parses and checks the grammar, lowers to a small IR, synthesizes through a stub provider, and ships a TextMate grammar. It does not call a vendor API.

## 15. Open questions

1. Should the default of `act_translate` stay `lexicon`, or move to `llm` with a lockfile cache?
2. Strict tbxset is decided: default strict, explicit `[act "…"]`.
3. Cast declaration stays required. `cast: auto` is the opt-in.
4. Front matter stays, alongside `timbrex.yaml`.
5. Is 24 emotions enough, and should valence and arousal be standard `emovec` axes?
6. Items marked uncertain in [CAPABILITIES.md](CAPABILITIES.md) and the language-preference table need a live check.
