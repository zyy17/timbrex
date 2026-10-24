# Qwen / 阿里云百炼 TTS

[English](readme.md) · [Timbrex 配置](../../providers.zh-cn.md) · [已支持的控制](../../capabilities.zh-cn.md)

面向开发者的厂商速查，核对日期：**2026-10-24**。本页介绍厂商提供的功能，不代表 Timbrex 已支持全部能力；当前支持范围见上方链接。

Qwen-Audio-TTS 提供多语言预置及自定义音色、表演指令、发音控制和流式输出。选型时先区分 API 家族：**Qwen Audio 3.x TTS、Qwen3-TTS 与 TTS-Next 不是同一套请求协议**。

## 模型与地域

当前标准语音模型包括 `qwen-audio-3.1-tts-flash`、`qwen-audio-3.1-tts-plus`、`qwen-audio-3.0-tts-flash` 和 `qwen-audio-3.0-tts-plus`，四者均提供北京和新加坡合成端点。音色的可用地域和能力随模型变化，请查阅[模型指南](https://www.alibabacloud.com/help/en/model-studio/non-realtime-tts-user-guide)及[音色目录](https://help.aliyun.com/en/model-studio/qwen-audio-tts-voice-list)。

## 语音能力

- 16 种语言提示、方言音色、自然语言情绪/风格/角色指令、23 种行内表演控制及 7 种非语言声音。指令可用于系统、复刻和设计音色，但方言能力取决于音色；设计音色不支持方言合成。[能力指南](https://www.alibabacloud.com/help/en/model-studio/non-realtime-tts-user-guide)
- 语速和音高范围为 0.5–2，音量为 0–100；支持随机种子、读音热修正和文本替换。输出 MP3、PCM、WAV 或 Opus，采样率覆盖 8–48 kHz，具体组合受编码限制。可选 AIGC 溯源标签不支持 PCM。[HTTP 参数](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api)
- 部分系统音色及复刻音色支持 SSML，包括停顿、别名、中文拼音/英文 CMU 音素、特殊读法、声音事件、音色/音效和背景音乐，并非所有音色都支持。详细 LaTeX 指南仅列出 CosyVoice，因此这里不认定 Qwen Audio 支持 LaTeX。[SSML 指南](https://help.aliyun.com/en/model-studio/ssml-latex-user-guide)
- 整段文本 HTTP 返回可下载结果，SSE 逐步返回音频；字词时间戳仅支持流式模式，且取决于音色。下载地址有效期为 24 小时。WebSocket 增量文本合成是另一种传输方式。[HTTP](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api) · [WebSocket](https://www.alibabacloud.com/help/en/model-studio/realtime-tts-user-guide)

当前 HTTP 参考未给出这四个模型明确的整段文本字符上限，不能套用 Qwen3 的 600 字符限制。标准 Flash/Plus 协议未记录原生多说话人对话或前后文文本参数。

## 相关但独立的产品

- **声音复刻/设计：**独立的 `voice-enrollment` 流程创建与目标模型绑定的音色。目前创建免费，语音合成收费；每个账号的复刻和设计音色共享 1,000 个配额。设计文档列出北京地域；复刻文档的新加坡列表未列出 3.1 Flash，使用这一组合前应确认。[复刻](https://www.alibabacloud.com/help/en/model-studio/voice-cloning-user-guide) · [设计](https://www.alibabacloud.com/help/en/model-studio/voice-design-user-guide)
- **Qwen3-TTS：**独立的 API 和音色目录。无日期的 Flash 别名仍有当前文档，但官方通知已将多种日期快照、instruct、VC 和 VD 版本的计划退役时间设为 2026-10-10。应核对具体 ID，不能直接照搬旧示例。[API](https://www.alibabacloud.com/help/en/model-studio/qwen-tts-api) · [退役通知](https://www.alibabacloud.com/en/notice/model_studio_notice_of_retirement_for_selected_legacy_longtail_models_7d4?_p_lc=1)
- **TTS-Next：**通过提示词生成语音、对话及音效环境，最多使用 3 段参考音频，不使用标准音色 ID。支持中英文，播客最长 4 分钟，其他音频最长 2 分钟，属于独立的生成产品。[参考](https://www.alibabacloud.com/help/en/model-studio/audio-generation-api)

## 标准价格

以下价格核对于 2026-10-24。**账号所属计费目录和部署地域都影响价格**；列出的是两种常见组合，不是汇率换算：

| 模型 | 计费单位 | 国际站目录，新加坡（USD） | 中国站目录，北京（CNY） |
| --- | --- | --- | --- |
| 3.1 Flash | 每百万输入 / 输出 Token | $0.23 / $1.87 | ¥1.50 / ¥12 |
| 3.1 Plus | 每万输入字符 | $0.30 | ¥1.40 |
| 3.0 Plus | 每万输入字符 | $0.20 | ¥1.40 |
| 3.0 Flash | 每万输入字符 | $0.15 | ¥1.00 |

按字符计费时，汉字计为 2 个字符，其他字符计为 1 个；SSML 标签不计费。符合资格的试用额度有效期为 90 天，国际站目录仅在新加坡提供，中国站目录仅在北京提供。其他地域费率、优惠及账号资格应查对应目录。[国际站价格](https://www.alibabacloud.com/help/en/model-studio/model-pricing) · [中国站价格](https://help.aliyun.com/zh/model-studio/model-pricing)

TTS-Next 的国际站价格按每百万输入/输出 Token 计算：新加坡为 $0.93/$1.87，北京为 $0.848/$1.696。不能用上述字符费率估算按 Token 计费的音频。[模型价格](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-3-1-tts-next)
