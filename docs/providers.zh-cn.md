# 厂商接入

[English](providers.md) · [支持的控制](capabilities.zh-cn.md) · [教程](tutorial.zh-cn.md)

Timbrex 直接调用同步 HTTP API。适配器依据下列官方参考实现，核对日期为 **2026-10-22**。离线测试验证请求和响应契约，尚未验证真实账号兼容性或声音质量。凭证必须属于端点所在区域，并具有所选模型和音色的权限。

## 配置

设置来自 YAML 和环境变量。见 [配置](configuration.zh-cn.md)。将凭证放在环境变量或密钥管理器中。Timbrex 不加载 `.env` 文件。不要把密钥写入 `.tbx` 文件或提交到仓库。各厂商小节列出所需的环境变量。等价的 `providers.<id>` 文件字段同样有效。

[OpenAI](#openai) · [ElevenLabs](#elevenlabs) · [MiniMax](#minimax) · [Azure Speech](#azure-speech) · [Qwen](#qwen) · [Gemini TTS](#gemini-tts) · [Fish Audio](#fish-audio)

`stub` 不需要环境变量，也没有模型或音色；它只生成测试音。

`--model` 和 `--voice` 覆盖 front matter，后者覆盖环境变量，环境变量覆盖 YAML 配置和内置默认值。cast 音色映射优先于默认音色。`[voice "vendor-id"|text]` 直接选择音色。`[as lin|text]` 使用 Lin 的映射；已知说话人没有映射时仍使用默认音色，未知目标会报错。切换厂商不会自动转换其他厂商的模型或音色 ID。

端点覆盖用于指定的区域服务或可信兼容网关。API 凭证会发送至该端点。远程服务应使用 HTTPS；HTTP 用于本地测试。请求不会跟随重定向。

## OpenAI

- Provider ID：`openai`。
- 环境变量配置：`OPENAI_API_KEY`。
- 默认模型：`gpt-4o-mini-tts`。
- 默认音色：`coral`。
- 可选的模型默认值：`OPENAI_MODEL`。
- 可选的音色默认值：`OPENAI_VOICE`。
- 端点覆盖：`OPENAI_BASE_URL`（含 `/v1` 的基础 URL）。

```bash
# 先设置 OPENAI_API_KEY。
./timbrex synth examples/expressive.tbx --provider openai --voice coral -o openai.wav
```

使用 bearer 认证调用 `POST https://api.openai.com/v1/audio/speech`，设置 `response_format: pcm`。表演要求放在独立的 `instructions` 中，台词保留在 `input`。适配器分别限制台词和指令不超过 4096 字符。`tts-1`、`tts-1-hd` 可合成纯文本并控制语速，但拒绝表演和语言指令。

[GPT-4o Mini TTS 模型](https://developers.openai.com/api/docs/models/gpt-4o-mini-tts) 还有 2000 个输入 token 的限制。本地字符限制不等于 token 计数；Timbrex 不会对该模型输入进行分词，也不保证已接受的文本和指令符合其 token 预算。请拆分长片段并保持指令简洁。

OpenAI 已将 GPT-4o Mini TTS 标为弃用。[生命周期公告](https://developers.openai.com/api/docs/deprecations#2026-10-01-text-to-speech-models) 计划于 2027-01-06 下线 `tts-1`、`tts-1-hd` 和带日期的 GPT-4o Mini TTS 快照。推荐的 Realtime 替代方案需要不同的 API 实现；仅更改 `OPENAI_MODEL` 不能完成适配器迁移。

参考：[Create speech](https://developers.openai.com/api/reference/resources/audio/subresources/speech/methods/create)、[TTS 指南](https://developers.openai.com/api/docs/guides/text-to-speech)。发布 AI 生成语音时应遵守厂商的披露要求。

## ElevenLabs

- Provider ID：`elevenlabs`。
- 环境变量配置：`ELEVENLABS_API_KEY`，另需选择音色。
- 默认模型：`eleven_v3`。
- 默认音色：无；通过 `ELEVENLABS_VOICE_ID`、`--voice`、front matter 或 cast 映射设置。
- 可选的模型默认值：`ELEVENLABS_MODEL`。
- 可选的音色默认值：`ELEVENLABS_VOICE_ID`。
- 端点覆盖：`ELEVENLABS_BASE_URL`（含 `/v1` 的基础 URL）。

```bash
# 先设置 ELEVENLABS_API_KEY 和 ELEVENLABS_VOICE_ID。
./timbrex synth examples/expressive.tbx --provider elevenlabs -o elevenlabs.wav
```

使用 `xi-api-key` 调用 `POST https://api.elevenlabs.io/v1/text-to-speech/{voice_id}?output_format=pcm_24000`。适配器为 `eleven_v3`、`eleven_v4`、`eleven_v4_turbo` 生成原生 audio tags；两个 v4 模型不支持语速设置，因此会拒绝显式 `speed`、`fast` 和 `slow`。其他模型 ID 支持纯文本与语速，但拒绝表演标签和非语言事件。适配器采用保守的 5000 字符限制，包含生成的标签。语言提示会转为小写并提取主要语言子标签；主要子标签为空时报错。`eleven_multilingual_v2` 不接受 `language_code`，因此拒绝显式 `lang`。仅支持英语的 `eleven_flash_v2` 和 `eleven_turbo_v2` 只接受 `en`，包括 `en-US` 等 locale 形式。适配器不会查询其他模型支持的语言列表；ElevenLabs 可能忽略所选模型不支持的语言代码。

Eleven v3 的 speed 支持在官方产品文档和工具文档中描述不一致。目前保留原映射，等待真实账户验证；本地接受该设置不代表已确认它会改变生成语音。

参考：[Create speech](https://elevenlabs.io/docs/api-reference/text-to-speech/convert)、[模型语言支持](https://elevenlabs.io/docs/help-center/technical/what-models-do-you-offer-and-what-is-the-difference-between-them)、[audio tags](https://elevenlabs.io/docs/help-center/product/core-capabilities/text-to-speech/how-do-audio-tags-work-with-eleven-v3-and-v4)、[语速控制](https://elevenlabs.io/docs/eleven-creative/playground/text-to-speech)。Audio tags 描述期望的表演，不保证生成指定音效。

## MiniMax

- Provider ID：`minimax`。
- 环境变量配置：`MINIMAX_API_KEY`。
- 默认模型：`speech-2.8-hd`。
- 默认音色：`English_expressive_narrator`。
- 可选的模型默认值：`MINIMAX_MODEL`。
- 可选的音色默认值：`MINIMAX_VOICE_ID`。
- 端点覆盖：`MINIMAX_BASE_URL`（含 `/v1` 的基础 URL）。

```bash
# 先设置 MINIMAX_API_KEY。
./timbrex synth examples/expressive.tbx --provider minimax -o minimax.wav
```

使用 bearer 认证调用 `POST https://api.minimax.io/v1/t2a_v2`，请求非流式 hex 输出和 PCM 音频。适配器要求 `base_resp.status_code` 明确表示成功，检查完成状态和 hex 解码，并拒绝可选 `extra_info` 音频元数据中冲突的采样率、声道数、格式或解码后字节数。每段文字必须少于 10000 字符。

默认的 `speech-2.8-hd` 支持已映射的非语言事件。`[whisper]` 和 `[style fluent]` 要求使用 `speech-2.6-hd` 或 `speech-2.6-turbo`。它们占用情绪字段，同一片段中不能彼此组合，也不能与另一种情绪组合。其他 style 不支持。音高要求整数半音。

语言提示覆盖文档列出的全部 40 种语言/方言，见[主要语言代码的具体映射](capabilities.zh-cn.md#具体映射)。显式提示要求使用 `speech-2.8-hd`、`speech-2.8-turbo`、`speech-2.6-hd`、`speech-2.6-turbo`、`speech-02-hd`、`speech-02-turbo`、`speech-01-hd` 或 `speech-01-turbo`。波斯语（`fa`）、菲律宾语（`fil`）和泰米尔语（`ta`）要求 2.6 或 2.8 模型，带 locale 后缀时也适用此限制。未指定 `lang` 时仍省略 `language_boost`；显式 `lang: auto` 或 `[lang auto|Text]` 发送 `"auto"`。无语言提示的请求仍允许原样传递模型 ID；未知模型与显式语言提示组合时会在本地失败。

提示不区分大小写，接受 `language[-Script][-REGION]`：已映射的主要语言代码，可选的四字母 script，以及可选的两字母或三数字 region。Script 和 region 后缀会被忽略，不保证口音；适配器只检查其形式，不核验注册表。仅保留完全匹配的旧别名 `zh-HK` 和 `zh-MO` 来选择粤语。其他 `zh` 形式（含 `zh-Hant-HK`）仍映射为 Chinese；明确指定粤语请用 `yue` 或 `yue-Hant-HK`。`no` 映射为 Norwegian，`nn` 映射为 Nynorsk。不会把 `nb` 和 `tl` 推断为 Norwegian 和 Filipino 的别名，也不会将 `UA` 或 `PH` 等国家代码当作语言提示。

未知代码、`en-foo` 等格式不符的提示，以及 extlang、variant、extension、private-use 形式，都会在任何合成请求前失败。这有意收紧了旧版只提取前缀的处理方式，避免格式错误的后缀被静默接受。请改用支持的主要语言代码或上述简单 locale 形式。

参考：[同步语音合成 API](https://platform.minimax.io/docs/api-reference/speech-t2a-http)。

## Azure Speech

- Provider ID：`azure`。
- 环境变量配置：`AZURE_SPEECH_KEY`，以及区域或端点。
- 默认模型：由音色决定。
- 默认音色：`en-US-JennyNeural`。
- 可选的模型默认值：—。
- 可选的音色默认值：`AZURE_SPEECH_VOICE`。
- 端点覆盖：`AZURE_SPEECH_ENDPOINT`（完整请求 URL），否则使用 `AZURE_SPEECH_REGION`。

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

- Provider ID：`qwen`。
- 环境变量配置：`DASHSCOPE_API_KEY`、`QWEN_ENDPOINT`。
- 默认模型：`qwen-audio-3.1-tts-flash`。
- 默认音色：`longanhuan_v3.1`。
- 可选的模型默认值：`QWEN_MODEL`。
- 可选的音色默认值：`QWEN_VOICE`。
- 端点覆盖：`QWEN_ENDPOINT`（必填完整请求 URL）。

```bash
# 先设置 DASHSCOPE_API_KEY，并替换工作空间占位符。
export QWEN_ENDPOINT='https://YOUR_WORKSPACE_ID.ap-southeast-1.maas.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer'
./timbrex synth examples/expressive.tbx --provider qwen -o qwen.wav
```

此适配器面向 **Qwen-Audio-TTS 3.0/3.1**，不是旧版 Qwen3 multimodal-generation API，也不是本地运行的 Qwen3 权重。支持的模型 ID 为 `qwen-audio-3.0-tts-flash`、`qwen-audio-3.0-tts-plus`、`qwen-audio-3.1-tts-flash` 和 `qwen-audio-3.1-tts-plus`。切换模型时应同时选择兼容音色。

必须设置工作空间端点。新加坡使用 `ap-southeast-1`，北京在同一 URL 结构中使用 `cn-beijing`。请求采用单数形式的 `input.instruction`、PCM 格式和 24000 Hz 采样率。完成响应提供签名音频 URL，Timbrex 下载时**不携带** API 密钥。语言提示接受 `zh en fr de ja ko ru pt th id vi es it ms fil ar`，会移除 locale 后缀。

参考：[HTTP API](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api)、[模型与使用限制](https://www.alibabacloud.com/help/en/model-studio/non-realtime-tts-user-guide)。

## Gemini TTS

- Provider ID：`gemini`。
- 环境变量配置：`GEMINI_API_KEY`。
- 默认模型：`gemini-3.8-flash-tts`。
- 默认音色：`Kore`。
- 可选的模型默认值：`GEMINI_MODEL`。
- 可选的音色默认值：`GEMINI_VOICE`。
- 端点覆盖：`GEMINI_BASE_URL` (`https://generativelanguage.googleapis.com/v1beta`)。

```bash
# 须已设置 GEMINI_API_KEY。
./timbrex synth examples/gemini.tbx -o gemini.wav
```

使用 **Google Gemini API**，不是 Google Cloud Text-to-Speech 或 Interactions：通过 `x-goog-api-key` 调用 `POST /v1beta/models/gemini-3.8-flash-tts:generateContent`。初版仅支持该模型及官方列出的 30 个预置音色，默认 `Kore`。自定义/复制音色、创建音色、流式模式和旧模型 schema 不在本适配器范围内。各角色使用独立的片段请求，不使用原生对话合成；可选并发也不会合并这些请求。

台词和 `speechMetadata.style` 分开传递。`fast`/`slow`、表演、情绪和发声控制是近似风格指令；数值 `speed`、`pitch` 和 `volume` 会被拒绝。部分事件映射到原生尖括号标记。显式 `lang` 映射到 `speechConfig.languageCode`，只接受 API 参考列出的 BCP-47 代码（如 `en-US`、`cmn-CN`，不接受 `en`、`zh-CN`）；未指定时由模型自动识别。台词加风格总计限制为 4096 个 Unicode 字符，仅作为本地资源上限，**不是** token 计数；服务端 8192-token 输入预算仍适用。

请求明确指定 `AUDIO_WAV`、`INLINE` 和 24000 Hz。响应必须恰好包含一个 `STOP` 候选和一个内联 `audio/wav` part，且输入/输出未被拦截。额外 part、多个候选、无效 JSON/base64 和 API 错误对象均失败。音频可以附带文档规定的非载荷元数据，但带有 thought 标记的音频和额外载荷会被拒绝。64 MiB HTTP 限制包含 JSON/base64 开销。WAV 解码遵循下文的共用格式检查。

参考：[GenerateContent TTS 指南](https://ai.google.dev/gemini-api/docs/generate-content/speech-generation)、[GenerateContent schema](https://ai.google.dev/api/generate-content)、[模型及限制](https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash-tts)。[当前产品 TTS 指南](https://ai.google.dev/gemini-api/docs/speech-generation) 以 Interactions API 为主，本适配器不调用该接口。离线样例验证所选 schema 和拒绝行为，尚未进行真实账号或声学验证。

## Fish Audio

- Provider ID：`fish`。
- 环境变量配置：`FISH_API_KEY`，音色/reference ID。
- 默认模型：`s2.1-pro`。
- 默认音色：无；通过 `FISH_VOICE_ID`、`--voice`、front matter 或角色映射设置。
- 可选的模型默认值：`FISH_MODEL`。
- 可选的音色默认值：`FISH_VOICE_ID`。
- 端点覆盖：`FISH_BASE_URL` (`https://api.fish.audio/v1`)。

```bash
# 须已设置 FISH_API_KEY 和 FISH_VOICE_ID。
./timbrex synth examples/fish.tbx -o fish.wav
```

使用 bearer 认证、`model: s2.1-pro` 请求头、JSON 和显式单说话人 `reference_id` 调用 `POST https://api.fish.audio/v1/tts`。初版仅支持 `s2.1-pro`；由于服务端会对未知模型请求头静默回退，未知及其他模型均在本地拒绝。没有默认账号音色。使用 `FISH_VOICE_ID`、`--voice`、front matter 或 `fish` 角色映射指定有权访问的音色。适配器不上传参考音频、不创建克隆，也不使用原生多说话人请求；各角色使用独立的片段请求，启用并发时也不例外。

语速映射为文档中的 0.5–2 倍率，音量直接传入 dB。音量采用本地 −60…+20 dB 范围，不宣称是厂商上限。明确启用响度规范化，明确关闭文本规范化，以避免 API 额外改写数字、日期和单位；发音仍取决于模型。显式 `lang` 会失败：Fish 的 `language` 只影响规范化/时间戳对齐，不能指定发声语言或口音。部分 S2 情绪和事件通过方括号标记传入；自由表演指令、数值情绪强度、音高和不支持的控制会在任何 HTTP 调用前失败。文本连同生成标记采用本地 4096 字符限制，不宣称是厂商最大值。

请求指定 `format: wav` 和 `sample_rate: 24000`。WAV 解码遵循下文的共用格式检查。即使 HTTP 状态成功，响应中的 JSON 错误也会导致失败。有效 WAV 不能证明模型读出了每个字。官方 HTTP 错误包括 400、401、402、403、404、429 和 5xx；失败时不重试，诊断也不包含响应正文。

参考：[TTS 接口](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech)、[OpenAPI schema](https://docs.fish.audio/api-reference/openapi.json)、[错误码](https://docs.fish.audio/api-reference/errors)、[模型选择](https://docs.fish.audio/developer-guide/models-pricing/choosing-a-model)、[S2 情绪与事件标记](https://docs.fish.audio/developer-guide/core-features/emotions)。离线样例覆盖请求和拒绝路径；真实账号访问、完整 WAV 互操作和声学效果尚未验证，不接受用特殊长度值表示尚未结束的流式 WAV。

## 并发生成

真实厂商支持 `synth --concurrency N`，默认值为 `1`（串行），允许范围是 `1`–`8`。确认账号允许同时发起请求后，可先从 `2` 开始尝试。这是本地安全上限，不代表每个厂商、套餐、区域或模型都接受八个并发请求。假的 `stub` provider 会拒绝大于 `1` 的值。该选项仅通过 CLI 设置，没有配置文件、front matter 或环境变量对应项。

```bash
# 先配置 OpenAI 凭证；此命令可能产生费用。
./timbrex synth examples/dialogue.tbx --provider openai --concurrency 2 -o dialogue.wav
```

所有语音请求都会在首次 HTTP 调用前完成准备与验证。并发只让这些已准备的片段任务并行进行，不拆分或合并片段，不改变请求文本、音色或控制，也不增加重试。无论完成顺序如何，最终 WAV 都按脚本顺序排列解码后的语音与本地停顿。停顿不发起网络请求。Qwen 的合成请求及后续音频下载属于同一个片段任务。

正在执行的任务数和尚未写入的片段窗口都不超过 `N`。前面的慢片段会阻止窗口继续推进，而不是让整份脚本的后续 PCM 不断积压。在内存中等待的已完成 PCM 数据长度受“窗口大小 × 最大解码片段”约束（真实厂商每个响应最多 64 MiB），但这不是所占堆内存或进程内存预算。HTTP 缓冲、JSON/hex/base64 解码和 WAV 底层缓冲可能与 PCM 同时存在，已准备请求以及源码、AST、IR 也会留在内存中。因此，提高并发数可能明显增加内存占用。

记录到第一个合成、解码或写入错误后，渲染器停止接收新任务，取消进行中的工作，等待已启动任务退出，再清理临时输出。触发取消的原始错误会被保留，不被其他任务的取消错误替代；同时发生的错误不保证按脚本顺序报告。HTTP 429 等失败不会重试，也不会自动降低并发数。取消无法撤销服务已接受的请求：已完成或结果不明的在途请求仍可能计费。在发布前失败时，已有输出文件保持不变。

离线测试通过模拟响应验证请求内容、顺序、边界与清理，不能证明真实服务的提速幅度或语音逐字节一致。即使请求内容不变，厂商也可能在不同运行中生成不同的声音。

## 失败处理与限制

适配器会在第一次请求前检查全部语音片段，发现不支持的控制和无效的本地设置。这不验证账号权限，也不保证服务一定接受请求。Timbrex 不会按厂商的 token 限制自动拆分文本；较长的台词应分成较短段落。

单次 HTTP 请求超时为两分钟，响应限制为 64 MiB。已知内容长度超过该限制时，不读取响应体即拒绝响应。长度未知或自动解压的响应体仍在读取时受同一上限约束。最终 PCM 输出限制为 1 GiB。运行时错误包含厂商和片段编号。HTTP 错误包含状态码，MiniMax 和 Qwen 的业务错误包含厂商错误代码。诊断不包含响应正文或签名 URL 的查询参数。HTTP 206 部分响应和中断的响应体会报错。请求不会自动重试。

### 音频格式检查

OpenAI、ElevenLabs、MiniMax、Azure 和 Qwen 请求原始的 24 kHz、单声道、S16LE PCM。适配器拒绝不符的媒体类型，以及明确提供但不符的 `rate`/`channels` 元数据。渲染器还会拒绝可识别的音频容器、空 PCM 和奇数字节数。可选元数据缺失时仍接受响应：原始 PCM 字节无法标明采样率或声道数，因此仍需依赖厂商按请求格式返回。

Gemini 和 Fish 请求 WAV，在渲染前提取其中的 PCM。解码器要求完整的 RIFF 长度、唯一的 PCM1 格式块和音频数据块、单声道 24 kHz 16 位采样、block alignment 为 2、byte rate 为 48000。允许额外的 RIFF 元数据块；截断文件、未知长度的流式 WAV、重复音频块和其他编码均报错。真实服务是否返回兼容的完整 WAV，尚未验证。

这些检查能够发现容器和格式问题，但不能证明模型读出了每个字，或准确执行了表演指令。各厂商还有自己的限制，例如 Azure 的十分钟截断，详见对应章节。

### 输出与恢复

所有 provider（包括 `stub`）都会在目标文件旁创建临时文件，合成、写入和关闭成功后才替换目标文件。在 POSIX 系统上，临时文件和完成后的文件使用私有的 `0600` 权限；进程的 umask 可以进一步收紧权限。替换已有文件时也会替换其权限，不保留原有模式。stub 仍生成 8 kHz、单声道、16 位测试音；真实厂商的输出仍为 24 kHz、单声道、16 位音频。

请求准备期间、创建输出前、调度和写入真实厂商片段时，以及最终重命名前，都会检查取消状态。进行中的 HTTP 请求也会收到取消上下文。若在替换前报告取消或失败，会删除临时输出并保留已有目标文件。取消若在最后一次检查后到达，替换仍可能完成；成功替换后不会回滚。父目录需要事先存在。之前的请求可能已经计费；目前没有自动重试、厂商回退、缓存或续传。

请求失败时，请对照厂商官方参考检查模型、音色、区域和密钥权限。离线测试验证的是 Timbrex 如何构造请求、处理响应，不代表服务可用性或语音质量。
