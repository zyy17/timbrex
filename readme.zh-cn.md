# Timbrex

[English](readme.md)

**用一个小巧的 DSL 生成富有表现力的语音。** 写下台词及其表演方式，通过 ElevenLabs、OpenAI、MiniMax、Azure Speech 或 Qwen 生成音频。

```tbx
[happy|很高兴见到你！] [pause 300ms] [slow|慢慢来。]
```

纯文本就是合法脚本。标签只有一种形式：`[name]` 或 `[name|content]`。台词可以使用任意语言，标签名使用英文 ASCII。语法冻结在 `timbrex: 1`，扩展厂商和词表不需要增加语法。

Timbrex 处于早期开发阶段。编译器和五个 HTTP 适配器已经实现，并有离线契约测试覆盖，但这些适配器**尚未通过真实厂商账号验证**。模型权限、音色、区域可用性和表演指令效果仍需实测。不同厂商不保证生成相同的表演效果。

## 生成第一段语音

需要 Go 1.27，与 [go.mod](go.mod) 一致。

```sh
make build
./timbrex check examples/expressive.tbx --strict-tags

# 先在环境中设置 OPENAI_API_KEY。
./timbrex synth examples/expressive.tbx --provider openai -o speech.wav
```

上面的示例使用各适配器的默认模型及合适的音色即可运行。ElevenLabs 还需要你的音色 ID，Qwen 需要工作空间端点。五个服务的配置见[厂商接入](docs/providers.zh-cn.md)。

无需凭证或网络请求，也可以体验编译流程：

```sh
./timbrex synth examples/expressive.tbx --provider stub -o tone.wav
```

`stub` 生成测试音，不生成语音。未选择厂商时默认使用它。

## 学习语言

```tbx
---
timbrex: 1
provider: openai
strict_tags: true
cast:
  lin:
    name: Lin
    voices: {openai: coral}
  chen:
    name: Chen
    voices: {openai: onyx}
---
lin: [happy] You came! [pause 300ms] [whisper|Come inside.]
chen: [slow|I've got something to tell you.]
```

- `[happy]` 作用于本轮剩余台词，`[happy|台词]` 只作用于其中的文字。
- 一行只有不带 span 的标签时，设置场景状态。独占一行的 `[reset]` 清除场景状态。
- 只有在 `cast` 中声明了 `lin`，`lin:` 才会开启角色发言。没有声明时，它仍是普通台词。`cast: auto` 可显式启用自动识别。
- 无前缀的段落由 `narrator` 朗读，也可以用 front matter 的 `narrator:` 指定其他 ID。
- 行首 `//` 表示注释，空行分隔发言。
- 自由表演指令优先写成 `[act "holding back tears"]`。`--strict-tags` 拒绝未知标签名；`--no-strict-tags` 允许 `[thoughtfully]` 这样的隐式表演指令。

详见[语言参考](docs/spec.zh-cn.md)和[已实现能力表](docs/capabilities.zh-cn.md)。词表认识一个标签，不代表所有厂商都能合成它。不支持的控制会报错，不会静默丢弃。

## 命令与输出

```sh
./timbrex version
./timbrex config
./timbrex check script.tbx
./timbrex synth script.tbx --provider openai --model gpt-4o-mini-tts --voice coral -o script.wav
```

设置的优先级依次为：CLI 参数、脚本 front matter、环境变量、选中的 YAML 配置、内置默认值。默认厂商为 `stub`。角色的 `cast.<id>.voices.<provider>` 优先于默认音色。`--voice` 提供兜底音色，不覆盖角色映射。

所有命令由 Cobra 驱动。配置通过 koanf 统一加载一次，读取存在的 `./timbrex.yaml`，或由 `--config` 指定的文件。`timbrex config` 展示文件和环境变量合并后的有效设置，API key 会隐藏。可从 [examples/timbrex.yaml](examples/timbrex.yaml) 开始；[配置指南](docs/configuration.zh-cn.md)说明文件查找、环境变量映射和校验规则。

`check` 验证语法和词表，不需要凭证或网络。`synth` 先执行相同检查，再验证适配器支持的控制，最后发出请求。严格模式参数覆盖 front matter 的 `strict_tags`，默认非严格。使用 `cast: auto` 时，`check` 将每个识别出的角色列出一次。

真实厂商输出单个 24 kHz、单声道、16 位 PCM WAV 文件。发言和控制变化会切成顺序请求，显式停顿生成为本地静音。span 边界处可能出现听感不连贯。整个渲染成功后才替换输出文件；渲染失败时已有音频保留，但已完成的 API 请求仍可能计费。没有自动重试。`-o` 默认为源文件路径换成 `.wav`，不允许覆盖源文件。

## 开发

```sh
make test    # 构建 ./timbrex 并运行全部 Go 测试
make lint    # 需要 golangci-lint v2.14.0，与 CI 一致
make tidy    # 验证模块文件，不修改它们
make fmt
```

测试覆盖解析、诊断、嵌套作用域、各适配器请求契约、精确停顿、失败处理和输出保留。测试只使用本地 HTTP 服务，不读取厂商凭证。参见[参与贡献](contributing.zh-cn.md)和 [AGENTS.md](AGENTS.md)。

处理流程为：源码 → lexer → parser/AST → check → lower/IR → provider → WAV。`internal/` 下的包遵循这些边界，只有 `internal/cli` 依赖 Cobra。`internal/config` 负责配置及环境变量访问，各厂商位于 `internal/providers/<name>`，接收解析后的设置。[编辑器支持](editors/vscode/readme.zh-cn.md)提供 VS Code 和 Cursor 使用的 TextMate 语法。

## 范围与文档

已实现：脚本解析和检查、统一配置管理、角色音色映射、控制作用域、顺序合成、本地停顿和 WAV 输出。尚未实现：流式输出、并发对话、音频混音、发音包装标签、厂商原生内容透传、音色克隆、数据集展开、锁文件，以及 CLI 的 `fmt`/`fix` 命令。词表保留了部分相关标签，但合成时会拒绝它们。

英文为主文档，中文版本描述相同的行为和限制。

| 文档 | English | 简体中文 |
| --- | --- | --- |
| 语言参考 | [Language reference](docs/spec.md) | [语言参考](docs/spec.zh-cn.md) |
| 配置管理 | [Configuration](docs/configuration.md) | [配置管理](docs/configuration.zh-cn.md) |
| 凭证和适配器 | [Provider setup](docs/providers.md) | [厂商接入](docs/providers.zh-cn.md) |
| 已实现控制 | [Capabilities](docs/capabilities.md) | [能力表](docs/capabilities.zh-cn.md) |
| 厂商调研 | [Provider research](docs/market_survey.md) | [厂商调研](docs/market_survey.zh-cn.md) |
| 设计思路 | [Design rationale](docs/reflection.md) | [设计思路](docs/reflection.zh-cn.md) |
| 参与贡献 | [Contributing](contributing.md) | [参与贡献](contributing.zh-cn.md) |

采用 MIT 许可证，详见 [MIT license](license)。
