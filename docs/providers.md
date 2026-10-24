# Provider setup

[简体中文](providers.zh-cn.md) · [Supported controls](capabilities.md) · [Tutorial](tutorial.md)

Timbrex uses synchronous HTTP APIs directly. The adapters are based on the official references below, reviewed on **2026-10-22**. Offline tests verify request and response contracts; live account compatibility and acoustic quality have not been verified. Credentials must belong to the endpoint's region and have access to the chosen model and voice.

## Configuration

Settings come from YAML and environment variables. See [configuration](configuration.md). Store credentials in the environment or a secret manager. Timbrex does not load `.env` files. Do not put keys in `.tbx` files or commit them. Each provider section lists its environment-based setup. Equivalent `providers.<id>` file fields are also valid.

[OpenAI](#openai) · [ElevenLabs](#elevenlabs) · [MiniMax](#minimax) · [Azure Speech](#azure-speech) · [Qwen](#qwen) · [Gemini TTS](#gemini-tts) · [Fish Audio](#fish-audio)

The `stub` provider needs no environment variables and has no model or voice; it writes a test tone.

`--model` and `--voice` override front matter, which overrides environment settings, which override YAML configuration and built-in defaults. Cast voice mappings override the default voice. A `[voice "vendor-id"|text]` span selects a voice directly. `[as lin|text]` uses Lin's mapping; a known speaker without a mapping still uses the default voice, and an unknown target is an error. Changing provider does not translate a model or voice ID belonging to another vendor.

Endpoint overrides target a regional service or a trusted compatible gateway. API credentials are sent to that endpoint. Use HTTPS for remote services. HTTP is accepted for local testing. Requests do not follow redirects.

## OpenAI

- Provider ID: `openai`.
- Environment setup: `OPENAI_API_KEY`.
- Default model: `gpt-4o-mini-tts`.
- Default voice: `coral`.
- Optional model default: `OPENAI_MODEL`.
- Optional voice default: `OPENAI_VOICE`.
- Endpoint override: `OPENAI_BASE_URL` (base URL including `/v1`).

```bash
# OPENAI_API_KEY must already be set.
./timbrex synth examples/expressive.tbx --provider openai --voice coral -o openai.wav
```

Calls `POST https://api.openai.com/v1/audio/speech` with bearer authentication and `response_format: pcm`. Acting is sent separately in `instructions`; spoken text stays in `input`. Input and instructions are each limited to 4096 characters by the adapter. `tts-1` and `tts-1-hd` can synthesize plain text with speed control, but reject acting and language instructions.

The [GPT-4o Mini TTS model](https://developers.openai.com/api/docs/models/gpt-4o-mini-tts) also has a 2000-input-token limit. The local character limit is not a token count; Timbrex does not tokenize this model's input or promise that accepted text/instructions fit its token budget. Split long segments and keep instructions concise.

OpenAI marks GPT-4o Mini TTS as deprecated. Its [lifecycle notice](https://developers.openai.com/api/docs/deprecations#2026-10-01-text-to-speech-models) schedules `tts-1`, `tts-1-hd`, and the dated GPT-4o Mini TTS snapshots for removal on 2027-01-06. The recommended Realtime replacement requires a different API implementation; changing `OPENAI_MODEL` alone does not migrate this adapter.

References: [Create speech](https://developers.openai.com/api/reference/resources/audio/subresources/speech/methods/create), [TTS guide](https://developers.openai.com/api/docs/guides/text-to-speech). Follow the provider's disclosure requirements when publishing AI-generated speech.

## ElevenLabs

- Provider ID: `elevenlabs`.
- Environment setup: `ELEVENLABS_API_KEY`, plus a voice selection.
- Default model: `eleven_v3`.
- Default voice: None; set `ELEVENLABS_VOICE_ID`, `--voice`, front matter, or cast mapping.
- Optional model default: `ELEVENLABS_MODEL`.
- Optional voice default: `ELEVENLABS_VOICE_ID`.
- Endpoint override: `ELEVENLABS_BASE_URL` (base URL including `/v1`).

```bash
# ELEVENLABS_API_KEY and ELEVENLABS_VOICE_ID must already be set.
./timbrex synth examples/expressive.tbx --provider elevenlabs -o elevenlabs.wav
```

Calls `POST https://api.elevenlabs.io/v1/text-to-speech/{voice_id}?output_format=pcm_24000` with `xi-api-key`. The adapter emits native audio tags for `eleven_v3`, `eleven_v4`, and `eleven_v4_turbo`; the two v4 models reject explicit `speed`, `fast`, and `slow` because they do not support the speed setting. Other model IDs accept plain speech and speed but reject acting tags and nonverbal events. A conservative 5000-character adapter limit includes generated tags. Language hints are lowercased and reduced to the primary language subtag; an empty primary subtag is an error. `eleven_multilingual_v2` rejects explicit `lang` because it does not accept `language_code`. The English-only `eleven_flash_v2` and `eleven_turbo_v2` models accept only `en`, including locale forms such as `en-US`. The adapter does not query other models' supported language lists; ElevenLabs can ignore a code that the selected model does not support.

Eleven v3 speed support is described inconsistently across official product and tooling documentation. Its current mapping is retained pending real-account validation; accepting the setting locally does not establish that it changes generated speech.

References: [Create speech](https://elevenlabs.io/docs/api-reference/text-to-speech/convert), [model language support](https://elevenlabs.io/docs/help-center/technical/what-models-do-you-offer-and-what-is-the-difference-between-them), [audio tags](https://elevenlabs.io/docs/help-center/product/core-capabilities/text-to-speech/how-do-audio-tags-work-with-eleven-v3-and-v4), [speed control](https://elevenlabs.io/docs/eleven-creative/playground/text-to-speech). Audio tags describe desired performance; they are not guaranteed sound effects.

## MiniMax

- Provider ID: `minimax`.
- Environment setup: `MINIMAX_API_KEY`.
- Default model: `speech-2.8-hd`.
- Default voice: `English_expressive_narrator`.
- Optional model default: `MINIMAX_MODEL`.
- Optional voice default: `MINIMAX_VOICE_ID`.
- Endpoint override: `MINIMAX_BASE_URL` (base URL including `/v1`).

```bash
# MINIMAX_API_KEY must already be set.
./timbrex synth examples/expressive.tbx --provider minimax -o minimax.wav
```

Calls `POST https://api.minimax.io/v1/t2a_v2` with bearer authentication, non-streaming hex output, and PCM audio settings. It requires an explicit successful `base_resp.status_code`, checks completion status and hex decoding, and rejects conflicting sample rate, channel count, format, or decoded byte count in optional `extra_info` audio metadata. Each text segment must be shorter than 10000 characters.

The default `speech-2.8-hd` supports the mapped nonverbal events. `[whisper]` and `[style fluent]` require `speech-2.6-hd` or `speech-2.6-turbo`. They occupy the emotion field and cannot be combined with each other or another emotion in the same segment. Other styles are unsupported. Pitch requires whole semitones.

Language hints cover all 40 documented language/dialect choices; see the [exact primary-code mappings](capabilities.md#exact-mappings). Explicit hints require one of `speech-2.8-hd`, `speech-2.8-turbo`, `speech-2.6-hd`, `speech-2.6-turbo`, `speech-02-hd`, `speech-02-turbo`, `speech-01-hd`, or `speech-01-turbo`. Persian (`fa`), Filipino (`fil`), and Tamil (`ta`) require the 2.6 or 2.8 models, including when a locale suffix is supplied. Requests without `lang` still omit `language_boost`; explicit `lang: auto` or `[lang auto|Text]` sends `"auto"`. No-language requests retain model-ID passthrough; unrecognized models with explicit language hints fail locally.

Hints are case-insensitive and accept `language[-Script][-REGION]`: a mapped primary code, optionally four script letters, optionally two region letters or three region digits. Script and region suffixes are ignored, not accent guarantees; the adapter checks their shape, not registry membership. Only the exact legacy aliases `zh-HK` and `zh-MO` select Cantonese. Other `zh` forms, including `zh-Hant-HK`, retain the Chinese mapping; use `yue` or `yue-Hant-HK` for unambiguous Cantonese. `no` maps to Norwegian, `nn` to Nynorsk. `nb` and `tl` are not inferred aliases for Norwegian and Filipino, and country codes such as `UA` or `PH` are not language hints.

Unknown codes, malformed hints such as `en-foo`, and extlang/variant/extension/private-use forms fail before any synthesis request. This deliberately tightens the old prefix-only handling, which could silently accept malformed suffixes. Use a supported primary code or the simple locale form instead.

Reference: [Synchronous speech synthesis API](https://platform.minimax.io/docs/api-reference/speech-t2a-http).

## Azure Speech

- Provider ID: `azure`.
- Environment setup: `AZURE_SPEECH_KEY`, and region or endpoint.
- Default model: Selected by voice.
- Default voice: `en-US-JennyNeural`.
- Optional model default: —.
- Optional voice default: `AZURE_SPEECH_VOICE`.
- Endpoint override: `AZURE_SPEECH_ENDPOINT` (full request URL), otherwise `AZURE_SPEECH_REGION`.

```bash
# AZURE_SPEECH_KEY must already be set.
export AZURE_SPEECH_REGION=eastus
./timbrex synth examples/expressive.tbx --provider azure --voice en-US-JennyNeural -o azure.wav
```

Calls `POST https://{region}.tts.speech.microsoft.com/cognitiveservices/v1` with the subscription key and SSML, requesting `raw-24khz-16bit-mono-pcm`. Text and attributes are XML-escaped. The voice name chooses the model; specifying `--model` or front matter `model:` is an error.

Style, role, and emphasis support depend on the selected voice. Azure may ignore a style that a voice does not support; Timbrex does not yet query voice capabilities. The adapter allows one style per segment and rejects conflicting styles. `[role Boy]` needs a style in the same scope. When combined with prosody controls, emphasis wraps a single prosody element containing the requested rate, pitch, and volume attributes, so equivalent controls produce the same supported SSML hierarchy regardless of tag order. This is a request-structure guarantee, not proof that a selected voice supports those controls. In particular, do not infer HD control support from its voice name: Microsoft’s [HD support table](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/high-definition-voices#supported-and-unsupported-ssml-elements-for-azure-speech-hd-voices) and some SSML examples disagree about prosody and styles.

The root SSML language uses a recognizable `language[-Script]-region` prefix in the voice name, including `sr-Latn-RS-NicholasNeural` and lowercase `en-us-Ava:DragonHDLatestNeural`. Locale casing is normalized; the voice ID is sent unchanged. An explicit `lang` that differs beyond casing is emitted inside `voice` as `<lang xml:lang="…">`, as required for multilingual language/accent overrides; the selected voice must support the target language. Names without this locale shape need explicit `lang` as the root default. Locale inference does not validate voice availability.

Azure REST truncates audio at ten minutes, potentially with HTTP 200. Timbrex rejects a segment at or above 28,800,000 bytes (ten minutes of the requested 24 kHz mono S16LE PCM) rather than treating it as complete; split long speech into shorter paragraphs. A genuinely complete segment of exactly ten minutes is also rejected because raw PCM has no completion marker. This boundary check cannot prove that shorter audio contains every requested word.

References: [REST API](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/rest-text-to-speech), [voice, style, role, and prosody](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice), [SSML structure](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-structure#document-structure), [voice catalog](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/language-support?tabs=tts#text-to-speech).

## Qwen

- Provider ID: `qwen`.
- Environment setup: `DASHSCOPE_API_KEY`, `QWEN_ENDPOINT`.
- Default model: `qwen-audio-3.1-tts-flash`.
- Default voice: `longanhuan_v3.1`.
- Optional model default: `QWEN_MODEL`.
- Optional voice default: `QWEN_VOICE`.
- Endpoint override: `QWEN_ENDPOINT` (required full request URL).

```bash
# DASHSCOPE_API_KEY must already be set; replace the workspace placeholder.
export QWEN_ENDPOINT='https://YOUR_WORKSPACE_ID.ap-southeast-1.maas.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer'
./timbrex synth examples/expressive.tbx --provider qwen -o qwen.wav
```

This adapter targets **Qwen-Audio-TTS 3.0/3.1**, not the older Qwen3 multimodal-generation API or locally hosted Qwen3 weights. It supports the `qwen-audio-3.0-tts-flash`, `qwen-audio-3.0-tts-plus`, `qwen-audio-3.1-tts-flash`, and `qwen-audio-3.1-tts-plus` model IDs. Choose a compatible voice when changing models.

The workspace endpoint is required. Singapore uses `ap-southeast-1`; Beijing uses `cn-beijing` in the same URL shape. The request uses `input.instruction` (singular), PCM format, and a 24000 Hz sample rate. A completed response provides a signed audio URL, which Timbrex downloads **without** the API key. Language hints accept `zh en fr de ja ko ru pt th id vi es it ms fil ar`; locale suffixes are removed.

References: [HTTP API](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api), [models and usage restrictions](https://www.alibabacloud.com/help/en/model-studio/non-realtime-tts-user-guide).

## Gemini TTS

- Provider ID: `gemini`.
- Environment setup: `GEMINI_API_KEY`.
- Default model: `gemini-3.8-flash-tts`.
- Default voice: `Kore`.
- Optional model default: `GEMINI_MODEL`.
- Optional voice default: `GEMINI_VOICE`.
- Endpoint override: `GEMINI_BASE_URL` (`https://generativelanguage.googleapis.com/v1beta`).

```bash
# GEMINI_API_KEY must already be set.
./timbrex synth examples/gemini.tbx -o gemini.wav
```

Uses the **Google Gemini API**, not Google Cloud Text-to-Speech or Interactions: `POST /v1beta/models/gemini-3.8-flash-tts:generateContent` with `x-goog-api-key`. The initial adapter supports only this model and the 30 documented prebuilt voices; the default is `Kore`. Custom/replicated voices, voice creation, streaming and legacy model schemas are outside this adapter's scope. Cast voices use separate segment requests, not native conversational synthesis; optional concurrency does not combine them.

Spoken text and `speechMetadata.style` are separate. `fast`/`slow`, acting, emotions and delivery controls are approximate style instructions; numeric `speed`, `pitch` and `volume` are rejected. Selected events use native angle-bracket cues. Explicit `lang` maps to `speechConfig.languageCode` and must be one of the API reference's supported BCP-47 codes (for example `en-US` or `cmn-CN`, not `en` or `zh-CN`); without it the model detects language. The adapter limits text plus style to 4096 Unicode characters as a local resource bound. This is **not** token counting; the service's 8192-token input budget still applies.

Requests explicitly select `AUDIO_WAV`, `INLINE` and 24000 Hz. A response must contain exactly one `STOP` candidate and one inline `audio/wav` part, without prompt/output blocking. Unexpected parts, multiple candidates, malformed JSON/base64 and API error objects fail. Documented non-payload metadata may accompany the audio; thought-marked audio and extra payloads are rejected. The 64 MiB HTTP limit includes JSON/base64 overhead. WAV decoding follows the shared format checks below.

References: [GenerateContent TTS guide](https://ai.google.dev/gemini-api/docs/generate-content/speech-generation), [GenerateContent schema](https://ai.google.dev/api/generate-content), [model and limits](https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash-tts). The [current product TTS guide](https://ai.google.dev/gemini-api/docs/speech-generation) leads with the Interactions API, which this adapter does not call. Offline fixtures verify the selected schema and rejection behavior; no real account or acoustic verification has been performed.

## Fish Audio

- Provider ID: `fish`.
- Environment setup: `FISH_API_KEY`, voice/reference ID.
- Default model: `s2.1-pro`; also supports `s2.1-pro-free` and `s2-pro`.
- Default voice: None; set `FISH_VOICE_ID`, `--voice`, front matter, or cast mapping.
- Optional model default: `FISH_MODEL`.
- Optional voice default: `FISH_VOICE_ID`.
- Endpoint override: `FISH_BASE_URL` (`https://api.fish.audio/v1`).

```bash
# FISH_API_KEY and FISH_VOICE_ID must already be set to your own resources.
./timbrex synth examples/fish.tbx -o fish.wav
```

Calls `POST https://api.fish.audio/v1/tts` with bearer authentication, the selected `model` header, JSON and an explicit single-speaker `reference_id`. Supported models are `s2.1-pro`, `s2.1-pro-free`, and `s2-pro`, using the same single-speaker request contract and S2 bracket cues. Unknown models, legacy `s1`, and `drama-3` fail locally; the service otherwise silently falls back for unrecognized model headers. The free model has fair-use limits and no TTFA or DPA guarantees; it is not chosen automatically. No account voice is chosen by default. Use `FISH_VOICE_ID`, `--voice`, front matter or a `fish` cast mapping for a voice you can access. The adapter does not upload reference audio, create clones, or use native multi-speaker requests; cast voices use separate segment requests, including when concurrency is enabled.

Speed maps to the documented 0.5–2 multiplier and volume is passed in dB. Volume has a local −60…+20 dB bound, not a claimed vendor limit. Loudness normalization is explicitly enabled. Text normalization is explicitly disabled to avoid the API's additional number/date/unit rewriting. Speech pronunciation is still model-dependent. Explicit `lang` fails: Fish's `language` field only affects normalization/timestamp alignment and cannot select spoken language or accent. Selected S2 emotion and event cues use square brackets. `act` accepts one positional direction or a bare note, such as `[act "whispers sweetly"]` or `[act holding back tears]`, and maps it to a natural-language bracket cue. Directions must be nonempty and cannot contain ASCII `[`/`]` or the native token prefix `<|`. Numeric emotion intensity, pitch and unsupported controls fail before any HTTP call. Text including generated cues has a local 4096-character bound, not a claimed vendor maximum.

Requests specify `format: wav` and `sample_rate: 24000`. WAV decoding follows the shared format checks below. JSON errors in a successful HTTP response also fail. A valid WAV does not prove the model recited every word. Official HTTP errors include 400, 401, 402, 403, 404, 429, and 5xx; they fail without retry, and response bodies stay out of diagnostics.

References: [TTS endpoint](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech), [OpenAPI schema](https://docs.fish.audio/api-reference/openapi.json), [errors](https://docs.fish.audio/api-reference/errors), [model choice](https://docs.fish.audio/developer-guide/models-pricing/choosing-a-model), [S2 models and natural-language control](https://docs.fish.audio/developer-guide/models-pricing/models-overview), [S2 emotion and event cues](https://docs.fish.audio/developer-guide/core-features/emotions). Offline fixtures cover requests and rejection paths. Live account access, complete-WAV interoperability and acoustic behavior remain unverified; streaming sentinel sizes are deliberately rejected rather than guessed.

## Concurrent generation

Real providers support `synth --concurrency N`, with a default of `1` (serial) and an allowed range of `1`–`8`. Start with `2` only if your account permits simultaneous requests. This is a local safety cap, not a claim that every provider, plan, region, or model accepts eight concurrent requests. The fake `stub` provider rejects values above `1`. The option is CLI-only; it has no configuration-file, front matter, or environment equivalent.

```bash
# Configure OpenAI credentials first; this command can incur charges.
./timbrex synth examples/dialogue.tbx --provider openai --concurrency 2 -o dialogue.wav
```

Every speech request is prepared and validated before any HTTP call. Concurrency only overlaps those prepared segment jobs: it does not split or merge segments, change request text, voices or controls, or add retries. The final WAV always places decoded speech and local pauses in script order, regardless of completion order. Pauses make no network calls. Qwen's synthesis request and subsequent audio download belong to the same segment job.

Both active jobs and the window of segments not yet written are bounded by `N`. A slow early segment can hold up that window rather than allowing later PCM to accumulate for the whole script. The byte lengths of completed PCM payloads waiting in memory are bounded by the window size times the largest decoded segment (at most 64 MiB per real-provider response). This is not a retained-heap or process-memory budget. HTTP buffers, JSON/hex/base64 decoding and WAV backing buffers can coexist with PCM; prepared requests and the source, AST and IR also remain in memory. Higher concurrency can therefore use substantially more memory.

On the first recorded synthesis, decoding, or writing failure, the renderer stops admitting jobs, cancels in-flight work, and waits for started jobs to finish before removing temporary output. The error that triggers cancellation is retained rather than replaced by a sibling's cancellation error; competing failures need not be reported in script order. HTTP 429 and other failures are not retried, and concurrency is not automatically reduced. Cancellation cannot undo a request already accepted by a service: completed or uncertain in-flight requests may still be billed. Existing output is preserved on a failure before publication.

Offline tests verify request contents, ordering, bounds and cleanup with simulated responses. They do not establish live-service speedups or byte-identical speech: provider generation may vary between runs even when request contents are unchanged.

## Failures and limits

All speech segments are checked against the adapter before the first request. This catches unsupported controls and invalid local settings, but does not test account access or guarantee that the service will accept the request. Timbrex does not automatically split text to fit a vendor's token limit; use shorter paragraphs for long speech.

Each HTTP request has a two-minute timeout and a 64 MiB response limit. Responses whose known content length exceeds this limit are rejected without reading the body. Unknown-length and automatically decompressed bodies remain subject to the same limit while reading. The final PCM output is limited to 1 GiB. Runtime failures identify the provider and segment number. HTTP errors include the status; MiniMax and Qwen business errors include the vendor error code. Diagnostics omit response bodies and signed URL query strings. HTTP 206 partial responses and interrupted bodies fail. No request is retried automatically.

### Audio format checks

OpenAI, ElevenLabs, MiniMax, Azure, and Qwen request raw 24 kHz mono S16LE PCM. The adapters reject conflicting media types or mismatched explicit `rate`/`channels` metadata. The renderer also rejects recognizable audio containers, empty PCM, and odd byte counts. Missing optional metadata remains accepted: raw PCM bytes do not identify their sample rate or channel count, so the format still depends on the vendor honoring the request.

Gemini and Fish request WAV and extract its PCM before rendering. Their decoder requires complete RIFF lengths, a single PCM1 format chunk and audio data chunk, mono 24 kHz 16-bit samples, block alignment 2, and byte rate 48000. Extra RIFF metadata chunks are allowed. Truncated files, unknown-size streaming WAV, duplicate audio chunks, and other encodings fail. Complete-WAV interoperability with live services remains unverified.

These checks detect container and format problems. They cannot prove that a model spoke every word or followed a direction accurately. See each provider's section for further limits, including Azure's ten-minute cutoff.

### Output and recovery

All providers, including `stub`, create a temporary file beside the destination and replace the destination only after successful synthesis, writing, and closing. On POSIX systems, temporary and completed files use private `0600` permissions, subject to a more restrictive process umask. Replacing an existing file replaces its permissions too; the old mode is not preserved. The stub remains an 8 kHz mono 16-bit test tone; real-provider output remains 24 kHz mono 16-bit audio.

Cancellation is checked during request preparation, before creating output, while scheduling and writing real-provider segments, and immediately before the final rename. In-flight HTTP requests receive the cancellation context. A cancellation or failure reported before replacement removes temporary output and preserves an existing destination. Cancellation arriving after the final check can still finish the replacement; successful replacement is not rolled back. The parent directory must already exist. Earlier requests may have incurred charges; there is no automatic retry, fallback, cache, or resume.

When a request fails, check the model, voice, region, and key permissions against the provider's official reference. Offline tests validate Timbrex's request construction and response handling; they do not establish service availability or speech quality.
