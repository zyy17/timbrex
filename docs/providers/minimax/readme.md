# MiniMax speech reference

[简体中文](readme.zh-cn.md) · [Timbrex setup](../../providers.md) · [Supported controls](../../capabilities.md)

Developer quick reference, reviewed **2026-10-24**. This covers the vendor’s TTS offering only, not full Timbrex support; see the links above for the current adapter scope.

## Models and strengths

`speech-2.8-hd` prioritizes quality; `speech-2.8-turbo` prioritizes latency and cost. Both offer 40 languages, seven emotion choices, and positioned nonverbal sounds. The 2.6 and 02 families remain documented legacy options. [Models](https://platform.minimax.io/docs/guides/models-intro).

## Capabilities

- Speech controls: voice, speed 0.5–2, volume (0, 10], integer pitch −12–12, emotion, language/dialect hints, and 19 sound tags. `fluent` and `whisper` are 2.6-only emotion options; 2.8 does not accept `whisper`.
- Pronunciation: replacement dictionaries, Pinyin, IPA, Japanese readings, and inline pronunciation. Native pauses, Chinese LaTeX reading, voice blending, and optional voice effects are also available.
- Output: MP3, PCM, WAV, FLAC, Ogg/Opus, and 8 kHz μ-law; sample rate, channels, and format-dependent bitrate controls. Sentence/word subtitles and usage metadata are available. [HTTP API and complete parameter reference](https://platform.minimax.io/docs/api-reference/speech-t2a-http).
- Text normalization: `basic` favors latency; `quality` adds contextual processing for non-streaming 2.8/2.6 HTTP requests. [Normalization and endpoints](https://platform.minimax.io/docs/guides/speech-t2a-websocket).

## API choices and limits

- HTTP accepts complete text under 10,000 characters, with whole-response or streaming output. Text over 3,000 characters is a candidate for streaming.
- WebSocket offers sentence-oriented and bidirectional text streaming. These are separate transports, not HTTP flags. [WebSocket guide](https://platform.minimax.io/docs/guides/speech-t2a-websocket).
- Asynchronous synthesis accepts up to 50,000 inline characters or a text file up to 1,000,000 characters. File jobs also support TXT/JSON/ZIP packaging. [Async API](https://platform.minimax.io/docs/api-reference/speech-t2a-async-create).
- Synthesis accepts existing system/custom voice IDs. Voice blending produces one mixed timbre, not a multi-speaker dialogue. [Synthesis options](https://platform.minimax.io/docs/api-reference/speech-t2a-http).

## Pricing and regions

Public pay-as-you-go prices, before account-specific offers:

| Service | International USD | China CNY |
| --- | --- | --- |
| 2.8 HD, sync or async | $100 / million input characters | ¥3.50 / 10,000 input characters |
| 2.8 Turbo, sync or async | $60 / million input characters | ¥2.00 / 10,000 input characters |

China billing counts a Chinese character as two characters. International subscriptions use monthly audio points; China resource packs use character allowances. Neither should be treated as an automatic pay-as-you-go discount. [International prices](https://platform.minimax.io/docs/pricing/overview), [China prices](https://platform.minimax.cn/docs/pricing/overview).

Use the endpoint, key, and billing account for the intended platform: international `api.minimax.io`, US West `api-uw.minimax.io`, or the current China documentation's `api.minimax.cn`. Do not assume keys or balances transfer between platforms. [International endpoints](https://platform.minimax.io/docs/guides/speech-t2a-websocket#endpoints), [China API](https://platform.minimax.cn/docs/api-reference/speech-t2a-http).
