# Provider integration research

[简体中文](MARKET_SURVEY.zh-CN.md)

Review date: **2026-10-22**. This document records API design differences relevant to Timbrex. It is not a quality benchmark, price comparison, or legal assessment. No live listening study was performed. Model availability and service terms must be checked with the vendor when deploying.

| Service | Documented control surface | Consequence for Timbrex |
| --- | --- | --- |
| [OpenAI](https://developers.openai.com/api/docs/guides/text-to-speech) | Speech endpoint with model, voice, speed, and model-dependent instructions | Keep performance instructions separate from spoken input |
| [ElevenLabs](https://elevenlabs.io/docs/api-reference/text-to-speech/convert) | Voice-specific endpoint, model selection, voice settings, model-dependent audio tags | Translate supported acting and events into native tags; preserve model restrictions |
| [MiniMax](https://platform.minimax.io/docs/api-reference/speech-t2a-http) | Voice settings, discrete emotions, PCM/hex response, model-specific interjections | Map a finite supported subset and check both HTTP and API completion status |
| [Azure Speech](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice) | SSML voice, prosody, style, and role with voice-dependent support | Escape XML and distinguish supported syntax from actual voice capabilities |
| [Qwen-Audio-TTS](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api) | Workspace endpoint, instruction and numeric controls, downloadable audio | Require the correct workspace endpoint and separate authenticated synthesis from signed downloads |

All five provide a path to compatible PCM, allowing a small sequential WAV renderer without compressed-audio concatenation or a transcoding dependency. This is an implementation choice, not a claim that the vendors' other audio formats are unsuitable.

Three questions remain for live evaluation: whether each documented request works for the target account and region; how reliably each model follows expressive controls; and how audible segment boundaries are. A useful evaluation should record exact model/voice IDs, source scripts, requested parameters, date, failures, and listening criteria. It should distinguish API acceptance from performance quality.

Voice cloning, native multi-speaker generation, streaming, alignment, and sound-effect generation may be separate vendor products. Their existence does not mean Timbrex integrates them. See [current capabilities](CAPABILITIES.md) and [setup](PROVIDERS.md) for implemented behavior.

## Related open-source projects

These projects overlap with different layers of Timbrex. This comparison is based on their published repositories and documentation, not a runtime or listening evaluation.

| Project | Relevant work | What to study |
| --- | --- | --- |
| [Speech Markdown](https://github.com/speechmarkdown/speechmarkdown-js) (MIT) | Speech markup grammar, parser, and platform formatters | Authoring syntax and target-specific conversion |
| [SSMD](https://github.com/buchwandler/ssmd) (MIT) | Markdown-style voice, pronunciation, and prosody annotations with structural parsing and migration | Scope, diagnostics, and semantic preservation |
| [Speech SDK](https://github.com/Jellypod-Inc/speech-sdk) (Apache-2.0) | TypeScript synthesis, streaming, and multi-speaker orchestration with native-dialogue and stitch paths | Rendering decisions and explicit capability constraints |
| [Podcastfy](https://github.com/souzatharsis/podcastfy) (Apache-2.0) | Content-to-podcast generation with multiple speech backends | End-to-end authoring and audio workflows |

A compact syntax and several adapters alone do not distinguish Timbrex. Its intended contribution is a readable performance script with explicit roles, scoped directions, nonverbal events, and honest target mappings. Interoperability with existing markup and synthesis tools is worth evaluating before reproducing their features.
