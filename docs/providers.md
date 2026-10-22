# Provider setup

[简体中文](providers.zh-cn.md) · [Supported controls](capabilities.md)

Timbrex uses synchronous HTTP APIs directly. The adapters are based on the official references below, reviewed on **2026-10-22**. Offline tests verify request and response contracts; live account compatibility and acoustic quality have not been verified. Credentials must belong to the endpoint's region and have access to the chosen model and voice.

## Configuration

Settings are loaded centrally from YAML and environment variables; see [configuration](configuration.md). Prefer credentials in your shell or secret manager. Timbrex does **not** load `.env` files. Never put keys in `.tbx` files or commit them. The table below describes environment-based setup; equivalent `providers.<id>` file fields are also supported.

| Provider ID | Required environment | Default model | Default voice |
| --- | --- | --- | --- |
| `openai` | `OPENAI_API_KEY` | `gpt-4o-mini-tts` | `coral` |
| `elevenlabs` | `ELEVENLABS_API_KEY`, plus a voice selection | `eleven_v3` | None; set `ELEVENLABS_VOICE_ID`, `--voice`, front matter, or cast mapping |
| `minimax` | `MINIMAX_API_KEY` | `speech-2.8-hd` | `English_expressive_narrator` |
| `azure` | `AZURE_SPEECH_KEY`, and region or endpoint | Selected by voice | `en-US-JennyNeural` |
| `qwen` | `DASHSCOPE_API_KEY`, `QWEN_ENDPOINT` | `qwen-audio-3.1-tts-flash` | `longanhuan_v3.1` |
| `stub` | None | None | None; test tone |

Optional environment defaults:

| Provider | Model | Voice | Endpoint override |
| --- | --- | --- | --- |
| OpenAI | `OPENAI_MODEL` | `OPENAI_VOICE` | `OPENAI_BASE_URL` (base URL including `/v1`) |
| ElevenLabs | `ELEVENLABS_MODEL` | `ELEVENLABS_VOICE_ID` | `ELEVENLABS_BASE_URL` (base URL including `/v1`) |
| MiniMax | `MINIMAX_MODEL` | `MINIMAX_VOICE_ID` | `MINIMAX_BASE_URL` (base URL including `/v1`) |
| Azure | — | `AZURE_SPEECH_VOICE` | `AZURE_SPEECH_ENDPOINT` (full request URL), otherwise `AZURE_SPEECH_REGION` |
| Qwen | `QWEN_MODEL` | `QWEN_VOICE` | `QWEN_ENDPOINT` (required full request URL) |

`--model` and `--voice` override front matter, which overrides environment settings, which override YAML configuration and built-in defaults. Cast voice mappings override the default voice. A `[voice "vendor-id"|text]` span selects a voice directly; `[as lin|text]` uses Lin's mapping, falling back to the default if no mapping is available. Changing provider does not translate a model or voice ID belonging to another vendor.

Endpoint overrides are intended for your chosen regional service or a trusted compatible gateway. API credentials are sent to that endpoint. Use HTTPS for remote services. HTTP is accepted for local testing. Requests do not follow redirects.

## OpenAI

```sh
# OPENAI_API_KEY must already be set.
./timbrex synth examples/expressive.tbx --provider openai --voice coral -o openai.wav
```

Calls `POST https://api.openai.com/v1/audio/speech` with bearer authentication and `response_format: pcm`. Acting is sent separately in `instructions`; spoken text stays in `input`. Input and instructions are each limited to 4096 characters by the adapter. `tts-1` and `tts-1-hd` can synthesize plain text with speed control, but reject acting and language instructions.

References: [Create speech](https://developers.openai.com/api/reference/resources/audio/subresources/speech/methods/create), [TTS guide](https://developers.openai.com/api/docs/guides/text-to-speech). Follow the provider's disclosure requirements when publishing AI-generated speech.

## ElevenLabs

```sh
# ELEVENLABS_API_KEY and ELEVENLABS_VOICE_ID must already be set.
./timbrex synth examples/expressive.tbx --provider elevenlabs -o elevenlabs.wav
```

Calls `POST https://api.elevenlabs.io/v1/text-to-speech/{voice_id}?output_format=pcm_24000` with `xi-api-key`. The adapter emits native audio tags for `eleven_v3`, `eleven_v4`, and `eleven_v4_turbo`. Other model IDs accept plain speech and speed but reject acting tags and nonverbal events. A conservative 5000-character adapter limit includes generated tags. Language hints use the primary language subtag; `eleven_multilingual_v2` rejects explicit `lang` because it does not accept `language_code`.

References: [Create speech](https://elevenlabs.io/docs/api-reference/text-to-speech/convert), [audio tags](https://elevenlabs.io/docs/help-center/product/core-capabilities/text-to-speech/how-do-audio-tags-work-with-eleven-v3-and-v4), [speed control](https://elevenlabs.io/docs/eleven-creative/playground/text-to-speech). Audio tags describe desired performance; they are not guaranteed sound effects.

## MiniMax

```sh
# MINIMAX_API_KEY must already be set.
./timbrex synth examples/expressive.tbx --provider minimax -o minimax.wav
```

Calls `POST https://api.minimax.io/v1/t2a_v2` with bearer authentication, non-streaming hex output, and PCM audio settings. It checks `base_resp.status_code`, completion status, and hex decoding. Each text segment must be shorter than 10000 characters.

The default `speech-2.8-hd` supports the mapped nonverbal events. Whisper is supported by the adapter only with `speech-2.6-hd` or `speech-2.6-turbo`. Whisper cannot be combined with another emotion in the same segment. Pitch requires whole semitones. Language hints currently map `zh en ja ko fr de es ru pt it ar hi th id vi yue`; `zh-HK` and `zh-MO` map to Cantonese. Other hints fail locally. This subset describes the adapter, not the vendor's full language coverage.

Reference: [Synchronous speech synthesis API](https://platform.minimax.io/docs/api-reference/speech-t2a-http).

## Azure Speech

```sh
# AZURE_SPEECH_KEY must already be set.
export AZURE_SPEECH_REGION=eastus
./timbrex synth examples/expressive.tbx --provider azure --voice en-US-JennyNeural -o azure.wav
```

Calls `POST https://{region}.tts.speech.microsoft.com/cognitiveservices/v1` with the subscription key and SSML, requesting `raw-24khz-16bit-mono-pcm`. Text and attributes are XML-escaped. The voice name chooses the model; specifying `--model` or front matter `model:` is an error.

Style, role, and emphasis support depend on the selected voice. Azure may ignore a style that a voice does not support; Timbrex does not yet query voice capabilities. The adapter allows one style per segment and rejects conflicting styles. `[role Boy]` needs a style in the same scope. Without explicit `lang`, the locale is inferred from the first two components of the voice name.

References: [REST API](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/rest-text-to-speech), [voice, style, role, and prosody](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice).

## Qwen

```sh
# DASHSCOPE_API_KEY must already be set; replace the workspace placeholder.
export QWEN_ENDPOINT='https://YOUR_WORKSPACE_ID.ap-southeast-1.maas.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer'
./timbrex synth examples/expressive.tbx --provider qwen -o qwen.wav
```

This adapter targets **Qwen-Audio-TTS 3.0/3.1**, not the older Qwen3 multimodal-generation API or locally hosted Qwen3 weights. It supports the `qwen-audio-3.0-tts-flash`, `qwen-audio-3.0-tts-plus`, `qwen-audio-3.1-tts-flash`, and `qwen-audio-3.1-tts-plus` model IDs. Choose a compatible voice when changing models.

The workspace endpoint is required. Singapore uses `ap-southeast-1`; Beijing uses `cn-beijing` in the same URL shape. The request uses `input.instruction` (singular), PCM format, and a 24000 Hz sample rate. A completed response provides a signed audio URL, which Timbrex downloads **without** the API key. Language hints accept `zh en fr de ja ko ru pt th id vi es it ms fil ar`; locale suffixes are removed.

References: [HTTP API](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api), [models and usage restrictions](https://www.alibabacloud.com/help/en/model-studio/non-realtime-tts-user-guide).

## Failures and limits

Each HTTP request has a two-minute timeout and a 64 MiB response limit. The final PCM output is limited to 1 GiB. All speech segments are validated before requests start. Runtime failures report the provider and segment number. HTTP errors include status codes; MiniMax/Qwen API errors include vendor error codes. Response bodies and signed URL query strings are omitted from diagnostics.

Audio responses must contain raw 24 kHz mono S16LE PCM. Conflicting media types, recognizable audio containers, and empty or odd-length PCM are rejected. Raw PCM has no metadata, so these checks cannot independently verify its sample rate, channel count, or sound quality; those rely on the requested vendor format.

The renderer uses a temporary file beside the destination and renames it only after success. Cancellation or failure removes temporary output and preserves an existing destination. The parent directory must exist. Earlier requests may already have incurred charges; there is no retry, fallback, cache, or resumable rendering.

For troubleshooting, check the selected model, voice, region, and key permissions against the linked vendor docs. A successful offline test establishes the implemented contract, not service availability or speech quality. Long input should be split into paragraphs; Timbrex does not automatically chunk it to vendor token limits.
