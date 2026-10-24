# Qwen / Alibaba Cloud Model Studio TTS

[简体中文](readme.zh-cn.md) · [Timbrex setup](../../providers.md) · [Supported controls](../../capabilities.md)

Developer quick reference, reviewed **2026-10-24**. This covers the vendor’s TTS offering only, not full Timbrex support; see the links above for the current adapter scope.

Qwen-Audio-TTS combines multilingual preset/custom voices, acting instructions, pronunciation controls and streaming. Choose the API family first: **Qwen Audio 3.x TTS, Qwen3-TTS and TTS-Next do not share one request contract**.

## Models and regions

The current standard speech family comprises `qwen-audio-3.1-tts-flash`, `qwen-audio-3.1-tts-plus`, `qwen-audio-3.0-tts-flash` and `qwen-audio-3.0-tts-plus`. All four have Beijing and Singapore synthesis endpoints. Voice availability and capabilities vary by model and region; consult the [model/voice guide](https://www.alibabacloud.com/help/en/model-studio/non-realtime-tts-user-guide) and [voice catalog](https://help.aliyun.com/en/model-studio/qwen-audio-tts-voice-list).

## Speech capabilities

- Sixteen language hints, dialect-aware voices, natural-language emotion/style/role instructions, 23 inline delivery controls and seven nonverbal events. Instructions apply to eligible existing system/custom voices; dialect synthesis is voice-specific. [Guide](https://www.alibabacloud.com/help/en/model-studio/non-realtime-tts-user-guide)
- Rate and pitch 0.5–2, volume 0–100, seed control, pronunciation hot-fixes and substitutions. MP3, PCM, WAV and Opus; sample rates span 8–48 kHz with codec-specific restrictions. Optional AIGC provenance tags exclude PCM. [HTTP options](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api)
- Eligible system voices and existing cloned-voice IDs support SSML: pauses, aliases, Chinese Pinyin/English CMU phonemes, specialized reading, sound events, voice/effect controls. SSML availability is not universal. The detailed LaTeX guide names CosyVoice, so Qwen Audio LaTeX support is not established here. [SSML guide](https://help.aliyun.com/en/model-studio/ssml-latex-user-guide)
- Whole-text HTTP returns a downloadable result; SSE returns audio progressively. Word timing is a streaming-only, voice-dependent feature. Download URLs expire after 24 hours. WebSocket incremental-text synthesis is a separate transport. [HTTP](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api) · [WebSocket](https://www.alibabacloud.com/help/en/model-studio/realtime-tts-user-guide)

No explicit whole-text character ceiling was established for these four models in the current HTTP reference. Do not apply Qwen3's 600-character limit to them. The standard Flash/Plus contract does not document native multi-speaker dialogue or previous/next-text context.

## Other TTS families

- **Qwen3-TTS:** a separate API and voice catalog. The undated Flash alias remains documented, but an official notice scheduled many dated, instruct, VC and VD variants for retirement on 2026-10-10. Check the exact ID rather than copying old examples. [API](https://www.alibabacloud.com/help/en/model-studio/qwen-tts-api) · [Retirement notice](https://www.alibabacloud.com/en/notice/model_studio_notice_of_retirement_for_selected_legacy_longtail_models_7d4?_p_lc=1)
- **TTS-Next (speech uses only):** text-driven single-speaker speech, dialogue and spoken podcasts in Chinese/English. It uses a separate API, accepts up to 3,000 text characters and returns non-streaming WAV, MP3 or PCM. Podcasts can last up to four minutes; other speech up to two minutes. [Model reference](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-3-1-tts-next)

## Standard pricing

Prices checked 2026-10-24; **account catalog and deployment region both matter**. These are two common combinations, not currency conversions:

| Model | Billing unit | International catalog, Singapore (USD) | China catalog, Beijing (CNY) |
| --- | --- | --- | --- |
| 3.1 Flash | 1M input / output tokens | $0.23 / $1.87 | ¥1.50 / ¥12 |
| 3.1 Plus | 10K input characters | $0.30 | ¥1.40 |
| 3.0 Plus | 10K input characters | $0.20 | ¥1.40 |
| 3.0 Flash | 10K input characters | $0.15 | ¥1.00 |

Character billing counts Han characters as two and other characters as one; SSML tags are excluded. Eligible trial quotas expire after 90 days and apply to Singapore in the international catalog, Beijing in the China catalog. Other regional rates, promotions and account eligibility require checking the correct catalog. [International pricing](https://www.alibabacloud.com/help/en/model-studio/model-pricing) · [China pricing](https://help.aliyun.com/zh/model-studio/model-pricing)

TTS-Next international rates per 1M input/output tokens are $0.93/$1.87 in Singapore and $0.848/$1.696 in Beijing. Do not estimate token-billed audio with the character tariffs above. [Model pricing](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-3-1-tts-next)
