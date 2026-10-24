# Authoring reference

Use this for writing and repairing scripts. For a chosen provider, also read [running.md](running.md); registered vocabulary is not a promise of synthesis support.

## Turn a brief into speech

Keep language natural for the audience. Give characters distinct intentions and phrasing rather than decorating every line. Express subtext as a short playable direction such as `[act "trying to sound reassuring"|It's all right.]`, not a biography. Character names are only display metadata.

Use punctuation for ordinary phrasing and explicit pauses for important beats. There is no implicit gap between turns. Exact runtime depends on generated speech; give only an estimate until rendered. Each change of control, voice, or pause can create a synthesis segment, so excessive tags can fragment delivery and increase requests.

Put non-spoken staging on separate `//` lines. A comment ends the paragraph: repeat the speaker prefix afterward. Preserve intended meaning when handling user-supplied dialogue, URLs, brackets, and stage directions.

## File and speakers

Front matter starts at the beginning of the file between column-zero `---` lines:

```tbx
---
timbrex: 1
strict_tags: true
narrator: guide
cast:
  guide: {name: Narrator}
  lin: {name: 林}
  chen: {name: 陈}
---
guide: 门铃响了。
lin: 你来了。 [pause 300ms] [whisper|进来吧。]
chen: [act "hesitant, then relieved"|我还以为你不会等我。]
```

- IDs match `[A-Za-z][A-Za-z0-9_.-]*` and are case-sensitive. Only declared IDs at column zero start turns. `林:` is not an alias for `lin:`.
- A declared `id:` starts a new turn even without a blank line. Ordinary following lines continue that turn.
- A blank line, standalone comment, scene line, or raw fence ends the paragraph. Ordinary text afterward is narration, not the previous speaker. Repeat the prefix to resume them.
- Unprefixed narration uses `narrator`, or the front matter's `narrator` ID. Include that ID in `cast` if it needs a voice mapping.
- `cast: auto` recognizes ASCII speaker prefixes without declarations. It does not choose distinct voices. Prefer explicit cast for stable scripts.
- With no matching cast ID, `note: hello` is spoken text. Escape a possible prefix as `lin\: hello` when it should be literal.
- Allowed front matter keys: `timbrex`, `provider`, `model`, `voice`, `lang`, `strict_tags`, `narrator`, `cast`. Cast entries allow `name` and `voices`. Unknown fields are errors; do not add `title`, `description`, `speed`, credentials, or `concurrency` here.

## Tags, scope, and reset

- Point tag: `[happy]` or `[pause 300ms]`. Span: `[happy|Welcome.]` or `[act "softly"|Welcome.]`. Spans nest and close with `]`, never `[/happy]`.
- A span may cross non-empty lines but must close within its paragraph. Full-width `［］` and `｜` are ordinary text, not syntax.
- State point tags inside a turn affect its remainder. A line consisting only of point tags is a scene line: state persists into later turns. Scene events happen once between turns.
- Each new turn starts from scene state, not the previous turn's state. Closing a nested span restores its enclosing state.
- `[reset]` in a turn restores scene state, even inside a nested span. It does not mean “restore the enclosing span” or “make neutral.” On a scene line it clears scene state.
- `[reset speed]` in a turn restores scene speed, or removes turn speed when none exists. On a scene line it clears scene speed.
- Related controls replace one another: `speed`/`fast`/`slow`, emotions, `whisper`/`shout`, and `voice`/`as`. Different features coexist only when the adapter can represent them.
- `[as lin|words]` uses a known speaker's mapping. Declare the ID in `cast`; mentioning an ID only in `as` with `cast: auto` does not declare it. `[voice "vendor-id"|words]` selects a vendor voice directly. Both require content; `check` does not resolve voices.

Common controls:

- Emotions include `neutral happy sad angry fear surprise calm excited tender serious curious sarcastic nervous tired confused hopeful crying`.
- Delivery includes `act whisper shout emph speed fast slow pitch volume style role accent dialect`, subject to adapter support.
- `[laugh]` requests a standalone event; `[laugh|That's a good one.]` requests accompanying speech. These are different capabilities. Do not assume every event supports speech or accepts a note.
- `[lang en-US|Hello.]` is a language span, not a translation command. Language hints are provider-specific.

Tag names are case-insensitive and normalize `_` to `-`; IDs/values do not. Without strict mode, unknown tags become implicit acting. Prefer explicit `act` so typos do not become performance instructions.

## Values and timing

- Use unquoted numeric controls: `[speed 1.1]`, `[speed 110%]`, `[pitch +2st]`, `[volume -3dB]`. `110%` means 1.1 times normal speed, not a 110% increase.
- Put a digit before decimals: `0.5s`, not `.5s`. Scientific notation and quoted numbers are not numeric controls.
- `pause`, `silence`, and `break` accept `0ms` through `1min`, in `ms`, `s`, or `min`. `gap` uses the same durations but belongs at the beginning of a turn or on a scene line.
- Bare pause defaults to 250 ms; `short`/`medium`/`long` mean 250 ms/500 ms/1 s. Write zero as `0ms`, not `0`. `1m` and `1s250ms` are not supported durations.
- Named arguments such as `intensity=0.5` parse but have no adapter mappings. Do not use them as vendor controls. Numeric emotion intensity is only an approximate text instruction on certain adapters.
- The checker validates pauses but not all other argument types/ranges. Review the selected adapter's bounds separately.

## Literal text and escaping

- Outside quoted arguments, `\[` and `\]` escape brackets; `[[` and `]]` are alternatives. Prefer backslashes when nesting because consecutive `]` inside multiple open tags close those tags.
- Escape literal `|`, `\`, and a speaker-prefix `:` as `\|`, `\\`, and `\:`.
- Quoted arguments support `\"` and `\\`: `[act "say \"fine\" reluctantly"|I'm fine.]`.
- Quotes in content are spoken punctuation: `[whisper|She said "hello".]`.
- Only a line whose first non-space characters are `//` is a comment. Inline `//`, `#`, HTML comments, and parentheses are spoken text.
- A raw fence needs an ASCII identifier after its opening three backticks. It stops Timbrex parsing, but every real adapter rejects it. It is not an escape hatch for SSML/vendor markup.
- Escaping does not neutralize vendor markup. ElevenLabs/Qwen/Fish reject literal ASCII brackets; Gemini rejects literal ASCII angle brackets. MiniMax reserves some parenthesized cues and `<#`. Check exact restrictions before synthesis. Offer Unicode punctuation or another provider without silently changing text.

## Unsupported production requests

No current adapter implements pronunciation wrappers (`ipa`, `pinyin`, `say`, `sub`, and related tags), audio/music mixing, overlapping speech, exact segment duration, or raw-provider passthrough. The language has no loops/conditionals. Do not generate these as working features.

For effects or overlap, preserve intent in non-spoken production notes and explain the external editing step or ask about an alternative. For pronunciation, offer a user-approved text rewrite or documented instruction-capable provider without promising phonetic accuracy.

Baseline: [DSL specification](https://github.com/zyy17/timbrex/blob/47d478de8b5ba0ec4134d9745bde0151a73155c5/docs/spec.md) and [vocabulary registry](https://github.com/zyy17/timbrex/blob/47d478de8b5ba0ec4134d9745bde0151a73155c5/internal/vocab/vocab.go). Use matching documentation for another revision.
