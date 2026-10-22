# Design rationale

[简体中文](reflection.zh-cn.md) · [Tutorial](tutorial.md)

Timbrex records dialogue, speakers, and performance in one script. Authors do not write per-vendor request bodies.

## Language size

Plain text needs no configuration. One tag form covers events, state, and spans. `[whisper|Come closer.]` places the direction next to its words. `[pause 300ms]` places timing at the event. The parser does not treat full-width punctuation, parentheses, or a display name as syntax. Explicit rules make scripts reviewable and errors reproducible.

The grammar is frozen. New capabilities add vocabulary or a provider mapping. Dataset loops and application logic belong outside the speech body. A formatter may rewrite whitespace; it must not invent meaning.

## Intent and audio

A tag states the author's request. An adapter decides whether its API can represent it. A numeric speed parameter, an SSML style, and a natural-language instruction are distinct mechanisms. They do not imply identical audio across vendors.

Unsupported controls fail. They are not dropped. A future fallback must expose any approximation. Capability tables describe tested adapter behavior and model restrictions, not vendor marketing or an unverified ranking.

## Pipeline

The pipeline is lexer → parser/AST → checker → lower/IR → provider. The lexer sees text and tokens. The checker sees the vocabulary. Providers see IR. Cast and configuration become IR fields. Adapters do not reread YAML. Typed arguments stay structured during lowering.

Cobra executes commands. `internal/config` loads file and environment settings with koanf. The CLI applies script and flag overrides. Adapters receive resolved options. There is no global configuration singleton.

Planning resolves scene, turn, and nested span state before network I/O. Each adapter builds its requests before rendering begins. The shared renderer concatenates matching PCM and local silence. It does not mix, overlap, or resample. Sequential calls add latency and possible discontinuities. Those limits belong in the usage docs.

## Evidence

Official API references justify request fields. Offline tests cover payloads, scope restoration, malformed responses, cancellation, and output preservation. Live tests must separately establish account compatibility and acoustic results. Documentation must state which evidence exists.

English and Chinese pages describe the same product. Examples must parse. Commands must exist. Local links must resolve. Planned features must be marked as plans.

## Boundaries

Position-based scope is semantic: moving `[happy]` onto its own line changes a turn instruction into scene state. A formatter must preserve that distinction. The default still accepts implicit free-form acting. Version-controlled scripts should use strict mode and explicit `act`. Changing that default requires an explicit compatibility decision.

Language validation and synthesis support are separate. `check` confirms syntax and vocabulary, while the adapter validates the requested performance. A future provider-aware check should reuse the adapter's request preparation without credentials or network access, not duplicate its capability rules in the CLI.

The shared segment planner is a baseline implementation, not a mandatory provider interface. Adapters receive the full IR and may eventually preserve whole utterances through native SSML or inline controls. Do not freeze a split at every control span. That would add latency, cost, and audible discontinuities on providers that can express a whole sentence.

One synthesis invocation selects one provider and one model. Cast mappings select voices within it. Cross-provider dialogue, calibrated emotion intensity, inaudible segment boundaries, and reproducible acoustic output are not current guarantees. Establish those guarantees with measured behavior before adding more vocabulary or abstraction.
