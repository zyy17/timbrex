# 厂商接入

[English](providers.md) · [支持的控制](capabilities.zh-cn.md)

Timbrex 直接调用同步 HTTP API。适配器依据下列官方参考实现，核对日期为 **2026-10-22**。离线测试验证请求和响应契约，尚未验证真实账号兼容性或声音质量。凭证必须属于端点所在区域，并具有所选模型和音色的权限。

## 配置

通过 shell 或密钥管理器设置凭证。Timbrex 读取环境变量，**不会**加载 `.env` 文件。不要把密钥写入 `.tbx` 文件或提交到仓库。

| Provider ID | 必需环境变量 | 默认模型 | 默认音色 |
| --- | --- | --- | --- |
| `openai` | `OPENAI_API_KEY` | `gpt-4o-mini-tts` | `coral` |
| `elevenlabs` | `ELEVENLABS_API_KEY`，另需选择音色 | `eleven_v3` | 无；通过 `ELEVENLABS_VOICE_ID`、`--voice`、front matter 或 cast 映射设置 |
| `minimax` | `MINIMAX_API_KEY` | `speech-2.8-hd` | `English_expressive_narrator` |
| `azure` | `AZURE_SPEECH_KEY`，以及区域或端点 | 由音色决定 | `en-US-JennyNeural` |
| `qwen` | `DASHSCOPE_API_KEY`、`QWEN_ENDPOINT` | `qwen-audio-3.1-tts-flash` | `longanhuan_v3.1` |
| `stub` | 无 | 无 | 无；测试音 |

可选的环境变量默认值：

| 厂商 | 模型 | 音色 | 端点覆盖 |
| --- | --- | --- | --- |
| OpenAI | `OPENAI_MODEL` | `OPENAI_VOICE` | `OPENAI_BASE_URL`（含 `/v1` 的基础 URL） |
| ElevenLabs | `ELEVENLABS_MODEL` | `ELEVENLABS_VOICE_ID` | `ELEVENLABS_BASE_URL`（含 `/v1` 的基础 URL） |
| MiniMax | `MINIMAX_MODEL` | `MINIMAX_VOICE_ID` | `MINIMAX_BASE_URL`（含 `/v1` 的基础 URL） |
| Azure | — | `AZURE_SPEECH_VOICE` | `AZURE_SPEECH_ENDPOINT`（完整请求 URL），否则使用 `AZURE_SPEECH_REGION` |
| Qwen | `QWEN_MODEL` | `QWEN_VOICE` | `QWEN_ENDPOINT`（必填完整请求 URL） |

`--model` 和 `--voice` 覆盖 front matter，后者覆盖环境变量默认值。cast 音色映射优先于默认音色。`[voice "vendor-id"|text]` 直接选择音色；`[as lin|text]` 使用 Lin 的映射，没有映射时回退到默认音色。切换厂商不会自动转换其他厂商的模型或音色 ID。

端点覆盖用于你选择的区域服务或可信兼容网关，API 凭证会发送至该端点。远程服务应使用 HTTPS；HTTP 用于本地测试。请求不会跟随重定向。

## OpenAI

```sh
# 先设置 OPENAI_API_KEY。
./timbrex synth examples/expressive.tbx --provider openai --voice coral -o openai.wav
```

使用 bearer 认证调用 `POST https://api.openai.com/v1/audio/speech`，设置 `response_format: pcm`。表演要求放在独立的 `instructions` 中，台词保留在 `input`。适配器分别限制台词和指令不超过 4096 字符。`tts-1`、`tts-1-hd` 可合成纯文本并控制语速，但拒绝表演和语言指令。

参考：[Create speech](https://developers.openai.com/api/reference/resources/audio/subresources/speech/methods/create)、[TTS 指南](https://developers.openai.com/api/docs/guides/text-to-speech)。发布 AI 生成语音时应遵守厂商的披露要求。

## ElevenLabs

```sh
# 先设置 ELEVENLABS_API_KEY 和 ELEVENLABS_VOICE_ID。
./timbrex synth examples/expressive.tbx --provider elevenlabs -o elevenlabs.wav
```

使用 `xi-api-key` 调用 `POST https://api.elevenlabs.io/v1/text-to-speech/{voice_id}?output_format=pcm_24000`。适配器为 `eleven_v3`、`eleven_v4`、`eleven_v4_turbo` 生成原生 audio tags。其他模型 ID 支持纯文本与语速，但拒绝表演标签和非语言事件。适配器采用保守的 5000 字符限制，包含生成的标签。语言提示使用主要语言子标签；`eleven_multilingual_v2` 不接受 `language_code`，因此拒绝显式 `lang`。

参考：[Create speech](https://elevenlabs.io/docs/api-reference/text-to-speech/convert)、[audio tags](https://elevenlabs.io/docs/help-center/product/core-capabilities/text-to-speech/how-do-audio-tags-work-with-eleven-v3-and-v4)、[语速控制](https://elevenlabs.io/docs/eleven-creative/playground/text-to-speech)。Audio tags 描述期望的表演，不保证生成指定音效。

## MiniMax

```sh
# 先设置 MINIMAX_API_KEY。
./timbrex synth examples/expressive.tbx --provider minimax -o minimax.wav
```

使用 bearer 认证调用 `POST https://api.minimax.io/v1/t2a_v2`，请求非流式 hex 输出和 PCM 音频。适配器检查 `base_resp.status_code`、完成状态和 hex 解码。每段文字必须少于 10000 字符。

默认的 `speech-2.8-hd` 支持已映射的非语言事件。适配器仅在 `speech-2.6-hd` 或 `speech-2.6-turbo` 下支持耳语，且同一片段不能同时使用耳语和另一种情绪。音高要求整数半音。语言提示当前映射 `zh en ja ko fr de es ru pt it ar hi th id vi yue`，`zh-HK` 和 `zh-MO` 映射为粤语。其他提示会在本地报错。这是适配器实现的子集，不代表厂商完整语言覆盖。

参考：[同步语音合成 API](https://platform.minimax.io/docs/api-reference/speech-t2a-http)。

## Azure Speech

```sh
# 先设置 AZURE_SPEECH_KEY。
export AZURE_SPEECH_REGION=eastus
./timbrex synth examples/expressive.tbx --provider azure --voice en-US-JennyNeural -o azure.wav
```

使用订阅密钥和 SSML 调用 `POST https://{region}.tts.speech.microsoft.com/cognitiveservices/v1`，请求 `raw-24khz-16bit-mono-pcm`。文本和属性均进行 XML 转义。音色名称决定模型；指定 `--model` 或 front matter 的 `model:` 会报错。

Style、role 和强调能力依赖所选音色。Azure 可能忽略音色不支持的 style，Timbrex 尚未查询音色能力。适配器只允许每个片段一个 style，冲突时会报错。`[role Boy]` 需要同一作用域中存在 style。没有显式 `lang` 时，从音色名称的前两部分推断语言区域。

参考：[REST API](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/rest-text-to-speech)、[音色、风格、角色和韵律](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice)。

## Qwen

```sh
# 先设置 DASHSCOPE_API_KEY，并替换工作空间占位符。
export QWEN_ENDPOINT='https://YOUR_WORKSPACE_ID.ap-southeast-1.maas.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer'
./timbrex synth examples/expressive.tbx --provider qwen -o qwen.wav
```

此适配器面向 **Qwen-Audio-TTS 3.0/3.1**，不是旧版 Qwen3 multimodal-generation API，也不是本地运行的 Qwen3 权重。支持的模型 ID 为 `qwen-audio-3.0-tts-flash`、`qwen-audio-3.0-tts-plus`、`qwen-audio-3.1-tts-flash` 和 `qwen-audio-3.1-tts-plus`。切换模型时应同时选择兼容音色。

必须设置工作空间端点。新加坡使用 `ap-southeast-1`，北京在同一 URL 结构中使用 `cn-beijing`。请求采用单数形式的 `input.instruction`、PCM 格式和 24000 Hz 采样率。完成响应提供签名音频 URL，Timbrex 下载时**不携带** API 密钥。语言提示接受 `zh en fr de ja ko ru pt th id vi es it ms fil ar`，会移除 locale 后缀。

参考：[HTTP API](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api)、[模型与使用限制](https://www.alibabacloud.com/help/en/model-studio/non-realtime-tts-user-guide)。

## 失败处理与限制

单次 HTTP 请求超时为两分钟，响应限制为 64 MiB；最终 PCM 输出限制为 1 GiB。所有语音片段会在发起请求前验证。运行时错误包含厂商和片段编号。HTTP 错误包含状态码，MiniMax/Qwen API 错误包含厂商错误代码。诊断省略响应正文和签名 URL 的查询参数。

渲染器在目标文件旁创建临时文件，成功后才重命名。取消或失败会删除临时输出并保留已有目标文件。父目录必须已存在。先前请求仍可能计费；没有重试、降级、缓存或断点续传。

排错时请对照官方文档检查模型、音色、区域和密钥权限。离线测试通过只证明实现的契约，不证明服务可用性或语音质量。长文本应自行分段，Timbrex 不会按厂商 token 限制自动切块。
