# Provider integration research

[简体中文](market_survey.zh-cn.md) · [Tutorial](tutorial.md)

Review date: **2026-10-22**; Fish model/acting coverage refreshed **2026-10-24**. This document records API design differences relevant to Timbrex. It is not a quality benchmark, price comparison, or legal assessment. No live listening study was performed. Model availability and service terms must be checked with the vendor when deploying.

| Service | Documented control surface | Consequence for Timbrex |
| --- | --- | --- |
| [OpenAI](https://developers.openai.com/api/docs/guides/text-to-speech) | Speech endpoint with model, voice, speed, and model-dependent instructions | Keep performance instructions separate from spoken input |
| [ElevenLabs](https://elevenlabs.io/docs/api-reference/text-to-speech/convert) | Voice-specific endpoint, model selection, voice settings, model-dependent audio tags | Translate supported acting and events into native tags; preserve model restrictions |
| [MiniMax](https://platform.minimax.io/docs/api-reference/speech-t2a-http) | Voice settings, discrete emotions, PCM/hex response, model-specific interjections | Map a finite supported subset and check both HTTP and API completion status |
| [Azure Speech](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice) | SSML voice, prosody, style, and role with voice-dependent support | Escape XML and distinguish supported syntax from actual voice capabilities |
| [Qwen-Audio-TTS](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api) | Workspace endpoint, instruction and numeric controls, downloadable audio | Require the correct workspace endpoint and separate authenticated synthesis from signed downloads |
| [Gemini TTS](https://ai.google.dev/gemini-api/docs/speech-generation) | Structured `speech_metadata.style`, inline vocal tags, WAV or raw PCM | Keep turn-level style out of the transcript; unwrap WAV before the shared PCM renderer |
| [Fish Audio](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech) | Bearer `POST /v1/tts`, `model` header, `reference_id`, S2 bracket cues, WAV/PCM/MP3/Opus | Require an explicit voice ID; map selected cues and free-form acting; unwrap WAV before the shared PCM renderer |

Timbrex requests compatible PCM, either raw or inside a WAV container, then concatenates the samples into one WAV. It extracts PCM from Gemini and Fish WAV responses without resampling. The current renderer does not combine compressed audio or depend on a transcoder.

Live evaluation still has to answer three questions: whether each documented request works for the target account and region; how reliably each model follows performance controls; and how audible segment boundaries are. Record exact model and voice IDs, source scripts, requested parameters, date, failures, and listening criteria. Distinguish API acceptance from performance quality.

Voice cloning, native multi-speaker generation, streaming, alignment, and sound-effect generation may be separate vendor products. Their existence does not mean Timbrex integrates them. See [current capabilities](capabilities.md) and [setup](providers.md) for implemented behavior.

## Related open-source projects

These projects overlap with different layers of Timbrex. This comparison is based on their published repositories and documentation, not a runtime or listening evaluation.

- [Speech Markdown](https://github.com/speechmarkdown/speechmarkdown-js) (MIT): Speech markup grammar, parser, and platform formatters. Relevant to Timbrex: Authoring syntax and target-specific conversion.
- [SSMD](https://github.com/buchwandler/ssmd) (MIT): Markdown-style voice, pronunciation, and prosody annotations with structural parsing and migration. Relevant to Timbrex: Scope, diagnostics, and semantic preservation.
- [Speech SDK](https://github.com/Jellypod-Inc/speech-sdk) (Apache-2.0): TypeScript synthesis, streaming, and multi-speaker orchestration with native-dialogue and stitch paths. Relevant to Timbrex: Rendering decisions and explicit capability constraints.
- [Podcastfy](https://github.com/souzatharsis/podcastfy) (Apache-2.0): Content-to-podcast generation with multiple speech backends. Relevant to Timbrex: End-to-end authoring and audio workflows.
