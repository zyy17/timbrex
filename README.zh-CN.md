# Timbrex

[English](README.md)

Timbrex 是一种面向表现力语音合成的标签脚本语言。`.tbx` 文件就是普通文本，再加上一种标记：`[name]` 或 `[name|content]`。同一份脚本可以编译到多个语音服务。

语法版本：`timbrex: 1`。设计说明：[docs/SPEC.zh-CN.md](docs/SPEC.zh-CN.md)（英文主文档：[docs/SPEC.md](docs/SPEC.md)）。

**状态：M0。** 编译器覆盖 v0.3.4 的主要表面：普通文本、`[tag]` 与 `[tag|content]`、参数、说话人行、cast、`//` 注释、YAML front matter、场景行，以及原文围栏。`timbrex check` 使用标准词表。`--strict-tags` 会拒绝未注册标签（批量模式的规则），并仍然接受 `[act "…"]`。`timbrex synth` 把脚本降到 IR 再调用 provider。当前树里只有 `stub`，它写出一段短促的 WAV 音调，并不合成语音。

## 目标

1. **小编译器。** 管线是 lex、parse、IR、lower。已经有可靠库能做的事就用库（命令行、YAML front matter，以及以后的 HTTP 或音频封装）。不要做成编译器框架。
2. **从脚本得到音频。** `timbrex check` 做检查。`timbrex synth` 写出音频文件。每个 provider 是独立包，共用一个接口，增加厂商时不改语法。
3. **代码易读。** 包短、函数短、少绕一层。

## 示例

见 `examples/hello.tbx`。

```sh
make build
./timbrex check examples/hello.tbx
./timbrex synth examples/hello.tbx -o hello.wav
```

这里的 `hello.wav` 是 stub 的音调，不是语音。台词和引号字符串可以用任何语言。标签名、参数名和配置键是英文 ASCII。

## 构建

需要 Go 1.27。

```sh
make build   # 生成 ./timbrex
make test
make lint
make fmt
make tidy    # go.mod 或 go.sum 会变化时失败
```

`timbrex version` 打印工具版本。`timbrex synth --provider` 选择已注册的 provider。未指定时用 front matter 的 `provider:`，再没有则是 `stub`。

`timbrex check --strict-tags` 是批量规则：未注册的 `[tag]` 报错 V041，自由表演要写成 `[act "…"]`。`--no-strict-tags` 强制单脚本规则。两个开关都不用时，看 front matter 的 `strict_tags`；普通脚本默认允许自由表演。单个未注册词与标准标签的编辑距离不超过 2 时给出警告 W040，命令仍然成功。非 ASCII 的自由表演是 info I050。`[laff]` 与 `[laugh]` 的距离是 3，所以不给建议。`[paus]` 会建议 `[pause]`。

## 编辑器高亮

TextMate 语法覆盖上述记号。VS Code 和 Cursor 用语言扩展加载。Sublime Text 以及其他 TextMate 编辑器可以直接指向同一份语法文件。Neovim 和 Helix 需要 tree-sitter，M0 不包含。

```sh
cd editors/vscode
cursor --install-extension .    # 或者 code --install-extension .
```

说明：[editors/vscode/README.md](editors/vscode/README.md)。语法文件：[editors/vscode/syntaxes/tbx.tmLanguage.yaml](editors/vscode/syntaxes/tbx.tmLanguage.yaml)。安装后重载窗口，打开 `.tbx`。语言 id 是 `timbrex`。`.timbrex` 使用同一语言。

## 文档

英文是主文本。中文对应文件使用 `.zh-CN.md` 后缀。

- [docs/SPEC.zh-CN.md](docs/SPEC.zh-CN.md) — 语言 v0.3.4（[English](docs/SPEC.md)）
- [docs/CAPABILITIES.zh-CN.md](docs/CAPABILITIES.zh-CN.md) — provider 能力（[English](docs/CAPABILITIES.md)）
- [docs/MARKET_SURVEY.zh-CN.md](docs/MARKET_SURVEY.zh-CN.md) — 市场与声音能力（[English](docs/MARKET_SURVEY.md)）
- [docs/REFLECTION.zh-CN.md](docs/REFLECTION.zh-CN.md) — 语法为何如此（[English](docs/REFLECTION.md)）

## 许可

[MIT](LICENSE)
