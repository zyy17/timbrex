<p align="center"><img src="docs/assets/logo.png" alt="Timbrex" width="420"></p>

# Timbrex

[English](readme.md)

Timbrex 是一门语音脚本语言和配套 CLI。将台词、角色、表演指令和停顿写在文本文件中，再通过语音合成（TTS）厂商生成 WAV 音频。

Timbrex 取意于音色（timbre）与表达（expression）。二者融合即为 Timbrex。

TTS API 负责生成语音，Timbrex 负责组织调用这些 API 的脚本：哪句台词用哪个音色、表演指令作用于哪些文字、哪里需要停顿。这些设置与台词放在一起，可以随文本编辑、审阅并纳入 Git 版本管理；适配器负责构建请求和拼接音频。

如果你在制作教程旁白、配音草稿或对话原型，希望修改台词时也能一并调整表达方式，可以使用 Timbrex。

## 看一份脚本

下面的对话为两个角色指定 OpenAI 音色，并加入停顿和表演指令：

```tbx
---
strict_tags: true
cast:
  lin: {voices: {openai: coral}}
  chen: {voices: {openai: onyx}}
---
lin: [happy] 你来了！[pause 300ms] [whisper|快进来。]
chen: [slow|我有件事想告诉你。]
```

`cast` 使 `lin:` 和 `chen:` 成为角色发言，并分别使用 `coral` 和 `onyx`，这两个音色 ID 仅适用于 OpenAI。`[happy]` 作用于 Lin 的这次发言，`[whisper|…]` 和 `[slow|…]` 作用于各自包含的文字，`[pause 300ms]` 则在两句话之间插入本地静音。

仓库中的 [examples/dialogue.tbx](examples/dialogue.tbx) 使用相同结构和英文台词，可直接运行。脚本使用 `.tbx` 或 `.timbrex` 扩展名，也可以只写普通文本。台词可使用任意语言，实际支持的语言、音色和指令取决于所选厂商与模型。

## 快速开始

安装 Git、Go 1.27 和 `make`，然后在终端运行：

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex check examples/dialogue.tbx
./timbrex synth examples/dialogue.tbx --provider stub -o tone.wav
```

`check` 打印 `ok`，不会联系语音服务。最后一条命令打印 `tone.wav` 并生成文件，可以用音频播放器打开。`stub` 不需要凭证，也不发送语音请求：它只产生一段测试音，不会朗读台词，也不会表现脚本中的停顿和表演指令。

### 生成语音

要使用示例中的 OpenAI 音色，在环境中配置 `OPENAI_API_KEY`，或在[配置文件](docs/configuration.zh-cn.md)中设置 `providers.openai.api_key`。也可以将 [.env.example](.env.example) 复制为 `.env`，只填写需要的设置，再[手动加载](docs/configuration.zh-cn.md#本地环境文件)；Timbrex 不会自动加载它。不要把密钥写入脚本、提交到仓库或粘贴到聊天中。然后在同一仓库目录运行：

```bash
./timbrex synth examples/dialogue.tbx --provider openai --model gpt-4o-mini-tts -o dialogue.wav
```

这条命令会向 OpenAI 发送请求，可能产生费用；成功后打印 `dialogue.wav`。使用前请阅读 [OpenAI 接入说明与模型生命周期提示](docs/providers.zh-cn.md#openai)。选择其他厂商时，按对应的[接入指南](docs/providers.zh-cn.md)配置，并使用该厂商支持的音色 ID 和控制指令。

[教程](docs/tutorial.zh-cn.md)会逐步介绍如何编写自己的脚本、添加角色，以及调整一段台词的表达方式。

## 厂商与当前支持范围

目前提供 **OpenAI、ElevenLabs、MiniMax、Azure Speech、Qwen、Gemini TTS 和 Fish Audio** 适配器。各适配器支持的控制不同：有些转换为数值 API 参数，有些转换为近似的表演提示。切换厂商不保证音色或表达效果相同，具体映射见[能力表](docs/capabilities.zh-cn.md)。

真实语音适配器在发送合成请求前，会验证全部计划合成的语音片段。通过 `check` 表示脚本符合语言规则，不代表厂商支持；语音片段使用了不支持的控制时，会在合成准备阶段报错。

语音默认串行合成，停顿在本地插入，最终输出 **24 kHz、单声道、16 位 PCM WAV**。真实厂商可通过 `--concurrency 2` 启用[有界并发](docs/providers.zh-cn.md#并发生成)，音频仍按脚本顺序输出。目前未实现混音、重叠对话、流式输出、厂商自动回退，以及音色创建或克隆。

项目仍在开发中，[DSL 规范 v1](docs/spec.zh-cn.md) 描述当前语言规则，后续可能调整。适配器已有离线请求与响应测试，真实账号兼容性和声音效果尚未验证。

## 文档

| 想要了解的内容 | 指南 |
| --- | --- |
| 编写并合成第一份脚本 | [教程](docs/tutorial.zh-cn.md) |
| 让 agent 编写、检查并合成脚本 | [Skill 快速开始](docs/agent-skill.zh-cn.md) |
| 理解编译器与合成实现 | [整体架构](docs/architecture.zh-cn.md) |
| 查询语法、作用域或诊断 | [DSL 规范 v1](docs/spec.zh-cn.md) |
| 配置默认值、凭证和覆盖项 | [配置](docs/configuration.zh-cn.md) |
| 选择厂商并查询支持的控制 | [厂商接入](docs/providers.zh-cn.md) · [能力表](docs/capabilities.zh-cn.md) |
| 启用语法高亮 | [VS Code / Cursor](editors/vscode/readme.zh-cn.md) |

## 参与贡献

欢迎提交问题报告、改进文档或完善厂商接入。报告问题时，请提供最小脚本、执行命令、预期结果与实际结果，并移除凭证和私密文本。

[贡献指南](contributing.zh-cn.md)介绍开发环境、包职责、离线测试和 PR 流程；语言与实现的设计取舍见[设计思路](docs/reflection.zh-cn.md)。

## 许可证

Copyright (c) 2026 zyy17

采用 [Apache License 2.0](license)。
