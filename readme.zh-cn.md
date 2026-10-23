# Timbrex

[English](readme.md)

Timbrex 将脚本转换为语音。你可以用纯文本写台词，加入语气、语速和停顿等指令，再选择一个 text-to-speech（TTS）厂商生成 WAV 文件。

```tbx
[happy|很高兴见到你！] [pause 300ms] [slow|慢慢来。]
```

同一份脚本可以包含多个角色，并为各角色指定音色。标签可以作用于一段文字、一次发言或后续发言，让表演指令与台词放在一起。只写普通文本也能使用。

## 本地试用

安装 Go 1.27 和 `make` 后，运行：

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex check examples/expressive.tbx
./timbrex synth examples/expressive.tbx --provider stub -o tone.wav
```

检查成功后打印 `ok`，生成文件后打印 `tone.wav`。`stub` 不需要账号，会写出一段 8 kHz 测试音。它用于验证文件生成流程，不会朗读台词，也不会表现脚本中的停顿和表演指令。

要生成语音，先配置厂商凭证，以及所需的音色或端点，再用 `--provider` 选择厂商。完整操作见[教程](docs/tutorial.zh-cn.md)，各厂商的配置见[厂商接入](docs/providers.zh-cn.md)。

## 支持范围

Timbrex 提供 OpenAI、ElevenLabs、MiniMax、Azure Speech、Qwen、Gemini TTS 和 Fish Audio 适配器。每个适配器把支持的指令转换为对应厂商的参数或表演提示。[能力表](docs/capabilities.zh-cn.md)区分了数值控制、近似表演指令和不支持的功能。

`check` 检查脚本的语言规则。`synth` 还会在发送请求前检查所选适配器的要求。因此，通过 `check` 的脚本仍可能包含该厂商不支持的控制；合成时会报错，不会悄悄删掉这些控制。

真实语音适配器逐段合成，在本地插入指定的停顿，最后写出 24 kHz、单声道、16 位 PCM WAV。目前不支持混音、重叠对话、流式输出或厂商之间的自动回退。Timbrex 用于将脚本生成音频，不提供音色创建或克隆功能。

适配器已有离线请求与响应测试，尚未验证真实账号兼容性和声音效果。模型可用性、音色要求和服务限制见[厂商接入](docs/providers.zh-cn.md)。

## 了解脚本语言

脚本使用 `.tbx` 或 `.timbrex` 扩展名。台词可以使用任意语言，标签名使用 ASCII 标识符。例如，`[slow|文字]` 为一段文字设置语速，`[pause 300ms]` 插入停顿。

[DSL 规范 v1](docs/spec.zh-cn.md)描述当前语言规则，开发期间可能调整。front matter 中的 `timbrex: 1` 表示所用的 DSL 版本，与 CLI 的软件版本无关。

## 查找文档

| 目标 | English | 简体中文 |
| --- | --- | --- |
| 编写并合成第一份脚本 | [Tutorial](docs/tutorial.md) | [教程](docs/tutorial.zh-cn.md) |
| 查询语法、作用域或诊断 | [DSL Specification v1](docs/spec.md) | [DSL 规范 v1](docs/spec.zh-cn.md) |
| 配置默认值、凭证和覆盖项 | [Configuration](docs/configuration.md) | [配置](docs/configuration.zh-cn.md) |
| 接入语音厂商 | [Provider setup](docs/providers.md) | [厂商接入](docs/providers.zh-cn.md) |
| 查询适配器支持的控制 | [Capabilities](docs/capabilities.md) | [能力表](docs/capabilities.zh-cn.md) |
| 开发或参与贡献 | [Contributing](contributing.md) | [参与贡献](contributing.zh-cn.md) |
| 启用编辑器高亮 | [VS Code / Cursor](editors/vscode/readme.md) | [VS Code / Cursor](editors/vscode/readme.zh-cn.md) |

设计背景见[设计思路](docs/reflection.zh-cn.md)和[厂商调研](docs/market_survey.zh-cn.md)。

采用 MIT 许可证，见 [license](license)。
