# 厂商接入

[English](providers.md) · [支持的控制](capabilities.zh-cn.md) · [教程](tutorial.zh-cn.md)

Timbrex 直接调用同步 HTTP API。适配器依据下列官方参考实现，核对日期为 **2026-10-22**。离线测试验证请求和响应契约，尚未验证真实账号兼容性或声音质量。凭证必须属于端点所在区域，并具有所选模型和音色的权限。

## 配置

设置从 YAML 和环境变量集中加载，详见 [配置](configuration.zh-cn.md)。建议通过 shell 或密钥管理器设置凭证。Timbrex **不会**加载 `.env` 文件。不要把密钥写入 `.tbx` 文件或提交到仓库。下表描述环境变量配置方式，也支持等价的 `providers.<id>` 文件字段。

| Provider ID | 必需环境变量 | 默认模型 | 默认音色 |
| --- | --- | --- | --- |
| `openai` | `OPENAI_API_KEY` | `gpt-4o-mini-tts` | `coral` |
| `elevenlabs` | `ELEVENLABS_API_KEY`，另需选择音色 | `eleven_v3` | 无；通过 `ELEVENLABS_VOICE_ID`、`--voice`、front matter 或 cast 映射设置 |
| `minimax` | `MINIMAX_API_KEY` | `speech-2.8-hd` | `English_expressive_narrator` |
| `azure` | `AZURE_SPEECH_KEY`，以及区域或端点 | 由音色决定 | `en-US-JennyNeural` |
| `qwen` | `DASHSCOPE_API_KEY`、`QWEN_ENDPOINT` | `qwen-audio-3.1-tts-flash` | `longanhuan_v3.1` |
| `gemini` | `GEMINI_API_KEY` | `gemini-3.8-flash-tts` | `Kore` |
| `fish` | `FISH_API_KEY`，音色/reference ID | `s2.1-pro` | 无；通过 `FISH_VOICE_ID`、`--voice`、front matter 或角色映射设置 |
| `stub` | 无 | 无 | 无；测试音 |

可选的环境变量默认值：

| 厂商 | 模型 | 音色 | 端点覆盖 |
| --- | --- | --- | --- |
| OpenAI | `OPENAI_MODEL` | `OPENAI_VOICE` | `OPENAI_BASE_URL`（含 `/v1` 的基础 URL） |
| ElevenLabs | `ELEVENLABS_MODEL` | `ELEVENLABS_VOICE_ID` | `ELEVENLABS_BASE_URL`（含 `/v1` 的基础 URL） |
| MiniMax | `MINIMAX_MODEL` | `MINIMAX_VOICE_ID` | `MINIMAX_BASE_URL`（含 `/v1` 的基础 URL） |
| Azure | — | `AZURE_SPEECH_VOICE` | `AZURE_SPEECH_ENDPOINT`（完整请求 URL），否则使用 `AZURE_SPEECH_REGION` |
| Qwen | `QWEN_MODEL` | `QWEN_VOICE` | `QWEN_ENDPOINT`（必填完整请求 URL） |
| Gemini | `GEMINI_MODEL` | `GEMINI_VOICE` | `GEMINI_BASE_URL` (`https://generativelanguage.googleapis.com/v1beta`) |
| Fish | `FISH_MODEL` | `FISH_VOICE_ID` | `FISH_BASE_URL` (`https://api.fish.audio/v1`) |

`--model` 和 `--voice` 覆盖 front matter，后者覆盖环境变量，环境变量覆盖 YAML 配置和内置默认值。cast 音色映射优先于默认音色。`[voice "vendor-id"|text]` 直接选择音色。`[as lin|text]` 使用 Lin 的映射；已知说话人没有映射时仍使用默认音色，未知目标会报错。切换厂商不会自动转换其他厂商的模型或音色 ID。

端点覆盖用于你选择的区域服务或可信兼容网关，API 凭证会发送至该端点。远程服务应使用 HTTPS；HTTP 用于本地测试。请求不会跟随重定向。

## OpenAI

```bash
# 先设置 OPENAI_API_KEY。
./timbrex synth examples/expressive.tbx --provider openai --voice coral -o openai.wav
```

使用 bearer 认证调用 `POST https://api.openai.com/v1/audio/speech`，设置 `response_format: pcm`。表演要求放在独立的 `instructions` 中，台词保留在 `input`。适配器分别限制台词和指令不超过 4096 字符。`tts-1`、`tts-1-hd` 可合成纯文本并控制语速，但拒绝表演和语言指令。

[GPT-4o Mini TTS 模型](https://developers.openai.com/api/docs/models/gpt-4o-mini-tts) 还有 2000 个输入 token 的限制。本地字符限制不等于 token 计数；Timbrex 不会对该模型输入进行分词，也不保证已接受的文本和指令符合其 token 预算。请拆分长片段并保持指令简洁。

OpenAI 已将 GPT-4o Mini TTS 标为弃用。[生命周期公告](https://developers.openai.com/api/docs/deprecations#2026-10-01-text-to-speech-models) 计划于 2027-01-06 下线 `tts-1`、`tts-1-hd` 和带日期的 GPT-4o Mini TTS 快照。推荐的 Realtime 替代方案需要不同的 API 实现；仅更改 `OPENAI_MODEL` 不能完成适配器迁移。

参考：[Create speech](https://developers.openai.com/api/reference/resources/audio/subresources/speech/methods/create)、[TTS 指南](https://developers.openai.com/api/docs/guides/text-to-speech)。发布 AI 生成语音时应遵守厂商的披露要求。

## ElevenLabs

```bash
# 先设置 ELEVENLABS_API_KEY 和 ELEVENLABS_VOICE_ID。
./timbrex synth examples/expressive.tbx --provider elevenlabs -o elevenlabs.wav
```

使用 `xi-api-key` 调用 `POST https://api.elevenlabs.io/v1/text-to-speech/{voice_id}?output_format=pcm_24000`。适配器为 `eleven_v3`、`eleven_v4`、`eleven_v4_turbo` 生成原生 audio tags；两个 v4 模型不支持语速设置，因此会拒绝显式 `speed`、`fast` 和 `slow`。其他模型 ID 支持纯文本与语速，但拒绝表演标签和非语言事件。适配器采用保守的 5000 字符限制，包含生成的标签。语言提示使用主要语言子标签；`eleven_multilingual_v2` 不接受 `language_code`，因此拒绝显式 `lang`。

Eleven v3 的 speed 支持在官方产品文档和工具文档中描述不一致。目前保留原映射，等待真实账户验证；本地接受该设置不代表已确认它会改变生成语音。

参考：[Create speech](https://elevenlabs.io/docs/api-reference/text-to-speech/convert)、[audio tags](https://elevenlabs.io/docs/help-center/product/core-capabilities/text-to-speech/how-do-audio-tags-work-with-eleven-v3-and-v4)、[语速控制](https://elevenlabs.io/docs/eleven-creative/playground/text-to-speech)。Audio tags 描述期望的表演，不保证生成指定音效。

## MiniMax

```bash
# 先设置 MINIMAX_API_KEY。
./timbrex synth examples/expressive.tbx --provider minimax -o minimax.wav
```

使用 bearer 认证调用 `POST https://api.minimax.io/v1/t2a_v2`，请求非流式 hex 输出和 PCM 音频。适配器要求 `base_resp.status_code` 明确表示成功，检查完成状态和 hex 解码，并拒绝可选 `extra_info` 音频元数据中冲突的采样率、声道数、格式或解码后字节数。每段文字必须少于 10000 字符。

默认的 `speech-2.8-hd` 支持已映射的非语言事件。适配器仅在 `speech-2.6-hd` 或 `speech-2.6-turbo` 下支持耳语，且同一片段不能同时使用耳语和另一种情绪。音高要求整数半音。语言提示当前映射 `zh en ja ko fr de es ru pt it ar hi th id vi yue`，`zh-HK` 和 `zh-MO` 映射为粤语。其他提示会在本地报错。这是适配器实现的子集，不代表厂商完整语言覆盖。

参考：[同步语音合成 API](https://platform.minimax.io/docs/api-reference/speech-t2a-http)。

## Azure Speech

```bash
# 先设置 AZURE_SPEECH_KEY。
export AZURE_SPEECH_REGION=eastus
./timbrex synth examples/expressive.tbx --provider azure --voice en-US-JennyNeural -o azure.wav
```

使用订阅密钥和 SSML 调用 `POST https://{region}.tts.speech.microsoft.com/cognitiveservices/v1`，请求 `raw-24khz-16bit-mono-pcm`。文本和属性均进行 XML 转义。音色名称决定模型；指定 `--model` 或 front matter 的 `model:` 会报错。

Style、role 和强调能力依赖所选音色。Azure 可能忽略音色不支持的 style，Timbrex 尚未查询音色能力。适配器只允许每个片段一个 style，冲突时会报错。`[role Boy]` 需要同一作用域中存在 style。根 SSML 语言使用音色名称中可识别的语言区域前缀。显式 `lang` 与该语言区域不同时，会在 `voice` 内生成 `<lang xml:lang="…">`，以表达多语言或口音覆盖；所选音色必须支持目标语言。没有语言区域前缀的自定义音色使用显式 `lang` 作为根默认语言。

Azure REST 会在十分钟处截断音频，且可能仍返回 HTTP 200。片段达到或超过 28,800,000 字节（所请求的 24 kHz 单声道 S16LE PCM 的十分钟长度）时，Timbrex 会报错而不会视为完整结果；请将较长语音拆成较短段落。由于原始 PCM 没有完成标记，恰好十分钟且完整的片段也会被拒绝。该边界检查不能证明更短的音频包含全部请求文字。

参考：[REST API](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/rest-text-to-speech)、[音色、风格、角色和韵律](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice)。

## Qwen

```bash
# 先设置 DASHSCOPE_API_KEY，并替换工作空间占位符。
export QWEN_ENDPOINT='https://YOUR_WORKSPACE_ID.ap-southeast-1.maas.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer'
./timbrex synth examples/expressive.tbx --provider qwen -o qwen.wav
```

此适配器面向 **Qwen-Audio-TTS 3.0/3.1**，不是旧版 Qwen3 multimodal-generation API，也不是本地运行的 Qwen3 权重。支持的模型 ID 为 `qwen-audio-3.0-tts-flash`、`qwen-audio-3.0-tts-plus`、`qwen-audio-3.1-tts-flash` 和 `qwen-audio-3.1-tts-plus`。切换模型时应同时选择兼容音色。

必须设置工作空间端点。新加坡使用 `ap-southeast-1`，北京在同一 URL 结构中使用 `cn-beijing`。请求采用单数形式的 `input.instruction`、PCM 格式和 24000 Hz 采样率。完成响应提供签名音频 URL，Timbrex 下载时**不携带** API 密钥。语言提示接受 `zh en fr de ja ko ru pt th id vi es it ms fil ar`，会移除 locale 后缀。

参考：[HTTP API](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api)、[模型与使用限制](https://www.alibabacloud.com/help/en/model-studio/non-realtime-tts-user-guide)。

## 失败处理与限制

单次 HTTP 请求超时为两分钟，响应限制为 64 MiB；最终 PCM 输出限制为 1 GiB。所有语音片段会在发起请求前验证。运行时错误包含厂商和片段编号。HTTP 错误包含状态码，MiniMax/Qwen API 错误包含厂商错误代码。诊断省略响应正文和签名 URL 的查询参数。

音频响应必须是原始的 24 kHz、单声道、S16LE PCM。媒体类型或明确提供的 `rate`/`channels` 媒体类型参数不符、可识别的音频容器（包括 FLAC 和 MP4）、空数据或字节数为奇数的 PCM 均会报错。HTTP 206 部分响应和中断的响应体均会报错；不会自动重试合成请求。缺失可选格式元数据的响应仍被接受。原始 PCM 字节不能标明采样率或声道数，因此缺少明确元数据时仍依赖厂商按请求格式返回音频；这些检查不能验证声音质量。

渲染器在目标文件旁创建临时文件，成功后才重命名。取消或失败会删除临时输出并保留已有目标文件。父目录必须已存在。先前请求仍可能计费；没有重试、降级、缓存或断点续传。

排错时请对照官方文档检查模型、音色、区域和密钥权限。离线测试通过只证明实现的契约，不证明服务可用性或语音质量。长文本应自行分段，Timbrex 不会按厂商 token 限制自动切块。

## Gemini TTS

```sh
# 须已设置 GEMINI_API_KEY。
./timbrex synth examples/gemini.tbx -o gemini.wav
```

使用 **Google Gemini API**，不是 Google Cloud Text-to-Speech 或 Interactions：通过 `x-goog-api-key` 调用 `POST /v1beta/models/gemini-3.8-flash-tts:generateContent`。初版仅支持该模型及官方列出的 30 个预置音色，默认 `Kore`。自定义/复制音色、创建音色、流式模式和旧模型 schema 不在本适配器范围内。角色音色按片段串行合成，不使用原生对话合成。

台词和 `speechMetadata.style` 分开传递。`fast`/`slow`、表演、情绪和发声控制是近似风格指令；数值 `speed`、`pitch` 和 `volume` 会被拒绝。部分事件映射到原生尖括号标记。显式 `lang` 映射到 `speechConfig.languageCode`，只接受 API 参考列出的 BCP-47 代码（如 `en-US`、`cmn-CN`，不接受 `en`、`zh-CN`）；未指定时由模型自动识别。台词加风格总计限制为 4096 个 Unicode 字符，仅作为本地资源上限，**不是** token 计数；服务端 8192-token 输入预算仍适用。

请求明确指定 `AUDIO_WAV`、`INLINE` 和 24000 Hz。响应必须恰好包含一个 `STOP` 候选和一个内联 `audio/wav` part，且输入/输出未被拦截。额外 part、多个候选、无效 JSON/base64 和 API 错误对象均失败。共用解码器要求完整 RIFF 长度、PCM1、单声道、24 kHz、16-bit、对齐 2、字节率 48000；未知长度 WAV、重复音频块和其他编码会失败，额外 RIFF 元数据块可接受。64 MiB HTTP 限制包含 JSON/base64 开销。本适配器先提取 WAV 内的 PCM，再交给共用渲染器；前述原始 PCM 限制适用于其他适配器。

参考：[当前 TTS 指南](https://ai.google.dev/gemini-api/docs/speech-generation)、[GenerateContent schema](https://ai.google.dev/api/generate-content)、[模型及限制](https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash-tts)。离线样例验证所选 schema 和拒绝行为，尚未进行真实账号或声学验证。

## Fish Audio

```sh
# 须已设置 FISH_API_KEY 和属于你自己的 FISH_VOICE_ID。
./timbrex synth examples/fish.tbx -o fish.wav
```

使用 bearer 认证、`model: s2.1-pro` 请求头、JSON 和显式单说话人 `reference_id` 调用 `POST https://api.fish.audio/v1/tts`。初版仅支持 `s2.1-pro`；由于服务端会对未知模型请求头静默回退，未知及其他模型均在本地拒绝。没有默认账号音色。请用 `FISH_VOICE_ID`、`--voice`、front matter 或 `fish` 角色映射指定你有权访问的音色。适配器不上传参考音频、不创建克隆，也不使用原生多说话人请求；角色音色按片段串行合成。

语速映射为文档中的 0.5–2 倍率，音量直接传入 dB。音量采用本地 −60…+20 dB 范围，不宣称是厂商上限。明确启用响度规范化，明确关闭文本规范化，以避免 API 额外改写数字、日期和单位；发音仍取决于模型。显式 `lang` 会失败：Fish 的 `language` 只影响规范化/时间戳对齐，不能指定发声语言或口音。部分 S2 情绪和事件通过方括号标记传入；自由表演指令、数值情绪强度、音高和不支持的控制会在任何 HTTP 调用前失败。文本连同生成标记采用本地 4096 字符限制，不宣称是厂商最大值。

请求指定 `format: wav` 和 `sample_rate: 24000`。有界解码器遍历 RIFF chunk，不直接删除固定 44 字节头；只接受完整的 PCM1、单声道、24 kHz、16-bit WAV，且对齐、字节率和长度必须一致。成功 HTTP 响应内的 JSON 错误、截断/未知长度 WAV、重复音频块和其他格式均失败。本适配器先提取 WAV 内的 PCM 再渲染；前述原始 PCM 限制适用于其他适配器。有效 WAV 不能证明模型读出了每个字。官方 HTTP 错误包括 400、401、402、403、404、429 和 5xx；失败时不重试，诊断也不包含响应正文。

参考：[TTS 接口](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech)、[OpenAPI schema](https://docs.fish.audio/api-reference/openapi.json)、[错误码](https://docs.fish.audio/api-reference/errors)、[模型选择](https://docs.fish.audio/developer-guide/models-pricing/choosing-a-model)、[S2 情绪与事件标记](https://docs.fish.audio/developer-guide/core-features/emotions)。离线样例覆盖请求和拒绝路径；真实账号访问、完整 WAV 互操作和声学效果尚未验证，流式哨兵长度会明确拒绝而不是猜测。
