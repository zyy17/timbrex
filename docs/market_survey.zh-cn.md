# 厂商接入调研

[English](market_survey.md)

核对日期：**2026-10-22**。本文记录与 Timbrex 有关的 API 设计差异，不是质量基准、价格比较或法律评估。未进行真实试听研究，部署时应向厂商确认模型可用性和服务条款。

| 服务 | 官方控制方式 | 对 Timbrex 的影响 |
| --- | --- | --- |
| [OpenAI](https://developers.openai.com/api/docs/guides/text-to-speech) | 语音端点提供模型、音色、语速和依赖模型的指令 | 将表演指令与台词输入分开 |
| [ElevenLabs](https://elevenlabs.io/docs/api-reference/text-to-speech/convert) | 音色专属端点、模型选择、音色设置、依赖模型的 audio tags | 将支持的表演与事件转为原生标签，保留模型限制 |
| [MiniMax](https://platform.minimax.io/docs/api-reference/speech-t2a-http) | 音色设置、离散情绪、PCM/hex 响应、依赖模型的语气词 | 映射明确的能力子集，同时检查 HTTP 与 API 完成状态 |
| [Azure Speech](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice) | SSML 音色、韵律、style 和 role，支持情况取决于音色 | 转义 XML，区分语法支持与实际音色能力 |
| [Qwen-Audio-TTS](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api) | 工作空间端点、指令与数值控制、可下载音频 | 要求正确的工作空间端点，区分认证合成请求与签名下载 |
| [Gemini TTS](https://ai.google.dev/gemini-api/docs/speech-generation) | 结构化 `speech_metadata.style`、行内发声标记、WAV 或原始 PCM | 将回合级风格与台词分开；在共享 PCM 渲染前解开 WAV |

这些适配器均可得到兼容 PCM，因此能用小型顺序 WAV 渲染器实现，而无需拼接压缩音频或引入转码依赖。这是实现选择，不意味着厂商的其他音频格式不合适。

真实评估仍需回答三个问题：文档请求能否在目标账号和区域工作，模型遵循表演控制的稳定程度，以及片段边界的可听程度。有效评估应记录精确模型/音色 ID、源脚本、请求参数、日期、失败情况和听评标准，并区分 API 接受请求与实际表演质量。

音色克隆、原生多角色生成、流式、对齐和音效生成可能属于独立厂商产品。厂商提供这些产品，不代表 Timbrex 已集成。已实现行为见[当前能力](capabilities.zh-cn.md)与[接入配置](providers.zh-cn.md)。

## 相关开源项目

以下项目分别与 Timbrex 的不同层次重叠。比较依据是其公开仓库和文档，并非运行或试听评估。

| 项目 | 相关工作 | 值得研究的部分 |
| --- | --- | --- |
| [Speech Markdown](https://github.com/speechmarkdown/speechmarkdown-js)（MIT） | 语音标记语法、解析器和平台 formatter | 写作语法与目标平台转换 |
| [SSMD](https://github.com/buchwandler/ssmd)（MIT） | Markdown 风格的音色、发音和韵律标注，以及结构解析和迁移 | 作用域、诊断与语义保留 |
| [Speech SDK](https://github.com/Jellypod-Inc/speech-sdk)（Apache-2.0） | TypeScript 合成、流式与多角色编排，区分原生对话和拼接路径 | 渲染决策与显式能力约束 |
| [Podcastfy](https://github.com/souzatharsis/podcastfy)（Apache-2.0） | 采用多个语音后端的内容转播客工具 | 端到端写作与音频工作流 |

仅靠紧凑语法和几个适配器不足以形成 Timbrex 的差异化。它希望提供的是可读的表演脚本，明确表达角色、局部指令和非语言事件，并诚实映射到目标厂商。在重复实现已有功能前，值得评估与现有标记及合成工具互操作。
