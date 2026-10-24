# 配置

[English](configuration.md) · [厂商接入](providers.zh-cn.md) · [教程](tutorial.zh-cn.md)

配置用于设置默认 provider、模型、音色和严格模式。`check`、`synth` 和 `config` 会加载一份 YAML 文件及文档列出的环境变量。脚本与命令行参数可以覆盖相应默认值；凭证和端点只能来自配置文件或环境变量。

## 配置文件

默认情况下，`check`、`synth` 和 `config` 读取**当前工作目录**下存在的 `timbrex.yaml`。不会搜索脚本所在目录、父目录或主目录。`--config` 指定一个文件。显式路径不存在时会报错。

脚本和配置输入必须是普通文件；支持指向普通文件的符号链接。命名管道（FIFO）、设备和目录均会被拒绝。

Interrupt 和 SIGTERM 会取消命令上下文。CLI 在配置加载、读取、解析、检查和 lowering 的阶段边界，以及报告检查成功或开始合成之前检查取消状态。这些检查点不会中断正在执行的解析、检查或文件系统操作。合成成功提交输出后，后续取消不会将已完成的操作改判为失败。

```bash
./timbrex --config examples/timbrex.yaml config
./timbrex --config examples/timbrex.yaml check examples/expressive.tbx
./timbrex --config examples/timbrex.yaml synth examples/expressive.tbx --provider stub -o tone.wav
```

`--config` 是全局参数，可放在命令之前或之后。帮助、版本和 shell 补全不加载配置。Timbrex 不加载 `.env`，不展开 YAML 中的 `${VARIABLE}`，也不监听文件变化。

## 本地环境文件

[.env.example](../.env.example) 按厂商列出所有支持的环境变量。所有赋值行都已注释且值为空，因此未经修改的模板会保留已有环境变量、YAML 设置和内置默认值，不会选择付费厂商。`stub` 不需要凭证；只配置你打算使用的语音服务即可。必需的凭证、音色和端点见[厂商接入](providers.zh-cn.md)。

在仓库根目录创建本地副本（若已有 `.env`，请保留，不要覆盖）：

```bash
cp -i .env.example .env
chmod 600 .env
```

在本地编辑 `.env`，仅取消需要的变量的注释，并将值放在单引号内。`=` 两侧不要加空格。单引号会保留 `$`、空格、`#` 等字符的原样含义；值中含单引号时需要使用 shell 转义。导出的非空值会覆盖 YAML 设置；不使用的行保持注释，也能保留 shell 中已有的值。

Timbrex 没有自动加载 dotenv 的功能，也不引入 dotenv 依赖。在 Bash、Zsh 等兼容 POSIX 的 shell 中，运行 Timbrex 前手动导出编辑后的文件：

```bash
set -a
. ./.env
set +a
./timbrex config
```

`.` 命令会执行 shell 代码：只加载自己编辑并检查过的文件，不要加载来源不可信的下载文件。导出的变量在当前 shell 及其子进程中生效。`config` 隐去 API key，且不会联系厂商；之后若用真实厂商运行 `synth`，可能产生费用。仓库忽略 `.env` 和 `.env.*` 本地变体，但保留 `.env.example`。不要将真实凭证写入模板、提交到仓库或粘贴到聊天中。

## 文件格式

```yaml
provider: openai
strict_tags: true
providers:
  openai:
    model: gpt-4o-mini-tts
    voice: coral
    endpoint: https://api.openai.com/v1
  azure:
    region: eastus
    voice: en-US-JennyNeural
```

文件包含一个 YAML 映射，键区分大小写。未知键、未知厂商 ID、错误值类型、重复键、多个 YAML 文档和 null 值均会报错。空文件合法。即使环境变量会覆盖错误的文件值，也会先报错。

| 键 | 含义 |
| --- | --- |
| `provider` | 默认厂商，初始为 `stub` |
| `strict_tags` | 默认标签检查模式，初始为 `false` |
| `providers.<id>.api_key` | 对应厂商的凭证，也可以使用该厂商的环境变量 |
| `providers.<id>.model` | 默认模型；Azure 通过音色选择模型，指定模型会报错 |
| `providers.<id>.voice` | 未配置 cast 映射的角色使用的兜底音色 |
| `providers.<id>.endpoint` | OpenAI、ElevenLabs、MiniMax、Gemini、Fish 的 API 基础 URL；Azure、Qwen 的完整请求 URL |
| `providers.azure.region` | 未设置端点时使用的 Azure 区域 |
| `providers.<id>.synthesis` | 下文列出的强类型单次请求选项，不会修改已保存的音色设置 |

厂商 ID 为 `stub`、`openai`、`elevenlabs`、`minimax`、`azure`、`qwen`、`gemini`、`fish`。默认值和支持的环境变量名见 [厂商接入](providers.zh-cn.md)。[完整示例](../examples/timbrex.yaml) 包含各语音服务。stub 忽略语音设置。

端点必须是 HTTP(S) URL，不能嵌入凭证、查询参数或 fragment。凭证会发送至配置的端点，应使用可信服务或网关。Azure 显式端点优先于区域配置。检查脚本和查看配置不要求凭证，合成时验证所选厂商的凭证和端点。

## 优先级

每项设置取已提供值中优先级最高的一个：

1. CLI 参数（`--provider`、`--model`、`--voice`、严格模式参数）。
2. 脚本 front matter（`provider`、`model`、`voice`、`strict_tags`）。
3. 环境变量。
4. 选中的 YAML 文件。
5. 内置默认值。

`TIMBREX_PROVIDER` 覆盖文件中的默认厂商，`TIMBREX_STRICT_TAGS` 覆盖严格模式，接受 Go 布尔写法，包括 `true` 和 `false`。原有的 `OPENAI_API_KEY`、`OPENAI_MODEL`、`OPENAI_BASE_URL` 等厂商变量继续有效。未设置或为空的环境变量保留低优先级设置，无关变量被忽略。

Cast 音色映射仍然优先于兜底音色，包括 `--voice`。`voice` 或 `as` span 在自身作用域内覆盖音色选择，不改变角色 ID。切换厂商不会转换模型和音色 ID。端点和凭证不能来自脚本。

`--strict-tags=false` 显式关闭严格模式，`--no-strict-tags=false` 则开启。两个参数名互斥，从而区分“未传入”和“显式 false”。

## 厂商合成选项

选中的 YAML 文件支持在 `providers.elevenlabs.synthesis`、`providers.fish.synthesis`、`providers.qwen.synthesis` 和 `providers.minimax.synthesis` 下设置强类型的单次请求选项。这些选项没有 CLI、环境变量或脚本 front matter 覆盖项。其他厂商拒绝 `synthesis` 配置段；它不是任意 API 参数透传，也不增加 DSL 语法。

省略字段会保留原请求。显式 `0`、`false`、允许的空字符串和空列表会原样保留；列表不会合并或去重。空映射不提供覆盖值。Null、未知或带点的 schema 键、错误标量类型、非有限数值和超范围值会在本地失败。所有片段都会在首个 HTTP 请求或输出变更前完成准备。这些选项不会创建资源，也不会修改已保存的音色或词典。

### ElevenLabs

[Create Speech](https://elevenlabs.io/docs/api-reference/text-to-speech/convert) 支持以下请求覆盖项；另见[音色设置 schema](https://elevenlabs.io/docs/api-reference/voices/settings/update)。

```yaml
providers:
  elevenlabs:
    synthesis:
      seed: 42
      apply_text_normalization: auto
      apply_language_text_normalization: false
      voice_settings:
        stability: 0.5
        similarity_boost: 0.75
        style: 0
        use_speaker_boost: false
      pronunciation_dictionary_locators:
        - pronunciation_dictionary_id: YOUR_EXISTING_DICTIONARY_ID
          version_id: YOUR_EXISTING_VERSION_ID
```

- `seed`：`0`–`4294967295` 的整数，原样发送给每个片段。服务仅尽力提供确定性；分段、文本、模型或音色变化都会改变请求。
- `apply_text_normalization`：`auto`、`on` 或 `off`。`apply_language_text_normalization` 是布尔值；ElevenLabs 当前说明支持日语，且可能显著增加延迟。它不替代现有 `lang` 规则。
- `voice_settings.stability` 和 `similarity_boost`：`0`–`1` 的数值。`style` 是控制厂商风格夸张程度的有限数值，不是 DSL 的文本 `style`；链接的 API schema 未公布数值上下界，Timbrex 不自行设定。`use_speaker_boost` 是布尔值。这些设置与现有 DSL 语速并存，不覆盖语速。
- `pronunciation_dictionary_locators`：最多三个现有词典，按提供顺序应用。每项必须有 `pronunciation_dictionary_id`；`version_id` 可省略。Timbrex 在本地拒绝空值或无效 UTF-8 的 ID，以及包含空白或控制字符的 ID 或显式版本，保留重复项，并在显式提供时发送 `[]`。它不读取词典内容，也不验证其中各条发音规则的模型兼容性。

[Playground 指南](https://elevenlabs.io/docs/eleven-creative/playground/text-to-speech) 描述了各模型不同的 UI 控件可用性，但不是完整的 HTTP 适用性约定。本地 schema 检查通过不代表每个设置在每个模型上都有效；不会依据隐藏的 UI 控件推断新的模型限制。见[配置示例](../examples/elevenlabs-synthesis.yaml)；合成前需设置有权使用的音色和 API key。

### Fish

[TTS API](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech) 和 [OpenAPI schema](https://github.com/fishaudio/docs/blob/main/api-reference/openapi.json) 定义了以下 S2 请求控制项：

```yaml
providers:
  fish:
    synthesis:
      normalize: true
      normalization_language: en
      prosody:
        normalize_loudness: false
      temperature: 0.7
      top_p: 0.7
      chunk_length: 200
      min_chunk_length: 0
      latency: balanced
      condition_on_previous_chunks: false
      early_stop_threshold: 0
```

- `normalize` 和 `prosody.normalize_loudness`：布尔值。省略时保留 Timbrex 显式设置的 `normalize: false` 和响度规范化 `true`，不会采用新的服务默认值。响度规范化在支持的三个 S2 模型上生效。
- `normalization_language`：映射为 API 的 `language`，用于文本规范化和对齐，而非朗读语言或口音。最多 35 个 ASCII 字母、数字、`_` 或 `-`；允许 `""` 和 `auto`。它不会使 DSL `lang` 获得支持。
- `temperature`、`top_p`、`early_stop_threshold`：`0`–`1` 的数值。
- `chunk_length`：`100`–`300` 的整数；`min_chunk_length`：`0`–`100` 的整数。它们控制单次请求内的服务端分块，不改变 Timbrex 脚本分段。
- `latency`：`low`、`normal` 或 `balanced`；`condition_on_previous_chunks`：布尔值。

此 schema 不提供 `features`、发音词典、`max_new_tokens` 或 `repetition_penalty`。见[配置示例](../examples/fish-synthesis.yaml)。

### Qwen

[Qwen-Audio-TTS HTTP API](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api) 接受 `input.seed` 和 `input.hot_fix`：

```yaml
providers:
  qwen:
    synthesis:
      seed: 0
      hot_fix:
        pronunciation:
          - text: 天气
            replacement: tian1 qi4
        replace:
          - text: 今天
            replacement: 金天
```

`seed` 是 `0`–`65535` 的整数，每个片段使用相同值。每个 hot-fix 列表由强类型的 `text`/`replacement` 字符串对组成，序列化为 `input.hot_fix.pronunciation` 或 `input.hot_fix.replace` 下有序的单条目对象，绝不是 `input.replace`。两个键都必填。Timbrex 在本地拒绝空源文本或包含无效 UTF-8 的字符串对；保留字面空白、Unicode、空替换字符串、重复源键和相互重叠的词。文档未明确重复项或重叠项的优先级，也未明确空替换值的行为：保留顺序不代表承诺任何匹配算法。显式 `[]` 会发送；省略的列表和空 `hot_fix` 映射不会添加字段。见[配置示例](../examples/qwen-synthesis.yaml)。

### MiniMax

[同步 TTS API](https://platform.minimax.io/docs/api-reference/speech-t2a-http) 支持顶层 `text_normalization_mode`：

```yaml
providers:
  minimax:
    synthesis:
      text_normalization_mode: quality
```

允许值为 `basic` 和 `quality`。`quality` 要求非流式 `speech-2.8-hd`、`speech-2.8-turbo`、`speech-2.6-hd` 或 `speech-2.6-turbo`；其他模型 ID 会被 Timbrex 拒绝，避免静默降级。当前适配器采用非流式请求。省略时仍关闭文本规范化；不会添加已弃用的 `voice_setting.text_normalization` 字段。见[配置示例](../examples/minimax-synthesis.yaml)。

计费提醒：即使 Timbrex 不创建音色，使用现有设计或克隆音色进行首次合成仍可能产生额外音色费用（见[价格说明](https://platform.minimax.io/docs/pricing/overview#audio)）。对于规范化文本，不要只根据原脚本估算用量：[响应 schema](https://platform.minimax.io/docs/api-reference/speech-t2a-http.md) 区分计费字符 `usage_characters`、仅供参考的 `word_count` 和高质量规范化字符 `norm_text_char_count`（同样仅供参考，目前不用于计费）。Timbrex 尚不显示这些用量元数据或 `usage_voice_count`。

## 合成并发数

`synth --concurrency N` 控制真实厂商同时进行的语音片段任务数。默认值为 `1`（串行），接受 `1` 到 `8` 的整数，输出仍按脚本顺序排列。`stub` 只接受 `1`。它刻意设计为单次运行的 CLI 参数，不是 YAML 键、front matter 字段或环境变量；`timbrex config` 不显示它。调高前请阅读[并发生成](providers.zh-cn.md#并发生成)。

## 查看

`timbrex config` 将合并后的默认值、文件和环境变量以 YAML 输出。已配置的 API key 替换为 `REDACTED`。它不读取脚本、不应用 `synth` 参数、不验证凭证，也不联系厂商。词典标识符和 hot-fix 文本不会脱敏；不要公开包含私人文本的配置。输出不是凭证备份。仓库忽略根目录的 `timbrex.yaml`。不要提交凭证。
