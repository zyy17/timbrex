# Timbrex design notes

> 2026-10-20. Input: the v0.2 draft, the v0.1 draft, [CAPABILITIES.md](CAPABILITIES.md), and [MARKET_SURVEY.md](MARKET_SURVEY.md).
> The capability model and the fallback chain in v0.2 were right. The surface syntax and the language boundary were not.
> v0.3 is one bracket, one rule, and one open registry. Control flow leaves the DSL. The grammar freezes. Growth is vocabulary and capability files.
> The normative text is [SPEC.md](SPEC.md). Chinese text: [REFLECTION.zh-CN.md](REFLECTION.zh-CN.md).

## 1. Who the language is for

| Fact | Consequence |
| --- | --- |
| Most scripts will be written by a model and edited by a person | The syntax must be a shape the model has already seen: ElevenLabs v3 `[tag]`, Markdown, and `name: line`. The rules must fit in one prompt. Common mistakes must be machine-fixable. |
| Many authors type with an IME | Do not depend on characters that an IME rewrites. The grammar itself stays ASCII. Cleanup is `timbrex fix`, not a second grammar. |
| Providers already use `[]`, `()`, and `<>`, and they disagree | The DSL occupies one pair. The compiler escapes the others per provider. |
| Provider features move quickly, from fixed enums toward free text, song, effects, and realtime | The grammar must freeze. Change lives in the vocabulary and in caps. The IR must be open. |
| Performance text, configuration, and data-generation logic are different | Each gets the format that fits it. |

Easy means few rules, and few exceptions per rule. Extensible means a new capability touches the outermost layer: ideally one YAML file, not the grammar, the core, or the IR types.

## 2. What was wrong with the mixed syntax

- Too many forms: point tags, XML spans, line directives, speaker decorations, narration markers, two comment styles, interpolation, front matter, and several raw-passthrough paths.
- One meaning had many spellings. Emotion could be written six ways. Scope depended on configuration, so the same script meant different things under different files.
- Angle brackets look like SSML, Cartesia, Gemini events, and other vendor marks. Closers mismatch. A bare `<` in prose needs escaping.
- Loops, picks, and conditionals inside a line turn the script into a template language. Performance text and sampling strategy have different authors and different lifetimes. Data generation also wants cartesian products, quotas, and external files, which a dialogue DSL will not finish. Interpolation of raw data into markup is an injection bug. v0.2 had no escape rule.
- "Unregistered means free-form acting" changes meaning when a later vocabulary registers that name. The fix is a locked vocabulary, a stable-meaning rule for new names, and an edit-distance warning.
- A speaker prefix without a cast rule treats ordinary colons as speakers.
- A closed IR struct (`Speed`, `Pitch`, `Dialect`, …) forces every new feature through every adapter.

What v0.2 got right, and v0.3 keeps: intent tag to IR to a caps-driven fallback chain; free-form acting as a first-class feature; namespace passthrough; caps files with `verified` and `deprecated`; external process plugins; manifest fields; compliance metadata; `voices.lock`.

## 3. Directions that were compared

XML/SSML is expressive and unpleasant to write, and vendors extend it with private namespaces. JSON is a good IR and a poor place to drop a laugh in the middle of a sentence. An embedded Go builder is a library API, not an authoring language. Speech Markdown is light but closed and SSML-only. Screenplay formats solve who speaks, not how. Game dialogue languages add a second command syntax. Typst shows that one call shape plus a content block covers the rest. ElevenLabs, Fish, and Inworld made `[natural language]` the de facto standard. Mustache shows that a template should only substitute, with escaping by default.

v0.3 is the screenplay line, the `[tag]` mark, and one content form `[name args|content]`. YAML and JSON hold config, data, and IR. Go is the library.

Precedents kept: namespaces from SSML, an open vocabulary and many compile targets from the failure of closed SSML compilers, near-compatibility with ElevenLabs v3, plain text as a valid file, one markup syntax, a fenced escape hatch, `{{x}}` escaped and `{{{x}}}` raw, and a strict parser with an explicit fixer.

## 4. Decisions

### 4.1 Three layers

| Layer | Carrier | Written by |
| --- | --- | --- |
| Performance | `.tbx` body | Model or writer |
| Config | YAML front matter, `timbrex.yaml`, or `cast.yaml` | Production |
| Data | `*.tbxset.yaml` or generated `items.jsonl` | Data work |

The body has no control flow. Dataset templates only substitute, and they escape by default.

### 4.2 One bracket

Every mark is `[head args…]` or `[head args…|content]`. There is no `<>`, `@`, `$`, or `#`. Scope is position only: mid-line, piped content, or a line of tags. Full-width lookalikes are not grammar. That reversal is recorded in the v0.3.1 note below.

### 4.3 Open vocabulary

Unregistered ASCII brackets are free-form acting, which keeps the ElevenLabs shape easy. Packages are version-locked. A new name must still mean the right thing as free text, or it stays in a namespace. `timbrex check` suggests a registered name at edit distance 2 or less.

### 4.4 Open IR

A node is a feature id, an argument map, a note, and a span. There is no struct of style fields. A new capability is a new string.

### 4.5 Five levels

Standard tag, package tag, namespace tag, namespace raw text or parameters, raw fence. None of them changes the grammar.

### 4.6 Three version axes

Grammar (`timbrex: 1`, meant to stay), vocabulary semver, and the caps verification date. A provider can ship weekly without touching the syntax.

## 5. Tradeoffs

| Choice | Cost | Mitigation |
| --- | --- | --- |
| Piped spans instead of BBCode | Long spans are clumsier. Models know BBCode | Long spans are state plus position. `timbrex fix` rewrites `[/x]`. A pipe does not collide with a state tag |
| Unknown means acting | Typos become acting. A future registration could change meaning | Lockfile, stable names, edit-distance warnings, `--strict-tags` |
| Speakers must be in cast | A cast block is required for dialogue | `cast: auto`, and check hints |
| Control flow is outside the script | A batch needs a tbxset file | The file is short. `timbrex new dataset` can sketch it |
| Brackets match ElevenLabs, Fish, Inworld, and Qwen | A reader may think the text is already native | That is intentional. `[el:…]` is the raw form when it must be verbatim |
| Parenthetical stage directions stay text | Writers must opt in | `parens: act` and `timbrex fix --parens` |
| Frozen grammar | Some future phrase may feel awkward | New things are tags, parameters, or fence info. A real grammar change is `timbrex: 2` plus `timbrex migrate` |

## 6. Still open

See [SPEC.md](SPEC.md) section 15. The sharp ones were whether BBCode `[/x]` is also accepted (no: it is a fix, not grammar), whether free-form acting is translated to the provider's preferred language, and whether front matter stays (yes, beside `timbrex.yaml`).

## Note (v0.3.1, 2026-10-20)

The DSL is English ASCII for tags, namespaces, parameters, keywords, YAML keys, emotions, and the standard library. Lines may be any language. Full-width equivalence and non-English aliases were dropped. `timbrex fix` rewrites them. `timbrex check` lints them. Speaker ids are ASCII. Display names live on `name:`. Free-form acting may be any language, English preferred, handled by `act_translate`. The span form stayed `[x|content]` because a BBCode opener is identical to a state tag, which would break "position decides scope."

## Note (v0.3.4, 2026-10-21)

The project is Timbrex. The CLI is `timbrex`. Scripts use `.tbx` (alias `.timbrex`). Batches use `*.tbxset.yaml`. Config is `timbrex.yaml`. The lock is `timbrex.lock`. The grammar key is `timbrex: 1`. The module is `github.com/zyy17/timbrex`.
