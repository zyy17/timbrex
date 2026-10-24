# Gemini TTS

[简体中文](readme.zh-cn.md) · [Timbrex setup](../../providers.md) · [Supported controls](../../capabilities.md)

Developer quick reference, reviewed **2026-10-24**. This describes the vendor offering, not full Timbrex support; see the links above for the current adapter scope.

## Models and fit

| Model | Best fit | Languages |
| --- | --- | --- |
| `gemini-3.8-flash-tts` | Expressive narration, demanding dialogue and regional pronunciation | 130 in the model comparison |
| `gemini-3.8-flash-lite-tts` | Lower-cost, high-volume read-aloud and voice-agent speech | 101 in the model comparison |

Both accept text and return audio, share the 3.8 request schema, and allow **8,192 input tokens / 16,384 output tokens**. Automatic language detection is broader than the explicit language-code selector. Context caching, thinking and tools are not supported by these TTS models. Sources: [Flash](https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash-tts), [Flash-Lite](https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash-lite-tts).

## Speech capabilities

- **Delivery:** turn-level natural-language directions for emotion, pace, pitch, volume, whispering and emphasis. These are performance instructions, not calibrated multipliers, semitones or decibels.
- **Vocal events:** inline human sounds such as laughter, crying, coughing, breathing, throat-clearing and yawning; short/long pauses. Non-vocal sound effects are not the intended use.
- **Dialogue:** up to two prebuilt speakers in one request, with per-turn speaker/style metadata; native backchannels and overlap cues. Custom-voice dialogue uses one synthesis request per speaker turn.
- **Voice identity:** select the right regional voice or design a persona; sustained accent, age and identity changes should not be treated as ordinary delivery tweaks.
- **Pronunciation and timing:** the reviewed TTS documents do not establish a phoneme/SSML lexicon interface or guaranteed word-level alignment. Generic transcription timestamps are not evidence of TTS alignment support.

See the [speech guide](https://ai.google.dev/gemini-api/docs/speech-generation) and [GenerateContent guide](https://ai.google.dev/gemini-api/docs/generate-content/speech-generation).

## Voices and adjacent products

- **Catalog:** 30 featured voices, including Kore and Puck, plus hundreds in the searchable Extended Voice Library. [Catalog](https://ai.google.dev/gemini-api/docs/generate-content/speech-generation#voice-options)
- **Voice design:** creates a reusable persona from a text description, including a preview. [Voice design](https://ai.google.dev/gemini-api/docs/voice-design?hl=en)
- **Voice replication:** requires a clean 10–30-second reference and a separate consent recording from the same adult. Stored voices use `voice_...`; stateless replication uses `voicekey_...`. Stored voices share a 200-per-project limit and expire one year after last use; stateless keys expire after seven days. [Voice replication](https://ai.google.dev/gemini-api/docs/voice-replication), [retention](https://ai.google.dev/gemini-api/docs/speech-generation#custom-voice-limits-and-ttl)
- **Live API:** a separate stateful, bidirectional voice/vision conversation service with interruption and tool support. It is not the text-recitation API or merely TTS streaming. Google Cloud Text-to-Speech is also a separate API/product surface. [Live API](https://ai.google.dev/gemini-api/docs/live-api)

Creating/managing a voice is a separate workflow from synthesizing with an existing voice ID. The reviewed pricing page does not establish a separate design/replication creation fee; do not assume creation is free.

## Delivery and limits

Unary output defaults to **24 kHz mono 16-bit PCM in WAV**. Streaming defaults to headerless PCM chunks. Documented alternatives include raw PCM, G.711 mu-law/A-law and sample-rate selection. MP3/Opus appear in the generic schema but are not confirmed by the model-specific output guide. Both Interactions and GenerateContent are documented API options. [Audio formats](https://ai.google.dev/gemini-api/docs/speech-generation#audio-output-formats), [API reference](https://ai.google.dev/api/generate-content)

Request/token quotas depend on the project, model and usage tier; check active limits in AI Studio rather than assuming a fixed RPM. [Rate limits](https://ai.google.dev/gemini-api/docs/rate-limits)

## Pricing

Gemini Developer API public paid-tier prices in USD per **1 million tokens**, text input / audio output, checked **2026-10-24** (not Vertex AI pricing):

| Tier | Input, either 3.8 model | Flash audio | Flash-Lite audio |
| --- | ---: | ---: | ---: |
| Standard | $0.50 | $9.00 | $6.00 |
| Batch | $0.25 | $4.50 | $3.00 |
| Flex | $0.25 | $4.50 | $3.00 |
| Priority | $0.90 | $16.20 | $10.80 |

These 3.8 rates apply **through 2026-12-31**; every listed rate doubles on **2027-01-01**. Audio uses 25 tokens/second. Standard output alone therefore costs approximately **$0.81/hour Flash / $0.54/hour Flash-Lite** today, excluding input. Free-tier access has separate quotas/data-use terms.

Older models remain listed: `gemini-3.1-flash-tts-preview` and `gemini-2.5-pro-preview-tts` cost $1/$20; `gemini-2.5-flash-preview-tts` costs $0.50/$10 at Standard. Their Batch rates are half. They use older prompting/output conventions, not the 3.8 schema. [Official prices](https://ai.google.dev/gemini-api/docs/pricing)

[Batch](https://ai.google.dev/gemini-api/docs/batch-api) is asynchronous, targeting completion within 24 hours. [Flex](https://ai.google.dev/gemini-api/docs/flex-inference) trades latency/availability for price; [Priority](https://ai.google.dev/gemini-api/docs/priority-inference) prioritizes latency/reliability at a premium. These modes are not equivalent to submitting ordinary requests concurrently.
