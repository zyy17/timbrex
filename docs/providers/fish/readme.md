# Fish Audio

[简体中文](readme.zh-cn.md) · [Timbrex setup](../../providers.md) · [Supported controls](../../capabilities.md)

Developer quick reference, reviewed **2026-10-24**. This describes the vendor offering, not full Timbrex support; see the links above for the current adapter scope.

## At a glance

Fish Audio emphasizes natural-language voice direction, multilingual speech and reusable or reference-based voices. Its direct TTS API supports both single-speaker speech and native dialogue.

| Model | Main distinction |
| --- | --- |
| `s2.1-pro` | Recommended production S2 model; 83 languages and open-ended expressive cues |
| `s2.1-pro-free` | Same model quality for development and smaller workloads; fair use, no TTFA or DPA guarantees |
| `s2-pro` | Previous S2 generation; 80+ languages |
| `drama-3` | Performance direction, sounds, pauses, paired delivery tags and phoneme control |
| `s1` | Legacy model, deprecated; retirement scheduled for December 31, 2026 |

S1 uses parentheses for cues, unlike the square-bracket directions used by S2. After retirement, S1 requests are documented to run on S2.1-Pro. An omitted or unknown model header also falls back to S2.1-Pro, so model names matter. See [models](https://docs.fish.audio/developer-guide/models-pricing/models-overview) and [model selection](https://docs.fish.audio/developer-guide/models-pricing/choosing-a-model).

## Capabilities

- **Voice direction:** emotions, delivery styles and human sounds. S2 accepts natural descriptions, not just a fixed emotion list. Drama 3 adds word-scoped paired tags and experimental language switching.
- **Voice choice:** saved library/custom voices or inline audio plus a transcript for zero-shot synthesis. Creating and managing saved clones is a separate workflow.
- **Dialogue:** S2 and Drama 3 can generate multiple speakers together rather than concatenate independent clips.
- **Pronunciation and readability:** phoneme cues, inline or managed pronunciation dictionaries, number/date/unit normalization. The language option affects normalization and alignment, not the spoken voice or accent.
- **Generation controls:** speed (0.5–2×), dB volume, loudness normalization, sampling variation, latency modes, chunking and previous-chunk conditioning. Optional quality-guard support depends on the backend.
- **Delivery:** HTTP audio, bidirectional WebSockets and streaming word-alignment snapshots; WAV/PCM, MP3 and Opus. WAV/PCM is mono 16-bit with rates up to 44.1 kHz; Opus uses 48 kHz.

See [TTS API](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech), [emotion control](https://docs.fish.audio/developer-guide/core-features/emotions), [Drama 3](https://docs.fish.audio/features/text-to-speech/drama-3), [streaming](https://docs.fish.audio/api-reference/endpoint/websocket/tts-live) and [timestamps](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech-stream-with-timestamps).

Voice design, saved-voice lifecycle and speech recognition are adjacent products, not extra TTS options. Performance cues are model interpretations rather than guarantees. The API schema does not publish a universal maximum text length; a client-side cap should not be presented as a vendor limit.

## Pricing and caveats

[API pricing](https://docs.fish.audio/developer-guide/models-pricing/pricing-and-rate-limits) is in **USD per million UTF-8 input bytes**, with no API subscription minimum:

- S2.1-Pro, S2-Pro and legacy S1: **$15**.
- S2.1-Pro Free: **$0**, subject to fair use.
- Drama 3: **$50 list price**; **$10 launch rate until November 12, 2026, 17:00 UTC**.

Bytes are not characters: non-ASCII scripts can cost more for the same character count. Avoid comparing these rates directly with a character-priced vendor.

Paid-account concurrency tiers are 5, 15 and 50 requests, unlocked by total prepaid amounts below $100, at least $100 and at least $1,000; Enterprise limits are negotiated. Free development has no guaranteed first-audio latency. Regional prices, numeric free-tier quotas and account-specific terms were not verified.
