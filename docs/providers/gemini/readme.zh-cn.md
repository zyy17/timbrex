# Gemini TTS

[English](readme.md) · [Timbrex 配置](../../providers.zh-cn.md) · [已支持的控制](../../capabilities.zh-cn.md)

面向开发者的厂商速查，核对日期：**2026-10-24**。本页仅介绍厂商的 TTS 能力，不代表 Timbrex 已全部支持；当前支持范围见上方链接。

## 模型与适用场景

| 模型 | 适用场景 | 语言 |
| --- | --- | --- |
| `gemini-3.8-flash-tts` | 富有表现力的旁白、复杂对话、地区发音 | 模型对比表列出 130 种 |
| `gemini-3.8-flash-lite-tts` | 更低成本的大规模朗读 | 模型对比表列出 101 种 |

两者均为文本输入、音频输出，共用 3.8 请求结构，限制为 **8,192 个输入 token / 16,384 个输出 token**。自动识别覆盖的语言多于显式语言代码选择器。TTS 模型不支持上下文缓存、思考或工具调用。来源：[Flash](https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash-tts)、[Flash-Lite](https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash-lite-tts)。

## 语音能力

- **表达控制：**通过每轮自然语言指令描述情绪、语速、音高、音量、耳语及重音。这些是表演要求，不是可精确量化的倍速、半音或分贝。
- **非语言发声：**可内联表达笑、哭、咳嗽、呼吸、清嗓、哈欠以及短暂停顿或长停顿；不以非人声音效为目标。
- **对话：**单次请求最多使用两个预置音色，逐轮指定说话人和风格；支持原生应和及重叠提示。自定义音色对话需要逐个说话人、逐轮合成。
- **音色身份：**应选择合适的已有地区音色；不要将长期口音、年龄和身份改变视为普通表达调整。
- **发音与时间信息：**本次查阅的 TTS 文档没有确立音素 / SSML 词典接口或有保证的词级对齐。

参见[语音指南](https://ai.google.dev/gemini-api/docs/speech-generation)及 [GenerateContent 指南](https://ai.google.dev/gemini-api/docs/generate-content/speech-generation)。

## 音色

- **音色目录：**包括 Kore、Puck 在内的 30 个精选音色，以及 Extended Voice Library 中数百个可搜索音色。[目录](https://ai.google.dev/gemini-api/docs/generate-content/speech-generation#voice-options)

合成可使用已有自定义音色 ID；音色创建和管理不在本参考范围内。[音色选择](https://ai.google.dev/gemini-api/docs/speech-generation)

## 输出与限制

非流式默认输出 **24 kHz、单声道、16-bit PCM 的 WAV 文件**；流式默认输出不带文件头的 PCM 分块。文档还列出原始 PCM、G.711 mu-law / A-law 和采样率选项。通用结构中虽有 MP3 / Opus，但模型专用输出指南未确认。Interactions 与 GenerateContent 都是有文档支持的 API 选项。[音频格式](https://ai.google.dev/gemini-api/docs/speech-generation#audio-output-formats)、[API 参考](https://ai.google.dev/api/generate-content)

请求和 token 配额取决于项目、模型及用量等级；应在 AI Studio 查看当前限额，不应假定固定 RPM。[速率限制](https://ai.google.dev/gemini-api/docs/rate-limits)

## 价格

Gemini Developer API 公开付费档的**每百万 token 美元价格**，分别计文本输入 / 音频输出，查阅日期为 **2026-10-24**（不是 Vertex AI 价格）：

| 计费模式 | 两款 3.8 模型的输入 | Flash 音频 | Flash-Lite 音频 |
| --- | ---: | ---: | ---: |
| Standard | $0.50 | $9.00 | $6.00 |
| Batch | $0.25 | $4.50 | $3.00 |
| Flex | $0.25 | $4.50 | $3.00 |
| Priority | $0.90 | $16.20 | $10.80 |

以上 3.8 价格适用至 **2026-12-31**；**2027-01-01** 起表中所有价格翻倍。音频按每秒 25 个 token 计费，因此当前 Standard 的纯输出费用约为 **Flash 每小时 $0.81 / Flash-Lite 每小时 $0.54**，不含输入。免费等级另有配额及数据使用条款。

价格页仍列有旧模型：`gemini-3.1-flash-tts-preview`、`gemini-2.5-pro-preview-tts` 的 Standard 输入 / 输出为 $1 / $20；`gemini-2.5-flash-preview-tts` 为 $0.50 / $10。其 Batch 价格减半。旧模型采用旧版提示及输出约定，不能直接套用 3.8 结构。[官方价格](https://ai.google.dev/gemini-api/docs/pricing)

[Batch](https://ai.google.dev/gemini-api/docs/batch-api) 是异步处理，目标为 24 小时内完成；[Flex](https://ai.google.dev/gemini-api/docs/flex-inference) 用延迟及可用性换取低价；[Priority](https://ai.google.dev/gemini-api/docs/priority-inference) 以更高价格优先保障延迟及可靠性。这些计费模式不等同于并发发送普通请求。
