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

Cast 音色映射仍然优先于兜底音色，包括 `--voice`。`voice` 或 `as` span 在自身作用域内覆盖当前角色的音色。切换厂商不会转换模型和音色 ID。端点和凭证不能来自脚本。

`--strict-tags=false` 显式关闭严格模式，`--no-strict-tags=false` 则开启。两个参数名互斥，从而区分“未传入”和“显式 false”。

## 查看

`timbrex config` 将合并后的默认值、文件和环境变量以 YAML 输出。已配置的 API key 替换为 `REDACTED`。它不读取脚本、不应用 `synth` 参数、不验证凭证，也不联系厂商。输出不是凭证备份。仓库忽略根目录的 `timbrex.yaml`。不要提交凭证。
