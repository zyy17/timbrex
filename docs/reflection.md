# Design rationale

[简体中文](reflection.zh-cn.md) · [DSL Specification v1](spec.md) · [Contributing](../contributing.md)

Timbrex keeps dialogue, speakers, and performance directions in a script that can be reviewed alongside code. The main design choices concern how much belongs in the language, how intent maps to different providers, and what the renderer can preserve.

## Keep the language small

Plain text is a valid script. Point tags place events or state changes next to speech; spans attach a direction to specific words. This lets an author add control without writing a provider request body. English ASCII identifiers make the structural syntax unambiguous while spoken text and acting instructions can use any language.

The cost is that position affects meaning. Moving `[happy]` from a speaker's line onto its own line changes a turn instruction into scene state. A future formatter must preserve that distinction. Implicit acting tags make drafting convenient but can turn a typo into an instruction, so strict mode and explicit `act` are useful for maintained scripts. The language may evolve during development; changes to these rules need examples and a review of their effect on existing scripts.

## Separate author intent from provider mechanisms

A numeric speed parameter, an SSML style, and a natural-language instruction can all express a request about delivery, with different precision. A shared tag does not make their results acoustically equivalent. The adapter must document how it represents the request and reject controls it cannot support. Any future approximation should be visible to the author.

The compiler separates parsing, vocabulary checking, and lowering to IR. Adapters receive structured arguments and the complete IR rather than reparsing the script. This keeps vendor-specific rules out of the parser and allows adapters to use different rendering strategies. A synthesis invocation uses one provider. Where the provider supports a model setting, that setting applies to the whole invocation; cast mappings choose voices within the provider.

## Preserve ordered rendering

The shared planner resolves scene, turn, and span state into segments. HTTP adapters prepare all segment requests before network I/O, so an unsupported control late in a script can fail before any synthesis request is sent. The renderer then concatenates compatible PCM segments and local silence into a WAV.

Generation remains serial by default. Optional [bounded concurrency](providers.md#concurrent-generation) overlaps independent requests while one writer preserves segment order. The reorder window is tied to concurrency, so a slow early segment applies backpressure instead of retaining the whole script's audio. This trades some potential throughput for bounded audio retention and straightforward failure cleanup; it does not promise live speedups or repeatable acoustics.

This approach is easy to inspect and test, but separate calls add latency and can produce audible boundaries. It cannot mix or overlap voices, resample audio, or guarantee continuity between segments. Providers receive the full IR, so a future adapter could preserve an utterance through native SSML or inline controls instead of splitting at each control change. That choice should follow measured provider behavior.

## Keep validation claims specific

`check` answers whether a script follows the implemented language rules; provider preparation answers whether the requested controls can be mapped to requests. A future provider-aware check could reuse that preparation without credentials or network calls.

Official API documentation supports request-field choices. Offline tests verify serialization, scope restoration, error handling, cancellation, and output preservation. They cannot establish current account access or voice quality. Those require separate live validation. The [capability reference](capabilities.md) records current adapter behavior; cross-provider consistency, calibrated emotion intensity, and repeatable acoustic output remain open problems.
