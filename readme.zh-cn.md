# Timbrex

[English](readme.md)

Timbrex 是一套用英文 ASCII 标签标注表演的语音脚本语言。`.tbx` 脚本经编译后，由 TTS 厂商生成音频。

```tbx
[happy|很高兴见到你！] [pause 300ms] [slow|慢慢来。]
```

纯文本就是合法脚本。标签只有一种形式：`[name]`、`[name args]` 或 `[name args|content]`。台词可用任意语言，标签名保持英文 ASCII。语法冻结为 `timbrex: 1`（[SPEC v0.3.4](docs/spec.zh-cn.md)）。

## 安装

需要 Go 1.27，与 [go.mod](go.mod) 一致。

```sh
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex version
```

在仓库根目录，`go run .` 与 `./timbrex` 是同一个程序。没有 `cmd/timbrex` 包。

## 快速开始

```sh
./timbrex check examples/expressive.tbx
./timbrex synth examples/expressive.tbx --provider stub -o tone.wav
# 先设置 OPENAI_API_KEY。
./timbrex synth examples/expressive.tbx --provider openai -o speech.wav
```

`check` 打印 `ok` 或诊断。`stub` 写出 8 kHz 测试音，不是语音，也不需要密钥。HTTP 适配器写出单个 24 kHz、单声道、16 位 PCM WAV。它们有离线契约测试，尚未用真实账号验证。

## 厂商

| ID | 必需配置 | 默认模型 |
| --- | --- | --- |
| `stub` | 无 | 测试音（不是语音） |
| `openai` | `OPENAI_API_KEY` | `gpt-4o-mini-tts` |
| `elevenlabs` | `ELEVENLABS_API_KEY` 和音色 ID | `eleven_v3` |
| `minimax` | `MINIMAX_API_KEY` | `speech-2.8-hd` |
| `azure` | `AZURE_SPEECH_KEY`，以及区域或端点 | 由音色决定 |
| `qwen` | `DASHSCOPE_API_KEY` 和 `QWEN_ENDPOINT` | `qwen-audio-3.1-tts-flash` |
| `gemini` | `GEMINI_API_KEY` | `gemini-3.8-flash-tts` |
| `fish` | `FISH_API_KEY` 和音色 / reference ID | `s2.1-pro` |

Gemini 拒绝数值语速，使用 [examples/gemini.tbx](examples/gemini.tbx)。Fish 需要音色 reference ID，使用 [examples/fish.tbx](examples/fish.tbx)。词表认识一个标签，不代表所有厂商都能合成它。不支持的控制会在发请求前报错，不会静默丢弃。配置与映射见 [厂商接入](docs/providers.zh-cn.md)、[能力表](docs/capabilities.zh-cn.md)。

## 命令

`version`、`check`、`synth`、`config`。优先级：命令行参数、脚本 front matter、环境变量、`./timbrex.yaml` 或 `--config`、内置默认值。默认厂商是 `stub`。详见 [配置](docs/configuration.zh-cn.md)。

## 文档

从 [教程](docs/tutorial.zh-cn.md) 开始。开发见 [参与贡献](contributing.zh-cn.md)。

| 文档 | English | 简体中文 |
| --- | --- | --- |
| 教程 | [Tutorial](docs/tutorial.md) | [教程](docs/tutorial.zh-cn.md) |
| 语言参考 | [SPEC v0.3.4](docs/spec.md) | [语言参考](docs/spec.zh-cn.md) |
| 配置 | [Configuration](docs/configuration.md) | [配置](docs/configuration.zh-cn.md) |
| 厂商接入 | [Providers](docs/providers.md) | [厂商接入](docs/providers.zh-cn.md) |
| 已实现控制 | [Capabilities](docs/capabilities.md) | [能力表](docs/capabilities.zh-cn.md) |
| 厂商调研 | [Research](docs/market_survey.md) | [厂商调研](docs/market_survey.zh-cn.md) |
| 设计思路 | [Design](docs/reflection.md) | [设计思路](docs/reflection.zh-cn.md) |
| 参与贡献 | [Contributing](contributing.md) | [参与贡献](contributing.zh-cn.md) |
| 编辑器高亮 | [VS Code / Cursor](editors/vscode/readme.md) | [VS Code / Cursor](editors/vscode/readme.zh-cn.md) |

采用 MIT 许可证，见 [license](license)。
