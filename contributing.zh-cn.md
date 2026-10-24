# 参与 Timbrex 贡献

[English](contributing.md)

本文介绍本地开发、测试和 pull request 流程。可先通过 [教程](docs/tutorial.zh-cn.md) 体验语言，在 [DSL 规范 v1](docs/spec.zh-cn.md) 中查阅规则，在 [AGENTS.md](AGENTS.md) 中了解包的职责。语言仍在开发中；提议修改规则时，应说明对现有脚本的影响。

## 准备开发环境

安装 [go.mod](go.mod) 声明的 Go 版本、golangci-lint **v2.14.0** 和 Node.js **24**，与 CI 保持一致。Node.js 仅用于编辑器运行时测试。若准备提交 pull request，请先 fork 仓库，再克隆自己的 fork。只想了解项目时，可直接克隆上游：

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
go mod download
make editor-deps
make build
./timbrex check examples/hello.tbx
./timbrex synth examples/hello.tbx --provider stub -o hello.wav
```

该示例的 `check` 结果为 `ok`。`stub` 无需凭证，也不发送网络请求，只写入一段测试音，用于验证输出流程，不能用于评价语音质量。程序入口是仓库根目录的 `main.go`；也可以执行 `go run . check examples/hello.tbx`。

## 找到需要修改的代码

[架构指南](docs/architecture.zh-cn.md)用一份脚本贯穿编译器、请求规划和最终音频输出，配有图表与源码链接。

主要处理流程是 `internal/lex` → `internal/parse` 和 `internal/ast` → `internal/check` → `internal/lower` 和 `internal/ir` → `internal/providers`。标准标签位于 `internal/vocab`，命令组装位于 `internal/cli`，YAML 配置和环境变量映射位于 `internal/config`。测试与对应代码放在一起。

从源码开始的 lowering 测试和基准测试必须依次调用 `parse.Parse` → `check.Apply` → `check.Failed` → `lower.Script`，仅在检查通过后转换为 IR。专门验证无效输入的 planner 测试应直接构造 IR。IR 通过 utterance 的 speaker 区分场景行与语音，通过节点种类区分状态与事件，并用成对的 span 标记界定局部作用域，不再另设 scope 字段。

使用简短、符合 Go 惯例的函数，优先沿用现有写法。解析逻辑应独立于厂商和 I/O，只有 `internal/cli` 导入 Cobra。适配器接收已解析的配置和 IR，不重新加载配置或解析脚本。标识符、错误、帮助文本和代码注释使用英文。注释应解释约束或不易看出的设计原因，避免复述代码。

报告 bug 时，请提供最小 `.tbx` 脚本、执行命令、预期与实际结果，以及 `timbrex version` 输出。厂商问题还需模型和区域信息，并去除凭证、私有文本和签名音频 URL。重大的语法、架构或兼容性变更请先开 issue 讨论；范围明确的修复或有文档依据的厂商接入可以直接提交 pull request。

## 运行检查

开发过程中可先测试相关包，例如 `go test ./internal/parse`。提交前执行与 CI 一致的检查：

```bash
make verify
```

`make verify` 运行 `make tidy`、`make test`、`make lint` 和 `make test-editor`。`make tidy` 检查 `go.mod` 或 `go.sum` 是否需要变化，但不改写文件。`make test` 构建 `./timbrex` 并运行 `go test ./...`。`make lint` 运行 golangci-lint 及其格式检查。`make test-editor` 运行实际的 TextMate/Oniguruma 分词器测试。需要应用 Go 格式化时可执行 `make fmt`。CI 在 Linux 和 macOS 上运行 Go 测试，在 Linux 上运行编辑器运行时测试。

准备环境时，用 `go mod download` 和 `make editor-deps` 安装依赖。验证过程不会执行 `npm ci`，也不会联系 TTS 服务；Go 模块和工具已缓存后，可以离线运行。依赖变化时再重新安装。

收集 Go 覆盖率时，运行 `go test -count=1 -coverprofile=coverage.out ./...`，无需跳过子进程测试。CLI 输入辅助进程只继承白名单中的运行时环境变量（包括 `GOCOVERDIR`），不继承厂商凭证或配置；Go 会将其子进程计数合并到父进程的覆盖率报告中。单独构建的 CLI 二进制、编辑器运行时测试和真实服务行为不计入该报告；语句覆盖率不能证明分支覆盖率或音频质量。

真实编码器导出测试需要 `PATH` 中有可信的 `ffmpeg` 和 `ffprobe`。普通 `go test ./...` 在两者可用时运行这些测试，仅在缺少工具时跳过；已安装但不兼容的构建会失败，不会跳过。本地可运行 `TIMBREX_TEST_FFMPEG=1 go test ./...` 强制要求这些测试。Linux CI 设置 `TIMBREX_TEST_FFMPEG=1`，缺少工具或编码器会使该必需检查失败。测试只编码和检查本地样本，不发送付费 TTS 请求。原生 WAV 与模拟进程失败测试不依赖 FFmpeg。支持的编码器见[音频输出](docs/audio-output.zh-cn.md)。

默认测试必须离线。用 `httptest` 验证厂商请求，为行为修复和边界情况补充回归测试。常规测试不能要求凭证或产生费用。修改 TextMate YAML 语法后，用 `go test ./editors/vscode -update` 重新生成随包提供的 JSON，详见 [编辑器指南](editors/vscode/readme.zh-cn.md)。

真实服务验证应单独进行，并由执行者明确启用。记录模型、音色、区域、请求参数、日期和结果，不记录秘密。API 是否接受请求与音频质量应分别说明，模拟请求不能证明这两项。

## 新增厂商

先阅读官方 API 文档，在适配器和 [接入指南](docs/providers.zh-cn.md) 中链接来源。核实认证方式、请求结构、音频格式，以及完成和错误响应。在 `internal/providers/<name>` 实现 `Provider`，通过 `init` 注册，由 CLI 空白导入；在 `internal/config` 添加默认值和环境变量映射。

在接入指南和 [能力参考](docs/capabilities.zh-cn.md) 中说明支持的模型、凭证、标签映射、数值范围、语言提示和限制。适配器无法表达的控制应报错。测试完整的 CLI 到请求路径、畸形响应和失败处理。只有代码确实被多处复用时，才将其放到 `internal/providers`。

## 编写和审查文档

每个页面各司其职：README 介绍项目和首次运行，教程带读者完成脚本，规范说明语言规则，配置与厂商页面说明接入方法，设计说明解释取舍。详细内容用链接引用，避免多处重复。

面向初次接触项目的开发者写作：说明功能的用途，给出小示例，在限制会影响使用时就地解释。少用口号，不写缺乏依据的结论或对未来兼容性的承诺。中文应自然简洁，不必照搬英文句式；标识符和有助于理解的技术术语可保留英文。区分已实现行为、近似处理和提案，厂商行为引用官方来源。

英文和 `.zh-cn.md` 版本一起更新，示例与技术含义保持一致。文档文件名使用小写，`AGENTS.md` 除外。文档测试会解析所有 `tbx` 围栏示例和 `.tbx` 文件，检查本地文件链接、双语标题层级、Markdown 围栏、表格转义和粗体标签边界。还应使用兼容 GitHub 的渲染器预览改动页面及其中的图示，语法检查不能代替阅读效果检查。测试通过不代表厂商支持该功能或音频质量合格，这类说明需要结合适配器代码和测试核实。

## 提交 pull request

每次变更尽量聚焦，提交消息使用英文 [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)，例如 `fix: preserve speaker after nested span`。说明解决的问题、修改后的行为和已通过的检查；有助于理解时给出简短的前后对比，并注明兼容性变化或尚未完成的验证。不提交密钥、`.env` 文件或生成的音频。贡献适用仓库的 [Apache License 2.0](license)。
