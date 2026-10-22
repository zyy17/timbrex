# Design rationale

[简体中文](reflection.zh-cn.md)

Timbrex exists to make expressive speech easy to write and read. A script should show the dialogue, its speakers, and its performance without requiring the author to learn every vendor's request format.

## Keep the language small

Plain text works without configuration. One tag form covers events, state, and spans. `[whisper|Come closer.]` keeps the direction next to its words; `[pause 300ms]` puts timing where it happens. The parser does not guess that full-width punctuation, parentheses, or a display name are syntax. Explicit rules make scripts reviewable and errors reproducible.

The grammar is frozen. New capabilities should normally add vocabulary or a provider mapping. Dataset loops and application logic belong outside the speech body. A future formatter may make source prettier, but must not invent its meaning.

## Express intent without promising identical audio

A tag says what the author wants. An adapter decides whether its API can represent it. A numeric speed parameter, an SSML style, and a natural-language instruction are distinct mechanisms. None establishes that two services will sound the same.

The current implementation reports unsupported controls rather than silently discarding them. A future fallback system must expose any approximation. Capability declarations should describe tested adapter behavior and model restrictions, not vendor marketing or an unverified quality ranking.

## Make the compiler followable

The pipeline is lexer → parser/AST → checker → lower/IR → provider. The lexer sees text and tokens, the checker sees the vocabulary, and the providers see IR. Cast and configuration become explicit IR fields rather than forcing adapters to read YAML. Typed arguments stay structured during lowering.

Provider planning resolves scene, turn, and nested span state before network I/O. Each adapter builds its requests before rendering begins. A small shared renderer joins matching PCM segments and local silence. It does not try to be a general audio workstation. Sequential calls trade some continuity and latency for a straightforward implementation; these limitations belong in the usage docs.

## Keep evidence close to claims

Official API references justify request fields. Offline tests verify payloads, scope restoration, malformed responses, cancellation, and output preservation. Live tests must separately establish account compatibility and acoustic results. Documentation should say which evidence exists.

English and Chinese docs describe the same product. Examples should parse, commands should exist, links should resolve locally, and planned features should be identified as plans. Contribution rules and focused Conventional Commits help reviewers evaluate those claims.

## Current design boundaries

Position-based scope is compact but meaningful: moving `[happy]` onto its own line changes a turn instruction into scene state. A formatter must preserve that distinction. The current default still permits implicit free-form acting; strict mode with explicit `act` is the recommended form for version-controlled scripts. Changing that default requires an explicit compatibility decision.

Language validation and synthesis support are separate. `check` confirms syntax and vocabulary, while the adapter validates the requested performance. A future provider-aware check should reuse the adapter's request preparation without credentials or network access, not duplicate its capability rules in the CLI.

The shared segment planner is a baseline implementation, not a mandatory provider interface. Adapters receive the full IR and may eventually preserve whole utterances through native SSML or inline controls. Keep this option open: splitting every control span permanently would impose avoidable latency, cost, and audible discontinuities on providers that can express the whole sentence.

One synthesis invocation selects one provider and one model. Cast mappings select voices within it. Cross-provider dialogue, calibrated emotion intensity, seamless segment boundaries, and reproducible acoustic output are not current guarantees. Establish those guarantees with measured behavior before adding more vocabulary or abstraction.
