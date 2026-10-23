# 参与 Timbrex 贡献

[English](contributing.md)

本文介绍本地开发、测试和 pull request 流程。可先通过 [教程](docs/tutorial.zh-cn.md) 体验语言，在 [DSL 规范 v1](docs/spec.zh-cn.md) 中查阅规则，在 [AGENTS.md](AGENTS.md) 中了解包的职责。语言仍在开发中；提议修改规则时，应说明对现有脚本的影响。

## 准备开发环境

安装 [go.mod](go.mod) 声明的 Go 版本，以及 CI 使用的 golangci-lint **v2.14.0**。若准备提交 pull request，请先 fork 仓库，再克隆自己的 fork。只想了解项目时，可直接克隆上游：

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex check examples/hello.tbx
./timbrex synth examples/hello.tbx --provider stub -o hello.wav
```

该示例的 `check` 结果为 `ok`。`stub` 无需凭证，也不发送网络请求，只写入一段测试音，用于验证输出流程，不能用于评价语音质量。程序入口是仓库根目录的 `main.go`；也可以执行 `go run . check examples/hello.tbx`。

## 找到需要修改的代码

主要处理流程是 `internal/lex` → `internal/parse` 和 `internal/ast` → `internal/check` → `internal/lower` 和 `internal/ir` → `internal/providers`。标准标签位于 `internal/vocab`，命令组装位于 `internal/cli`，YAML 配置和环境变量映射位于 `internal/config`。测试与对应代码放在一起。

使用简短、符合 Go 惯例的函数，优先沿用现有写法。解析逻辑应独立于厂商和 I/O，只有 `internal/cli` 导入 Cobra。适配器接收已解析的配置和 IR，不重新加载配置或解析脚本。标识符、错误、帮助文本和代码注释使用英文。注释应解释约束或不易看出的设计原因，避免复述代码。

报告 bug 时，请提供最小 `.tbx` 脚本、执行命令、预期与实际结果，以及 `timbrex version` 输出。厂商问题还需模型和区域信息，并去除凭证、私有文本和签名音频 URL。重大的语法、架构或兼容性变更请先开 issue 讨论；范围明确的修复或有文档依据的厂商接入可以直接提交 pull request。

## 运行检查

开发过程中可先测试相关包，例如 `go test ./internal/parse`。提交前执行与 CI 一致的检查：

```bash
make tidy
make test
make lint
```

`make tidy` 检查 `go.mod` 或 `go.sum` 是否需要变化，但不改写文件。`make test` 构建 `./timbrex` 并运行 `go test ./...`。`make lint` 运行 golangci-lint 及其格式检查；需要应用 Go 格式化时可执行 `make fmt`。CI 会在 Linux 和 macOS 上运行测试。

默认测试必须离线。用 `httptest` 验证厂商请求，为行为修复和边界情况补充回归测试。常规测试不能要求凭证或产生费用。修改 TextMate YAML 语法后，用 `go test ./editors/vscode -update` 重新生成随包提供的 JSON，详见 [编辑器指南](editors/vscode/readme.zh-cn.md)。

真实服务验证应单独进行，并由执行者明确启用。记录模型、音色、区域、请求参数、日期和结果，不记录秘密。API 是否接受请求与音频质量应分别说明，模拟请求不能证明这两项。

## 新增厂商

先阅读官方 API 文档，在适配器和 [接入指南](docs/providers.zh-cn.md) 中链接来源。核实认证方式、请求结构、音频格式，以及完成和错误响应。在 `internal/providers/<name>` 实现 `Provider`，通过 `init` 注册，由 CLI 空白导入；在 `internal/config` 添加默认值和环境变量映射。

在接入指南和 [能力参考](docs/capabilities.zh-cn.md) 中说明支持的模型、凭证、标签映射、数值范围、语言提示和限制。适配器无法表达的控制应报错。测试完整的 CLI 到请求路径、畸形响应和失败处理。只有代码确实被多处复用时，才将其放到 `internal/providers`。

## 编写和审查文档

每个页面各司其职：README 介绍项目和首次运行，教程带读者完成脚本，规范说明语言规则，配置与厂商页面说明接入方法，设计说明解释取舍。详细内容用链接引用，避免多处重复。

面向初次接触项目的开发者写作：说明功能的用途，给出小示例，在限制会影响使用时就地解释。少用口号，不写缺乏依据的结论或对未来兼容性的承诺。中文应自然简洁，不必照搬英文句式；标识符和有助于理解的技术术语可保留英文。区分已实现行为、近似处理和提案，厂商行为引用官方来源。

英文和 `.zh-cn.md` 版本一起更新，示例与技术含义保持一致。文档文件名使用小写，`AGENTS.md` 除外。文档测试会解析所有 `tbx` 围栏示例和 `.tbx` 文件，检查本地文件链接、双语标题层级、Markdown 围栏和表格转义。测试通过不代表厂商支持该功能或音频质量合格，这类说明需要结合适配器代码和测试核实。

## 提交 pull request

每次变更尽量聚焦，提交消息使用英文 [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)，例如 `fix: preserve speaker after nested span`。说明解决的问题、修改后的行为和已通过的检查；有助于理解时给出简短的前后对比，并注明兼容性变化或尚未完成的验证。不提交密钥、`.env` 文件或生成的音频。贡献适用仓库的 [MIT 许可证](license)。
