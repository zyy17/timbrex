# MiniMax 语音参考

[English](readme.md) · [Timbrex 配置](../../providers.zh-cn.md) · [已支持的控制](../../capabilities.zh-cn.md)

面向开发者的厂商速查，核对日期：**2026-10-24**。本页介绍厂商提供的功能，不代表 Timbrex 已支持全部能力；当前支持范围见上方链接。

## 模型与特点

`speech-2.8-hd` 侧重音质，`speech-2.8-turbo` 侧重延迟和成本。两者支持 40 种语言、七种情绪选择和定点插入的非语言声音。2.6 和 02 系列仍列为历史模型。[模型](https://platform.minimax.io/docs/guides/models-intro)。

## 功能

- 语音控制：音色、0.5–2 倍语速、(0, 10] 音量、−12–12 的整数音高参数、情绪、语言和方言提示，以及 19 种声音标签。`fluent` 和 `whisper` 情绪仅适用于 2.6；2.8 不接受 `whisper`。
- 发音：替换词典、拼音、IPA、日语读法和行内发音标注；另有原生停顿、中文 LaTeX 朗读、音色混合和可选音效。
- 输出：MP3、PCM、WAV、FLAC、Ogg/Opus 和 8 kHz μ-law；可配置采样率、声道和特定格式的码率，并返回句级或词级字幕及用量信息。[HTTP API 与完整参数](https://platform.minimax.io/docs/api-reference/speech-t2a-http)。
- 文本规范化：`basic` 侧重低延迟；`quality` 为非流式 2.8/2.6 HTTP 请求增加上下文处理。[规范化与接入地址](https://platform.minimax.io/docs/guides/speech-t2a-websocket)。

## 接口与限制

- HTTP 一次提交少于 10,000 字符的完整文本，可选择一次性返回或流式输出；超过 3,000 字符时建议考虑流式输出。
- WebSocket 提供按句和双向文本流式合成，属于独立传输方式，而非 HTTP 参数。[接口目录](https://platform.minimax.io/docs/llms.txt)。
- 异步合成支持最多 50,000 字符的直接文本，或最多 1,000,000 字符的文本文件。文件任务还支持 TXT、JSON、ZIP。[异步 API](https://platform.minimax.io/docs/api-reference/speech-t2a-async-create)。
- 音色复刻、设计、查询和删除属于独立 API；生成的音色 ID 可用于合成。音色混合得到一个混合音色，不等于多人对话。[音色复刻](https://platform.minimax.io/docs/api-reference/voice-cloning-clone)、[音色设计](https://platform.minimax.io/docs/api-reference/voice-design-design)。

## 价格与地域

以下为公开按量价格，不含账户专属优惠：

| 服务 | 国际平台 USD | 中国平台 CNY |
| --- | --- | --- |
| 2.8 HD，同步或异步 | $100 / 百万输入字符 | ¥3.50 / 万输入字符 |
| 2.8 Turbo，同步或异步 | $60 / 百万输入字符 | ¥2.00 / 万输入字符 |
| 快速音色复刻 | $1.50 / 音色 | ¥9.90 / 音色 |
| 音色设计 | $3 / 音色 | ¥9.90 / 音色 |

新音色在首次用于合成时收取音色费，试听合成另行计费。中国平台将一个汉字计为两个字符。国际订阅使用每月音频积分，中国资源包使用字符额度，不能直接当作按量价格折扣。[国际价格](https://platform.minimax.io/docs/pricing/overview)、[中国价格](https://platform.minimax.cn/docs/pricing/overview)。

接入地址、密钥和计费账户应匹配目标平台：国际平台 `api.minimax.io`、美国西部 `api-uw.minimax.io`，或中国平台当前文档中的 `api.minimax.cn`。不要假定密钥或余额可以跨平台使用。[国际接入地址](https://platform.minimax.io/docs/guides/speech-t2a-websocket#endpoints)、[中国 API](https://platform.minimax.cn/docs/api-reference/speech-t2a-http)。
