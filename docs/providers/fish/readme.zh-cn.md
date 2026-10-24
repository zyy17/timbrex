# Fish Audio

[English](readme.md) · [Timbrex 配置](../../providers.zh-cn.md) · [已支持的控制](../../capabilities.zh-cn.md)

面向开发者的厂商速查，核对日期：**2026-10-24**。本页仅介绍厂商的 TTS 能力，不代表 Timbrex 已全部支持；当前支持范围见上方链接。

## 快速了解

Fish Audio 侧重自然语言表演指令、多语言语音和可复用音色。其直接 TTS API 同时支持单人语音与原生多角色对话。

| 模型 | 主要特点 |
| --- | --- |
| `s2.1-pro` | 推荐用于生产的 S2 模型，支持 83 种语言和开放式表演指令 |
| `s2.1-pro-free` | 相同模型质量，适合开发和较小规模业务；受公平使用限制，不提供 TTFA 或 DPA 保证 |
| `s2-pro` | 上一代 S2，支持 80 多种语言 |
| `drama-3` | 表演指令、声音、停顿、成对表达标签和音素控制 |
| `s1` | 已弃用的旧模型，计划于 2026 年 12 月 31 日退役 |

S1 使用圆括号提示，S2 使用方括号指令。官方说明 S1 退役后，其请求会由 S2.1-Pro 处理。缺失或未知的模型请求头也会回退到 S2.1-Pro，因此模型名必须准确。参考[模型目录](https://docs.fish.audio/developer-guide/models-pricing/models-overview)和[模型选择](https://docs.fish.audio/developer-guide/models-pricing/choosing-a-model)。

## 能力范围

- **表演指令**：情绪、表达方式和人声反应。S2 支持自然语言描述，不限于固定情绪列表。Drama 3 增加词语范围的成对标签和实验性语言切换。
- **音色选择**：通过已有的音色库或自定义音色 ID 选择合成音色。
- **多角色对话**：S2 和 Drama 3 可一次生成多个说话人的对话，无需分别生成后拼接。
- **发音与文本处理**：音素提示、内联或托管发音词典、数字与日期及单位规范化。language 选项影响规范化和时间对齐，不直接改变音色或口音。
- **生成控制**：0.5–2 倍语速、dB 音量、响度规范化、采样变化、延迟模式、分块和前序音频上下文。可选的 quality-guard 功能取决于后端。
- **交付方式**：HTTP 音频、双向 WebSocket 和流式词级对齐快照；支持 WAV/PCM、MP3、Opus。WAV/PCM 为单声道 16 位，采样率最高 44.1 kHz；Opus 使用 48 kHz。

参考：[TTS API](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech)、[情绪控制](https://docs.fish.audio/developer-guide/core-features/emotions)、[Drama 3](https://docs.fish.audio/features/text-to-speech/drama-3)、[流式接口](https://docs.fish.audio/api-reference/endpoint/websocket/tts-live)和[时间戳](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech-stream-with-timestamps)。

表演提示依赖模型理解，不是效果保证。API schema 没有给出统一的文本长度上限，不应将客户端限制写成厂商上限。

## 价格与注意事项

[API 价格](https://docs.fish.audio/developer-guide/models-pricing/pricing-and-rate-limits)以**每百万 UTF-8 输入字节的美元金额**计算，无 API 最低订阅要求：

- S2.1-Pro、S2-Pro 和旧版 S1：**$15**。
- S2.1-Pro Free：**$0**，受公平使用限制。
- Drama 3：**标价 $50**；**上线优惠价 $10，截至 2026 年 11 月 12 日 17:00 UTC**。

字节不等于字符：相同字符数的非 ASCII 文本可能产生更高费用。不能直接与按字符计费的厂商比较。

付费账户并发档位为 5、15 和 50 个请求，分别对应累计预付金额低于 $100、达到 $100 和达到 $1,000；企业并发需协商。免费开发档不保证首音频延迟。区域价格、免费档的具体额度及账户专属条款未核实。
