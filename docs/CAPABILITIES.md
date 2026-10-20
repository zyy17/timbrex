# TTS provider capability matrix

Checked 2026-10-20. Chinese text: [CAPABILITIES.zh-CN.md](CAPABILITIES.zh-CN.md). This file is the English primary.

Legend: confirmed in official docs; conditional or partial; not found in official docs and needs a live check; officially unsupported.

The native-syntax column is what an adapter should emit. It is not Timbrex syntax.

## 1. Commercial APIs

### Azure AI Speech (Neural / HD)

SSML is confirmed (`break`, `prosody`, `emphasis`, `say-as`, `phoneme`, `sub`, `lang`, `voice`, `audio`, `bookmark`, `lexicon`). Style and role are confirmed on some voices via `mstts:express-as`. HD voices use inline marks such as `[whispering]` until `[Neutral]`, a new sentence, or a newline. Silence, target duration, background audio, multi-voice requests, word and viseme events, and custom or personal voice are confirmed. There is no general nonverbal tag. Streaming is confirmed.

Sources: Microsoft Learn, speech synthesis markup, HD voices, and markup structure.

### Google Cloud TTS

Neural2, WaveNet, Standard, and Studio take standard SSML. Chirp 3 HD SSML is a preview and sync-only. Chirp 3 HD pauses are `[pause short]`, `[pause]`, and `[pause long]` in the `markup` field. Chirp 3 HD does not take speed or pitch parameters. Gemini TTS takes a natural-language style (`speech_metadata.style`) and inline voice events such as `<sigh>` and `<laughs>`. Multi-speaker setup, emphasis by capitals, and partial `<mark>` timestamps are confirmed. A Cartesia summary said Gemini 2.5/3.1 preview TTS would shut down after 2026-11-17. Treat that as a vendor announcement, not as this file's claim.

### AWS Polly

SSML is confirmed. Generative voices take a subset and error on unsupported tags. Emotion is unsupported. Newscaster domain, whispered effect, and DRC are conditional. Speech marks are confirmed.

### ElevenLabs (v3 / v4)

Audio tags are free text, not an enum: `[laughs]`, `[whispers]`, `[sighs]`, and many others. Effect, accent, and singing tags are experimental. v3/v4 do not take SSML `<break>`. Emphasis is capitals and punctuation. Dialogue is the Text to Dialogue API. Speed and stability, pronunciation dictionaries, timestamps, and voice design or cloning are confirmed.

### OpenAI gpt-4o-mini-tts

Style, emotion, accent, pace, and tone go through `instructions`. `speed` is a parameter. SSML and inline tags are unsupported. Multi-speaker and timestamps are unsupported. The model is deprecated and scheduled to shut down on 2027-01-06, with migration toward the Realtime API.

### Cartesia Sonic-3

Speed, volume, emotion (beta, English, a large enum), break, spell, `[laughter]`, pronunciation dictionaries, and streaming word timestamps are confirmed.

### Hume Octave

Acting instructions are a per-utterance `description` on Octave 1. Octave 2 docs have called this upcoming. Speed, trailing silence, per-utterance voice changes, and voice design are confirmed.

### Fish Audio

S2 and S2.1-Pro take free `[whisper]`-style prompts anywhere, including natural language. S1 (retiring 2026-12-31) uses a fixed parenthetical set. Cloning and streaming are confirmed.

### MiniMax Speech (T2A v2)

Pauses are `<#0.5#>`. Emotion is `voice_setting.emotion`. Speed, volume, and pitch are parameters. Pinyin, IPA, and Jyutping are parenthetical. Speech 2.8 adds a fixed nonverbal parenthetical list. Voice mixing, subtitles, cloning, and voice design are confirmed. Whisper as an emotion value is model-dependent.

### Volcengine Doubao speech synthesis 2.0

Official 2.0 voices take `context_texts` instructions. Emotion enums and SSML on 2.0 were not confirmed in the docs checked. Cloning (`seed-icl-2.0`) is confirmed.

### Alibaba CosyVoice / Qwen-TTS

CosyVoice SSML is confirmed behind `enable_ssml`. Instruction text is confirmed and voice-specific. Qwen-Audio-TTS has used inline tags plus instruction. Qwen3-TTS instruction support was not confirmed. LaTeX reading is documented with SSML.

### Tencent Cloud TTS

Emotion category and intensity are confirmed on multi-emotion voices. SSML is confirmed, mainly for Chinese. Speed and volume are parameters.

### iFlytek hyper-real synthesis

`oral_level` is `high`, `mid`, or `low`. Speed, volume, and pitch exist. Emotion and nonverbal controls were not confirmed.

## 2. Open models

| Model | Confirmed control |
| --- | --- |
| ChatTTS | Token-level `[laugh]`, `[uv_break]`, `[lbreak]`. Sentence refine prompts. Speaker embedding sampling. Speed is unstable. |
| CosyVoice 2/3 | Inline nonverbal tokens, span tags such as `<strong>`, instruct text, pinyin and ARPAbet, zero-shot cloning, streaming |
| IndexTTS2 | An 8-d emotion vector, `emo_text`, an emotion reference clip, `emo_alpha`, duration control |
| F5-TTS | Reference-audio cloning, speed, diffusion steps. No inline tags |
| Dia | Speakers `[S1]` `[S2]` and a parenthetical nonverbal list |
| Kokoro | `[word](/IPA/)`, stress marks, speed |

## 3. Cross-check

Four control styles show up together: SSML, inline tags, natural-language instructions, and structured parameters.

The trend is from fixed enums toward free text (Fish S1 to S2, ElevenLabs tags, Gemini style). Two widely used APIs have lifecycle risk (OpenAI gpt-4o-mini-tts, Gemini preview TTS). The DSL must treat free-text acting as a first-class feature and keep adapters replaceable.

Nonverbal overlap across four or more vendors: laugh, sigh, breath or inhale or exhale, cough, gasp, clear-throat, sniff. Those belong in the core vocabulary.

## 4. Leaderboard models (Artificial Analysis, 2026-10)

The ranking list was supplied upstream and was not re-ranked here. The control notes below were checked.

| Model | Confirmed control | Multi-speaker |
| --- | --- | --- |
| ElevenLabs Eleven v4 / v4 Turbo | Same free-text `[...]` tags as v3. Dialogue API. No SSML break | Yes |
| Alibaba Qwen-Audio-3.1-TTS-Plus | `instruction`, rate, pitch, volume, SSML, pronunciation hot-fix, seed, character timestamps. Inline tags were documented for other 3.x flash/plus ids, not clearly for 3.1-plus | Unconfirmed |
| Cartesia Sonic 3.6 | Same family as Sonic-3 | No |
| Google Gemini 3.8 Flash TTS | Verbatim `text`, `speech_metadata.style`, inline `<sigh>`-style events, capitals, voice design, multi-speaker | Yes |
| Inworld Realtime TTS-2 | English `[...]` steering until it changes or `[reset]`, a nonverbal list of about 50, breaks, custom pronunciation, verbatim, timestamps and visemes, `delivery_mode` | Unconfirmed |
| Fish Audio Drama 3 | Free `[...]` in any language, a small set of paired word tags, experimental language tags, phonemes, temperature and top_p | Yes, with a speaker token |
| MiniMax Speech 2.8 | Parenthetical nonverbals, `<#x#>`, emotion enum, phonetic marks | No |
| StepFun StepAudio 2.5 / 3 | Global `instruction`, inline `(...)` that is not spoken, speed, volume, pronunciation map, streaming, cloning | No |

Head models use brackets plus free text, and they do not agree which bracket means what. The compiler must escape ordinary parentheses for Step and MiniMax. StepAudio documents that parenthetical text is not spoken.

## 5. More providers and voice features

See [MARKET_SURVEY.md](MARKET_SURVEY.md) for the market table. Short control notes:

| Provider | Expressive control | Voice |
| --- | --- | --- |
| xAI / Grok TTS | Voice tags (exact grammar not fully confirmed), speed, a `replace` pronunciation table, text normalization, character timestamps | Built-in voices and cloning with a passphrase plus a voiceprint check |
| Mistral Voxtral TTS | No tags. Emotion and accent come from the reference audio | Zero-shot cloning on the API. The open build is preset voices only |
| Murf Falcon 2 | Not confirmed here | Short cloning for enterprise, with consent |
| Deepgram Aura-2 | Little | No cloning in the sources checked |
| ElevenLabs voices | — | IVC, PVC, Voice Design, Voice Remix, Voice Library, voice-captcha |
| MiniMax voices | — | Cloning, voice design, `timbre_weights`, identity verification |
| Doubao voice clone 2.0 | — | Short reference. Best results need a longer take |
| Alibaba CosyVoice / Qwen-Audio voices | — | Cloning, voice design, optional hidden AIGC mark |

MiniMax's clone-preview word list is longer than the T2A nonverbal list (it adds items such as whistles, crying, and applause). Adapters should re-check that list against the clone API docs.
