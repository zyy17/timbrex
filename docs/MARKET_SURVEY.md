# TTS market and voice-capability survey

2026-10-20. Chinese text: [MARKET_SURVEY.zh-CN.md](MARKET_SURVEY.zh-CN.md). Expressive-control detail lives in [CAPABILITIES.md](CAPABILITIES.md). This file is the market picture plus cloning, design, and conversion.

Legend: confirmed from an official or primary source; from a third-party roundup and needs a re-check; not verified.

Prices are list prices and move. "$/1M characters" is a normalized unit where the source uses it.

## 1. Leaderboard snapshot (Artificial Analysis, 2026-10)

Provider Voice Arena leaders, as relayed from AA data by a third party: ElevenLabs Eleven v4 Turbo and Eleven v4, Qwen-Audio-3.1-TTS-Plus, Cartesia Sonic 3.6, Gemini 3.8 Flash TTS, Qwen-Audio-3.0-TTS-Plus, Speechify Simba 3.2, Fish Audio Drama 3 Preview. The board had on the order of 97 models, with a minority open-weight.

Open-weight top of the AA page at the time of the check: Fish Audio Drama 3 Preview, Breeze TTS 2, Ming-3.0-tts, Fish Audio S2 Pro, Step Audio EditX (Mar 2026).

Breeze TTS 2 and Ming-3.0-tts appeared on the board. Official repositories and licenses were not found in this pass.

Sources: `https://artificialanalysis.ai/text-to-speech/leaderboard/provider-voice` and the open-weight and controlled-voice boards.

## 2. Commercial APIs

Prices below that came from comparison blogs are third-party. Confirm them before depending on a number.

| Provider | Chinese | Control, short | Streaming | Price notes | Status |
| --- | --- | --- | --- | --- | --- |
| ElevenLabs v4 / v4 Turbo / v3 / Flash | Yes, 70+ languages. Dialects unconfirmed | Free `[tags]`, Dialogue API | Yes, v4 Turbo realtime | Turbo/Flash and v4 list prices differ | Active |
| OpenAI gpt-4o-mini-tts | Yes | `instructions` | Yes | Audio-token pricing | Deprecated, shutdown 2027-01-06 |
| Google Gemini Flash / Flash-Lite TTS | Yes | style plus inline tags, multi-speaker | Yes | Preview pricing, with a stated later increase | Preview |
| Google Chirp 3 HD | Yes | `[pause]`, SSML subset | Yes | Per-character list price | GA |
| Azure Neural / HD | Chinese plus dialects. Exact list needs a re-check | Full SSML, express-as | Yes | Neural and HD differ | Active |
| AWS Polly Neural / Generative | Mandarin confirmed. Cantonese unconfirmed | SSML subset | Yes | Neural and Generative differ | Active |
| Cartesia Sonic 3.6 | Yes | SSML-like plus English emotion | Yes, low latency | Plan-based | Active |
| Hume Octave 2 | Unconfirmed | description, speed | Yes | List price from a third party | Active |
| Inworld TTS-2 / Flash | Yes, language count from a third party | `[steering]` plus about 50 nonverbals | Yes | Tiered | Active |
| Fish S2.1-Pro / Drama 3 | Yes | Free `[...]`, word tags, multi-speaker | Yes | Similar list price in third-party notes | Active. S1 retires 2026-12-31 |
| MiniMax Speech 2.8 | Chinese and Cantonese | Parentheticals, pauses, emotion, phonetics | Yes | Turbo price from a third party | Active |
| Doubao speech 2.0 | Yes. Dialects unconfirmed | `context_texts` | Yes | RMB billing, not normalized here | Active |
| Alibaba Qwen-Audio / CosyVoice | Yes. Dialects via instruction | SSML, instruction, tags | Yes | Region prices differ | Active |
| Tencent Cloud TTS | Yes. Cantonese unconfirmed | SSML, emotion category | Yes | Not normalized | Active |
| iFlytek | Yes. Dialects unconfirmed | `oral_level` | Yes | Not normalized | Active |
| Baidu Intelligent Cloud | Yes | Not checked this pass | Unconfirmed | Unconfirmed | Unconfirmed |
| StepFun StepAudio 2.5 / 3 | zh/en, more languages in preview | instruction plus inline `(…)` | SSE and WebSocket | Not normalized | Active |
| Speechify Simba 3.2 | Language count from a third party | Unconfirmed | Yes | Third-party range | Active |
| Deepgram Aura-2 / Flux | No Chinese in the sources checked | Little | Yes, low latency | Third-party range | Active |
| Rime Mist v3 / Coda | Unconfirmed | Unconfirmed | Yes | Third-party | Active |
| Murf Falcon 2 | Yes | Unconfirmed | Yes | Per-minute list price | Active |
| Mistral Voxtral TTS | No Chinese. Several European languages plus hi/ar | Reference-audio performance | Yes | List price | Active since 2026-03 |
| xAI Grok TTS | Yes, 20+ languages | Voice tags, speed, replace, character timestamps | Yes | List price | Active |
| PlayHT | — | — | — | — | API shut down 2025-12-31 (third-party report) |

Kling and similar video products were not surveyed as standalone TTS APIs.

## 3. Voice capabilities (commercial)

| Provider | Zero-shot clone | Professional clone | Text design | Remix or conversion | Notes |
| --- | --- | --- | --- | --- | --- |
| ElevenLabs | IVC, best around 1–2 minutes | PVC, tens of minutes to hours | Yes, 20–1000 characters, three previews | Remix with strength. Speech-to-speech yes | Voice Library. PVC uses voice-captcha |
| MiniMax | About 10 s to 5 min, optional short prompt | Unconfirmed | Yes | Mix with `timbre_weights` | Identity verification. Unused voices expire |
| Doubao clone 2.0 | Product page says a few seconds. Best practice is longer | Unconfirmed | Unconfirmed | Unconfirmed | Cover both languages in the prompt when cloning across languages |
| Alibaba CosyVoice | About 10–20 s, with a continuous stretch | Unconfirmed | Yes, on Qwen-Audio | Unconfirmed | Account caps and idle expiry. Optional AIGC mark |
| Fish Audio | Reference audio or id. Minimum length not confirmed | Unconfirmed | Unconfirmed | Unconfirmed | Large community library |
| Mistral Voxtral | About 3 s, best 5–25 s, API only | No | No | No | Cross-lingual zero-shot. 20 presets. Open build cannot clone |
| xAI Grok | About 2 minutes of natural speech | No | No | A speech-to-speech conversation API, not a voice changer | Passphrase plus voiceprint |
| Murf Falcon 2 | About 10 s, enterprise | Unconfirmed | Unconfirmed | Unconfirmed | Claims explicit speaker consent |
| Inworld | About 3 s. Professional clone in beta | Beta | Unconfirmed | Unconfirmed | — |
| Gemini TTS | Unconfirmed | No | Voice design is mentioned | No | About 30 styles. SynthID not confirmed |
| Cartesia | IVC. Minimum unconfirmed | PVC mentioned | Unconfirmed | Voice changer unconfirmed | — |
| Azure | Personal Voice, consent recording | Custom Neural Voice, limited access | Unconfirmed | Unconfirmed | Limited access plus speaker consent |
| AWS Polly | No general zero-shot. Brand Voice is a project | Unconfirmed | No | No | — |

Speechify, Smallest, Rime, LMNT, Resemble, Deepgram, Gradium, Tencent, iFlytek, Baidu, and StepFun were not fully checked. Third-party notes: Speechify cloning, no cloning on Deepgram Aura-2, instant cloning on Gradium, a StepFun clone preview. Resemble has speech-to-speech and watermark products that were not re-read.

## 4. Open models

Licenses below that are marked uncertain were not re-read from the repository in this pass.

| Model | License notes | Chinese | Control | Clone |
| --- | --- | --- | --- | --- |
| Fish Drama 3 Preview | Open weights. License not confirmed | Yes | Free `[...]`, word tags, multi-speaker | Yes |
| Fish S2 Pro | Open weights. License not confirmed | Yes | Free `[...]` | Yes |
| Step Audio EditX | Unconfirmed | Yes | Emotion and style editing unconfirmed | Unconfirmed |
| CosyVoice 2/3 | Apache-2.0 reported | Yes, dialects via instruct | Inline tokens, instruct, pinyin | Yes, short reference |
| IndexTTS2 | Custom. Commercial terms unconfirmed | Yes | Emotion vector, emo_text, duration | Separate timbre and emotion references |
| F5-TTS | Code MIT. Weights CC-BY-NC reported | Yes | speed | Reference audio plus text |
| ChatTTS | Code AGPL reported. Weights non-commercial reported | Yes | `[laugh]` and refine tokens | Speaker sampling, not cloning |
| Dia | Apache-2.0 reported | English | `[S1]` `[S2]` plus parentheticals | Audio prompt |
| Kokoro 82M | Apache-2.0 reported | Limited | IPA, stress | No |
| Chatterbox | MIT reported | Unconfirmed | Emotion exaggeration parameter | Yes. Output watermark reported |
| Spark-TTS | Apache-2.0 reported | Yes | Gender, pitch, speed parameters | Yes |
| Higgs Audio v2 / v3 | v2 Apache-2.0. v3 research-only reported | Yes | Multi-speaker and effects unconfirmed | Yes |
| VibeVoice | MIT with a research-only statement reported | Yes | Long multi-speaker | Yes |
| GPT-SoVITS | MIT reported | Chinese, Cantonese, Japanese, English | — | Short zero-shot, longer fine-tune |
| Mistral Voxtral open build | CC BY-NC 4.0 confirmed | No | — | Presets only. The open build cannot clone |
| XTTS v2 | Non-commercial license reported. Company closed | Yes | — | Short reference |

## 5. What this means for Timbrex

1. Strong Chinese coverage in the sources checked: Qwen-Audio, MiniMax, Doubao, Fish, CosyVoice, Inworld, ElevenLabs, Azure. Deepgram, Mistral Voxtral, and Gradium did not show Chinese.
2. Cloning references have fallen to a few seconds on some APIs (Voxtral, Inworld, Doubao's claim, MiniMax, CosyVoice, Murf) while ElevenLabs IVC and Grok still want on the order of a minute or two. Professional cloning still wants a much longer corpus.
3. Text voice design is common: ElevenLabs, MiniMax, Qwen-Audio, Gemini, and Hume. ElevenLabs also remixes an existing voice.
4. Consent and marking differ: ElevenLabs voice-captcha, xAI passphrase plus voiceprint, MiniMax identity checks, Azure limited access, Alibaba's hidden AIGC mark, Chatterbox watermarks. Consent and watermark fields have to be first-class metadata.
5. Lifecycle risk: OpenAI gpt-4o-mini-tts (2027-01-06), Fish S1 (2026-12-31), PlayHT (already closed), Gemini preview pricing. Adapters must be replaceable, and caps need `deprecated`.
6. Open weights: Fish Drama 3 and S2 Pro led the open board in this snapshot. License is the selection constraint. Voxtral, F5, ChatTTS, XTTS, Higgs v3, and VibeVoice have non-commercial or research limits in the sources above.
