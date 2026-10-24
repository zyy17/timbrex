# Azure Speech

[English](readme.md) · [Timbrex 配置](../../providers.zh-cn.md) · [已支持的控制](../../capabilities.zh-cn.md)

面向开发者的厂商速查，核对日期：**2026-10-24**。本页介绍厂商提供的功能，不代表 Timbrex 已支持全部能力；当前支持范围见上方链接。

## 适用场景

如果多语言应用需要明确的发音控制、丰富的音色选择和区域部署选项，可以考虑 Azure Speech。标准神经音色和自定义音色面向不同需求；能力是否可用取决于音色、模型和区域。[概览](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/text-to-speech)、[音色目录](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/language-support?tabs=tts)

## 特性与取舍

- **音色系列：**包括标准、多语言神经音色，强调表现力的 DragonHD/Omni，以及预览版 MAI-Voice-2.1/2.1-Flash。MAI Flash 偏重低延迟交互，MAI 偏重长篇语音的表现力。预览功能没有 SLA，官方不建议用于生产环境。[HD 音色](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/high-definition-voices)、[MAI 音色](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/mai-voices)
- **发音和表现控制：**SSML 提供音素、词典、读音替换、数字日期解释和数学表达式朗读。风格、角色、重音和韵律控制依赖具体音色，并非所有音色都支持每种元素。[发音控制](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-pronunciation)、[表现控制](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice)
- **音频和时间信息：**支持多种编码、采样率和增量音频输出，长录音可使用独立的异步批量 API。SDK 可提供时间事件和流式文本输入；流式文本输入不支持 SSML。[REST 音频](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/rest-text-to-speech)、[批量合成](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/batch-synthesis)、[文本流式输入](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/how-to-lower-speech-synthesis-latency#input-text-streaming)
- **品牌音色：**专业音色和个人音色创建都需要资格审批与说话人授权。使用已有自定义音色，与创建或托管音色是不同流程。[自定义音色选项](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/personal-voice-overview)
- **相关产品：**包含有上下文的多说话人合成、声音转换、Voice Live 对话、数字人和视频翻译配音。这些能力需要不同输入或独立流程，不能简单当作单句 TTS 参数。[多说话人](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice#multi-talker-voice-example)、[声音转换](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/voice-conversion)、[Voice Live](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/voice-live)、[数字人](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/text-to-speech-avatar/what-is-text-to-speech-avatar)、[视频配音](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/video-translation-overview)

## 需要注意的限制

实时合成每个请求最多输出 10 分钟，更长的任务使用批量合成。F0 每 60 秒最多 20 次请求，S0 默认每秒 30 次。即使没有达到账号配额，某个区域的音色容量仍可能引起限流。[配额](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-services-quotas-and-limits#text-to-speech-quotas-and-limits-per-resource)

官方 HD 支持表与示例对部分 `express-as` 和 `prosody` 能力存在矛盾，依赖这些控制前应验证具体音色和版本。MAI 专页已描述 2.1 模型，但综合音色目录仍列有部分 2 系列 ID。[HD 支持情况](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/high-definition-voices#supported-and-unsupported-ssml-elements-for-azure-speech-hd-voices)、[MAI 指南](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/mai-voices)

新的长文本任务应使用 Batch synthesis。旧版 Long Audio API 计划于 **2027-04-01** 退役。[生命周期与迁移](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/batch-synthesis)

## 价格

官方页面列出 **F0 每月免费 50 万个 Neural 字符**。付费合成按字符计费，Neural/Neural HD Flash、Neural HD、MAI 和自定义音色分列计价。专业音色训练和托管、个人音色存储、数字人可能产生额外费用；承诺用量套餐并不覆盖所有类型。[官方价格](https://azure.microsoft.com/en-us/pricing/details/speech/)

截至 2026-10-24，可读取的价格页将付费金额显示为 `$-`，本次未核实区域付费单价。预算应以官方计算器中选定区域、币种和协议后的价格为准，不宜把历史报价当作 Azure 的统一价格。
