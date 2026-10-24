# OpenAI 语音

[English](readme.md) · [Timbrex 配置](../../providers.zh-cn.md) · [已支持的控制](../../capabilities.zh-cn.md)

面向开发者的厂商速查，核对日期：**2026-10-24**。本页介绍厂商提供的功能，不代表 Timbrex 已支持全部能力；当前支持范围见上方链接。

## 选择合适的 API

OpenAI 的 Speech API 适合通过提示词控制旁白表现和生成多语言语音。`tts-1`、`tts-1-hd` 和 `gpt-4o-mini-tts` 已弃用，计划于 **2027-01-06** 停用。官方建议通过 Realtime API 使用 `gpt-realtime-2.1-mini`；不能只替换现有 Speech API 请求中的模型名。[生命周期](https://developers.openai.com/api/docs/deprecations#2026-10-01-text-to-speech-models)

## 特性与取舍

- **表达控制：**GPT-4o Mini TTS 可以通过指令控制口音、情绪、语调、节奏和耳语。`tts-1` 偏重低延迟，`tts-1-hd` 偏重音质，但两者都不支持这些指令。音色主要针对英语优化，也支持多语言合成。[语音指南](https://developers.openai.com/api/docs/guides/text-to-speech)
- **音色选择：**Speech API 列出 13 个内置音色，也接受已获准的自定义音色引用；具体音色支持取决于模型。[Speech 参考](https://developers.openai.com/api/reference/resources/audio/subresources/speech/methods/create)
- **音频交付：**支持 MP3、Opus、AAC、FLAC、WAV 和原始 PCM，可逐步接收音频。PCM 为 24 kHz、16 位有符号小端格式。也支持 SSE，但 `tts-1` 和 `tts-1-hd` 除外。[输出格式](https://developers.openai.com/api/docs/guides/text-to-speech#supported-output-formats)、[流式选项](https://developers.openai.com/api/reference/resources/audio/subresources/speech/methods/create)
- **自定义音色：**基于录音创建音色需要资格审批、授权录音和匹配的样本。通过文字设计音色属于另一项能力，只能用于 Live，不能用于 Speech 或 Realtime。[自定义音色](https://developers.openai.com/api/docs/guides/custom-voices)、[音色创建](https://developers.openai.com/api/reference/resources/audio/subresources/voices/methods/create)
- **交互应用：**Realtime 提供会话、打断和工具调用，应与固定脚本的语音渲染分开评估。[Realtime](https://developers.openai.com/api/docs/guides/realtime)

## 需要注意的限制

Speech 请求的正文和指令各不超过 4096 个字符；Mini TTS 另有 2000 个输入 token 的限制。语速范围为 0.25–4 倍。Speech 请求结构没有原生 SSML、音素词典或数值音高、音量控制，提示词不能当作精确声学约束。[API 限制](https://developers.openai.com/api/reference/resources/audio/subresources/speech/methods/create)、[模型限制](https://developers.openai.com/api/docs/models/gpt-4o-mini-tts)

应明确告知听众声音由 AI 生成。[披露要求](https://developers.openai.com/api/docs/guides/text-to-speech)

## 价格

以下为 2026-10-24 核对的公开美元 API 价格。token 与字符不能直接换算。

| 模型 | 价格 | 来源 |
| --- | --- | --- |
| GPT-4o Mini TTS | 每百万输入文本 token $0.60，加每百万输出音频 token $12 | [模型价格](https://developers.openai.com/api/docs/models/gpt-4o-mini-tts) |
| TTS-1 | 每百万字符 $15 | [模型价格](https://developers.openai.com/api/docs/models/tts-1) |
| TTS-1 HD | 每百万字符 $30 | [模型价格](https://developers.openai.com/api/docs/models/tts-1-hd) |
| Realtime 2.1 Mini | 每百万 token：文本输入/输出 $0.60/$2.40；音频输入/输出 $10/$20 | [Realtime 价格](https://developers.openai.com/api/docs/pricing#realtime-and-audio-generation-models) |

Realtime 按会话用量计费，并非固定的按字符 TTS 单价。估算生产成本前，应确认当前账号条款和链接中的价格。通过文字设计音色免费，但试听和使用会产生 GPT-Live 会话费用；本次未核实基于录音创建音色的独立单价。[音色计费说明](https://developers.openai.com/api/docs/guides/custom-voices#create-and-save-the-voice)
