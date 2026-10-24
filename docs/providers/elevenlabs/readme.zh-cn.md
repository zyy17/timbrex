# ElevenLabs

[English](readme.md) · [Timbrex 配置](../../providers.zh-cn.md) · [已支持的控制](../../capabilities.zh-cn.md)

面向开发者的厂商速查，核对日期：**2026-10-24**。本页仅介绍厂商的 TTS 能力，不代表 Timbrex 已全部支持；当前支持范围见上方链接。

## 快速了解

ElevenLabs 提供富有表现力的语音、丰富的音色资源和多种低延迟模型，适合旁白、角色配音、多语言内容与实时语音交互。音色选择会明显影响结果；情绪标签和 seed 都不能保证每次生成完全相同。

| 模型 | 主要特点 | 官方文本上限 |
| --- | --- | --- |
| `eleven_v4` | 语音表现力较强，支持 90 多种语言 | 10,000 个字符 |
| `eleven_v4_turbo` | 面向实时交互的 v4 系列，模型推理延迟中位数约 100 ms | 需核对所选传输方式 |
| `eleven_v3` / `eleven_v3_conversational` | 上一代表现力模型，支持 70 多种语言；Conversational 面向实时使用 | v3 为 5,000 个字符 |
| `eleven_multilingual_v2` | 长文本表现较稳定，支持 29 种语言 | 10,000 个字符 |
| `eleven_flash_v2_5` / `eleven_flash_v2` | 低延迟语音；分别支持 32 种语言和英语 | 40,000 / 30,000 个字符 |

官方仍列出 Turbo v2/v2.5，但建议改用 Flash。模型延迟不含网络和应用开销。当前可用性见[模型目录](https://elevenlabs.io/docs/overview/models)和[产品限制](https://elevenlabs.io/docs/eleven-creative/playground/text-to-speech)。

## 能力范围

- **表演控制**：在支持该功能的模型中使用音频标签、情绪与非语言反应；可按请求调整稳定性、相似度及其他模型相关音色参数。
- **发音**：发音词典、文本规范化和模型专属的音素或 IPA 引导。v4 支持斜杠包围的 IPA；旧模型对 SSML 的支持有限。
- **连续性**：可利用前后文本或请求上下文连接多次生成。seed 仅提供尽力而为的可复现性。
- **交付方式**：完整音频、HTTP 流式输出、字符时间戳、增量文本 WebSocket 和多上下文会话。格式包括 MP3、PCM、WAV、Opus 及电话音频编码，不同传输方式的格式范围不同。
- **音频效果**：滤镜、距离感、空间效果和声像定位；立体声效果需要兼容格式。
- **多角色对话**：独立的 Text to Dialogue API 可同时生成多个音色。当前接口文档建议每次请求的合计文本不超过 2,000 个字符，即使底层模型上限更大。

参考：[语音 API](https://elevenlabs.io/docs/api-reference/text-to-speech/convert)、[提示指南](https://elevenlabs.io/docs/overview/capabilities/text-to-speech/best-practices)、[音频效果](https://elevenlabs.io/docs/overview/capabilities/text-to-speech)、[时间戳](https://elevenlabs.io/docs/api-reference/text-to-speech/convert-with-timestamps)和[对话 API](https://elevenlabs.io/docs/api-reference/text-to-dialogue/convert)。

## 价格与注意事项

[ElevenAPI 价格](https://elevenlabs.io/pricing/api)以美元计价，不含税。当前公开价格为：

- **每 1,000 个输入字符 $0.08**：v4、v3、Multilingual v2。
- **每 1,000 个输入字符 $0.04**：v4 Turbo、v3 Conversational、Flash/Turbo。

支持按量付费和订阅，月付套餐从 $6 起。包含用量、优惠、并发、商业使用权限和格式权限取决于套餐。按分钟展示的数字只是估算，不是计费单位。上述为 API 价格，不是 Creative 套餐积分换算。

192 kbps MP3 需要 Creator 或更高套餐；44.1 kHz PCM/WAV 需要 Pro 或更高套餐。企业选项包括零保留模式和[区域环境](https://elevenlabs.io/docs/overview/administration/data-residency)，但区域环境使用独立账户与资源，需核对可用性。

**文档分歧**：官方资料对 v3 是否支持语速控制，以及 REST 语速范围的描述不一致。依赖该功能前应核实所选模型；v4 不提供通常的 speed/style 控制。[产品指南](https://elevenlabs.io/docs/eleven-creative/playground/text-to-speech)、[提示词指南](https://elevenlabs.io/docs/overview/capabilities/text-to-speech/best-practices)、[REST 参数](https://elevenlabs.io/docs/api-reference/text-to-speech/convert)。
