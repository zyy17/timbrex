# OpenAI speech

[简体中文](readme.zh-cn.md) · [Timbrex setup](../../providers.md) · [Supported controls](../../capabilities.md)

Developer quick reference, reviewed **2026-10-24**. This describes the vendor offering, not full Timbrex support; see the links above for the current adapter scope.

## Choose the right API

OpenAI's Speech API is a compact option for prompt-directed narration and multilingual speech. Its `tts-1`, `tts-1-hd` and `gpt-4o-mini-tts` models are deprecated and scheduled to shut down on **2027-01-06**. OpenAI recommends `gpt-realtime-2.1-mini` through the Realtime API; it cannot replace a model name in an existing Speech API request. [Lifecycle](https://developers.openai.com/api/docs/deprecations#2026-10-01-text-to-speech-models)

## Features and tradeoffs

- **Expressive narration:** GPT-4o Mini TTS accepts delivery instructions for accent, emotion, intonation, pace and whispering. `tts-1` favors latency; `tts-1-hd` favors quality, but neither accepts these instructions. Voices are optimized for English, with multilingual speech available. [Speech guide](https://developers.openai.com/api/docs/guides/text-to-speech)
- **Voice choice:** the Speech API lists 13 built-in voices and accepts approved custom voice references. Voice availability depends on the model. [Speech reference](https://developers.openai.com/api/reference/resources/audio/subresources/speech/methods/create)
- **Audio delivery:** MP3, Opus, AAC, FLAC, WAV and raw PCM; audio can arrive incrementally. Raw PCM is 24 kHz, signed 16-bit little-endian. SSE is also available, except on `tts-1`/`tts-1-hd`. [Formats](https://developers.openai.com/api/docs/guides/text-to-speech#supported-output-formats), [streaming options](https://developers.openai.com/api/reference/resources/audio/subresources/speech/methods/create)
- **Custom voices:** recording-based creation requires eligibility, a consent recording and a matching sample. Prompt-designed voices are a separate capability usable only in Live, not Speech or Realtime. [Custom voices](https://developers.openai.com/api/docs/guides/custom-voices), [voice creation](https://developers.openai.com/api/reference/resources/audio/subresources/voices/methods/create)
- **Interactive applications:** Realtime provides conversational sessions, interruptions and tools. Treat this as a different workflow from rendering a fixed script. [Realtime](https://developers.openai.com/api/docs/guides/realtime)

## Limits worth knowing

Speech requests allow up to 4096 characters of text and 4096 characters of instructions; Mini TTS separately has a 2000-input-token limit. Speed is 0.25–4×. The Speech schema has no native SSML, phoneme dictionary or numeric pitch/volume controls. Prompt directions should not be treated as exact acoustic constraints. [API limits](https://developers.openai.com/api/reference/resources/audio/subresources/speech/methods/create), [model limit](https://developers.openai.com/api/docs/models/gpt-4o-mini-tts)

Clearly disclose AI-generated voices to listeners. [Disclosure](https://developers.openai.com/api/docs/guides/text-to-speech)

## Pricing

Public USD API rates reviewed 2026-10-24. Token and character units are not interchangeable.

| Model | Price | Source |
| --- | --- | --- |
| GPT-4o Mini TTS | $0.60 / 1M input text tokens + $12 / 1M output audio tokens | [Model pricing](https://developers.openai.com/api/docs/models/gpt-4o-mini-tts) |
| TTS-1 | $15 / 1M characters | [Model pricing](https://developers.openai.com/api/docs/models/tts-1) |
| TTS-1 HD | $30 / 1M characters | [Model pricing](https://developers.openai.com/api/docs/models/tts-1-hd) |
| Realtime 2.1 Mini | Per 1M tokens: text input/output $0.60/$2.40; audio input/output $10/$20 | [Realtime pricing](https://developers.openai.com/api/docs/pricing#realtime-and-audio-generation-models) |

Realtime uses session accounting rather than a fixed per-character TTS tariff. Check current account terms and the linked rates before estimating production costs. Prompt-designed voice creation is free, but auditioning and using it incurs GPT-Live session charges. A separate recording-based voice creation price was not verified. [Voice pricing details](https://developers.openai.com/api/docs/guides/custom-voices#create-and-save-the-voice)
