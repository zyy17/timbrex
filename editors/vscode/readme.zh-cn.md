# VS Code 与 Cursor 的 Timbrex 支持

[English](readme.md)

为 `.tbx` 和 `.timbrex` 提供语法高亮。词法遵循 SPEC v0.3.4。语言 ID 为 `timbrex`，scope 为 `source.tbx`。扩展只提供高亮，不提供补全、验证或 LSP。请运行 `timbrex check` 验证脚本。

## 试用扩展

在仓库根目录启动扩展开发窗口：

```sh
code --extensionDevelopmentPath="$PWD/editors/vscode" "$PWD/examples/expressive.tbx"
# 也可以使用 Cursor CLI：
cursor --extensionDevelopmentPath="$PWD/editors/vscode" "$PWD/examples/expressive.tbx"
```

本地安装时，将 `editors/vscode` 复制到编辑器扩展目录，命名为 `timbrex-local` 等名称，然后重新加载窗口。VS Code 通常使用 `~/.vscode/extensions`，Cursor 通常使用 `~/.cursor/extensions`；自定义配置和远程环境可能使用其他位置。仓库中没有已发布的 Marketplace 包。`code --install-extension` 接受扩展 ID 或 VSIX，不接受这个源码目录。

## 语法文件

编辑源文件：[syntaxes/tbx.tmLanguage.yaml](syntaxes/tbx.tmLanguage.yaml)。随包交付的运行时文件：[syntaxes/tbx.tmLanguage.json](syntaxes/tbx.tmLanguage.json)。VS Code 要求 JSON 格式，详见官方 [语法高亮指南](https://code.visualstudio.com/api/language-extensions/syntax-highlight-guide#using-yaml-to-write-a-grammar)。

修改 YAML 后，在仓库根目录重新生成：

```sh
go test ./editors/vscode -update
```

普通测试会检查 JSON 与 YAML 是否一致，以及扩展 manifest 是否引用正确文件。兼容 TextMate 的编辑器可以复用 JSON。尚未提供供 Neovim 或 Helix 使用的 tree-sitter 语法。

高亮覆盖 front matter、整行注释、原始围栏、标签和嵌套 span、引号标签参数、数字后缀及转义。span 内容中的引号仍是普通台词。第零列的 ASCII `ident:` 都会按角色着色；高亮器无法读取 cast，只有编译器能判断它是否真的开启发言。语法高亮只是视觉辅助，不能替代解析器。
