# ElevenLabs

[简体中文](readme.zh-cn.md) · [Timbrex setup](../../providers.md) · [Supported controls](../../capabilities.md)

Developer quick reference, reviewed **2026-10-24**. This covers the vendor’s TTS offering only, not full Timbrex support; see the links above for the current adapter scope.

## At a glance

ElevenLabs combines expressive speech, a large voice ecosystem and several latency-oriented models. It is useful for narration, characters, multilingual content and interactive speech. Voice choice strongly affects results; emotional tags and seed values do not guarantee an identical performance.

| Models | Main distinction | Published text limit |
| --- | --- | --- |
| `eleven_v4` | Expressive speech generation, 90+ languages | 10,000 characters |
| `eleven_v4_turbo` | Realtime v4 family, approximately 100 ms median model inference | Check selected transport |
| `eleven_v3` / `eleven_v3_conversational` | Previous expressive family, 70+ languages; conversational variant targets realtime use | 5,000 for v3 |
| `eleven_multilingual_v2` | Consistent long-form delivery, 29 languages | 10,000 |
| `eleven_flash_v2_5` / `eleven_flash_v2` | Low-latency speech; 32 languages / English | 40,000 / 30,000 |

Turbo v2/v2.5 remain documented but Flash is the recommended replacement. Model latency excludes network and application overhead. Consult the [model catalog](https://elevenlabs.io/docs/overview/models) and [product limits](https://elevenlabs.io/docs/eleven-creative/playground/text-to-speech) for current availability.

## Capabilities

- **Performance:** expressive audio tags, emotions and nonverbal reactions on supported models; per-request stability, similarity and other model-dependent voice settings.
- **Pronunciation:** dictionaries, normalization controls and model-specific phoneme/IPA guidance. v4 supports slash-delimited IPA; older-model SSML support is selective.
- **Continuity:** neighboring text or request context helps connect generations. A seed provides best-effort repeatability.
- **Delivery:** complete audio, HTTP streaming, character timestamps, incremental-input WebSockets and multi-context sessions. Formats include MP3, PCM, WAV, Opus and telephony encodings; not every transport exposes every format.
- **Audio effects:** filtering, distance, room effects and panning. Stereo effects require a compatible format.
- **Dialogue:** a separate Text to Dialogue API generates multiple voices together. Its current reference recommends no more than 2,000 combined text characters for reliable requests, even where the underlying model permits more.

See [speech API](https://elevenlabs.io/docs/api-reference/text-to-speech/convert), [prompting](https://elevenlabs.io/docs/overview/capabilities/text-to-speech/best-practices), [effects](https://elevenlabs.io/docs/overview/capabilities/text-to-speech), [timing](https://elevenlabs.io/docs/api-reference/text-to-speech/convert-with-timestamps) and [dialogue](https://elevenlabs.io/docs/api-reference/text-to-dialogue/convert).

## Pricing and caveats

[ElevenAPI pricing](https://elevenlabs.io/pricing/api), in USD before taxes, currently lists:

- **$0.08 per 1,000 input characters:** v4, v3, Multilingual v2.
- **$0.04 per 1,000 input characters:** v4 Turbo, v3 Conversational, Flash/Turbo.

Pay-as-you-go and subscriptions are available. Monthly plans start at $6; included usage, promotions, concurrency, commercial permissions and format entitlements depend on the plan. Published per-minute equivalents are estimates, not the billing unit. These are API prices, not Creative-plan credit conversions.

192 kbps MP3 requires Creator+; 44.1 kHz PCM/WAV requires Pro+. Enterprise options include zero retention and [regional environments](https://elevenlabs.io/docs/overview/administration/data-residency), with separate accounts/resources and availability checks.

**Documentation caveat:** official materials disagree about v3 speed support and the REST speed range. Verify the selected model before relying on speed control; v4 does not expose the usual speed/style controls.
