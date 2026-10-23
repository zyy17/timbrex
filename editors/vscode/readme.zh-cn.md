# VS Code 与 Cursor 的 Timbrex 支持

[English](readme.md)

本扩展为 VS Code 和 Cursor 中的 `.tbx`、`.timbrex` 脚本提供语法高亮。语言规则见 [DSL 规范 v1](../../docs/spec.zh-cn.md)。扩展负责高亮，脚本验证使用 Timbrex CLI；暂不提供补全或 LSP。

## 加载扩展

确保编辑器的 CLI 已加入 `PATH`，然后在仓库根目录执行以下任一命令，打开直接加载本地扩展源码的开发窗口：

```bash
code --extensionDevelopmentPath="$PWD/editors/vscode" "$PWD/examples/expressive.tbx"
# 也可以使用 Cursor CLI：
cursor --extensionDevelopmentPath="$PWD/editors/vscode" "$PWD/examples/expressive.tbx"
```

该源码加载选项见 VS Code 官方 [扩展测试指南](https://code.visualstudio.com/api/working-with-extensions/testing-extension)。无需额外构建扩展，manifest 会加载仓库内的 JSON 语法文件。

如需保留本地源码安装，可将整个 `editors/vscode` 目录复制到编辑器扩展目录下，命名为 `timbrex-local` 等名称，再重新加载窗口。VS Code 的默认本地目录为 `~/.vscode/extensions`，Cursor 为 `~/.cursor/extensions`；远程或自定义环境可能使用其他位置。更新时替换这份目录并再次重新加载。仓库未提供 VSIX 文件。[`code --install-extension`](https://code.visualstudio.com/docs/configure/command-line#working-with-extensions) 接受扩展 ID 或 VSIX 路径。

## 编辑和验证脚本

打开 `.tbx` 或 `.timbrex` 文件，确认状态栏的语言显示为 **Timbrex**；若没有，使用 **Change Language Mode（更改语言模式）** 选择 Timbrex。在仓库根目录验证脚本：

```bash
go run . check examples/expressive.tbx
```

高亮覆盖 front matter、整行注释、原始围栏、标签和嵌套 span、标签头中的引号参数、数字后缀及转义。span 内容中的引号仍是普通台词。顶格的 ASCII `ident:` 都会按角色着色；高亮器无法读取 `cast`，由编译器判断该前缀是否真的开启发言。标签有高亮，也可能尚未注册或不受所选厂商支持。

## 维护语法文件

编辑源文件：[syntaxes/tbx.tmLanguage.yaml](syntaxes/tbx.tmLanguage.yaml)。随包交付的运行时文件：[syntaxes/tbx.tmLanguage.json](syntaxes/tbx.tmLanguage.json)。VS Code 要求 JSON 格式，详见官方 [语法高亮指南](https://code.visualstudio.com/api/language-extensions/syntax-highlight-guide#using-yaml-to-write-a-grammar)。

修改 YAML 后，在仓库根目录重新生成：

```bash
go test ./editors/vscode -update
```

运行时回归测试使用实际的 TextMate 分词器和 Oniguruma 引擎。安装 Node.js 24 后，在此扩展目录运行：

```bash
npm ci --ignore-scripts
npm test
```

CI 会在 Go 测试之外单独运行这些测试。依赖仅用于开发；复制扩展的用户不需要 Node.js 或 `node_modules`。

普通测试会检查 JSON 与 YAML 是否一致，以及扩展 manifest 是否引用正确文件。语言 ID 为 `timbrex`，TextMate scope 为 `source.tbx`。兼容 TextMate 的编辑器可以复用 JSON。尚未提供供 Neovim 或 Helix 使用的 tree-sitter 语法。

## 许可证

Copyright (c) 2026 zyy17

本扩展采用 [Apache License 2.0](license)。单独复制扩展目录时，请保留此版权声明和许可证文件。
