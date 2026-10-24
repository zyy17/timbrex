# 使用 agent 编写 Timbrex 脚本

[English](agent-skill.md) · [教程](tutorial.zh-cn.md)

[Timbrex skill](../skills/timbrex/SKILL.md) 让 agent 用一套流程完成需求转脚本、台词与表演修改、语言检查、厂商配置、按要求生成音频，以及故障排查。它是可移植的指令包，不是独立的 TTS 服务，也不会自动安装运行环境。

## 1. 准备 CLI

写脚本不需要厂商账号；检查脚本需要 Timbrex CLI。可以使用已安装且可信的程序，或用 Git 克隆仓库，再用 [go.mod](../go.mod) 指定的 Go 版本和 `make` 构建：

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex version
```

已有仓库时，跳过前两条命令，在仓库根目录构建。如果程序不在 PATH 中，请告诉 agent 程序或仓库的位置。Skill 不会自动安装 Timbrex、Go 或厂商凭证；获得授权后，agent 可以构建可信的本地仓库。没有 CLI 时仍可写脚本，但应说明尚未执行验证。

## 2. 安装并加载 skill

将完整的 `skills/timbrex` 目录复制到目标项目的技能目录，包括 `references` 和 `assets`。不要只复制 `SKILL.md`。

- Codex CLI/IDE：`.agents/skills/timbrex/`。
- Claude Code：`.claude/skills/timbrex/`。
- Cursor：`.cursor/skills/timbrex/`，也可复用 Codex 的 `.agents/skills/timbrex/`。

在目标项目根目录运行以下命令，先替换源路径，并选择一种目标目录：

```bash
skill_src=/absolute/path/to/timbrex/skills/timbrex
skill_parent=.agents/skills
mkdir -p "$skill_parent"
test ! -e "$skill_parent/timbrex" && cp -R "$skill_src" "$skill_parent/timbrex"
```

按所用工具将 `skill_parent` 改为 `.claude/skills` 或 `.cursor/skills`。如果目标已存在，应检查并更新已有副本，不要用首次安装命令覆盖。仓库中的分发目录 `skills/timbrex` 本身不是宿主自动发现技能的目录。

在对应工具中打开目标项目。在 AI 对话框中输入，而不是在终端运行：

- Codex：`$timbrex` 后接任务描述，或通过 `/skills` 选择。
- Claude Code：`/timbrex` 后接任务描述。
- Cursor：输入 `/`，选择 `timbrex`，再描述任务。

如果看不到技能，检查路径并重新打开或重启工具。这是本地项目安装，不是托管服务或插件安装。技能内容可以复用，但发现方式、可用工具和权限由宿主决定。安装参考：[Codex](https://learn.chatgpt.com/docs/build-skills)、[Claude Code](https://code.claude.com/docs/en/skills)、[Cursor](https://cursor.com/docs/skills)。

## 3. 写第一份脚本

调用 skill 后，可以这样描述任务：

```text
写一份 reunion.tbx：旁白、林和陈在空荡的站台重逢，使用中文。
整体克制、温暖，加几处有意义的停顿，以及一句耳语。
使用显式 cast ID 和严格标签模式。
用 /absolute/path/to/timbrex/timbrex 检查。
只写脚本和检查，不生成音频。
```

将程序路径替换为真实路径。预期结果是一份 `.tbx` 文件和如实报告的验证结果，而不是编造的音色 ID 或“已经听过”的说法。可以要求估计时长，但准确时长取决于合成结果。

也可以从随附的[对话示例](../skills/timbrex/assets/dialogue.tbx)开始。[作用域与转义示例](../skills/timbrex/assets/scopes-and-escaping.tbx)演示嵌套控制、重置、字面括号和注释。

## 4. 本地检查和文件生成测试

在仓库根目录试运行随附示例：

```bash
./timbrex check skills/timbrex/assets/dialogue.tbx --strict-tags
mkdir -p out
./timbrex synth skills/timbrex/assets/dialogue.tbx --provider stub -o out/dialogue-tone.wav
```

第一条命令打印 `ok`，不联系语音服务。第二条生成测试音，而不是朗读台词。Stub 不表现音色、停顿或表演指令。如果输出文件已存在，请换一个新路径。

通过 `check` 只表示语言检查通过，不代表厂商支持，也不会验证所有参数范围、账号权限或音色 ID。当前没有 `--dry-run` 或独立的厂商预检命令。

## 5. 准备好后生成语音

选择[厂商](providers.zh-cn.md)，核实当前模型可用性和[支持的控制](capabilities.zh-cn.md)，通过环境变量或密钥管理器私下配置凭证。不要把密钥写进脚本或聊天。Timbrex 不会自动加载 `.env`。

需要多音色时，可以让 agent 准备一份工作副本：

```text
将 skills/timbrex/assets/dialogue.tbx 复制为新的 dialogue.tbx。
使用 OpenAI 的 gpt-4o-mini-tts。为 narrator 和 lin 设置 coral，
为 chen 设置 onyx，写入各角色的 voices.openai。
检查脚本，但先不要合成。
```

这些 ID 受本版本支持，但仍需核实当前模型可用性和账号音色权限。显示名称和 `cast: auto` 不会自动选择不同音色；`--voice` 只是默认值，不会覆盖 cast 映射。

以下示例会把脚本发送给 OpenAI，可能产生费用。请先配置账号、查看[模型生命周期提示](providers.zh-cn.md#openai)、审阅脚本，并决定生成语音后再运行：

```bash
./timbrex synth dialogue.tbx --provider openai --model gpt-4o-mini-tts --voice coral -o out/dialogue.wav
```

准备后的工作副本让林和陈使用不同音色；未修改的随附示例没有映射，会让所有角色使用同一个默认音色。选择其他厂商时，要使用其模型、音色 ID 和支持的控制；Azure 通过音色选择模型，会拒绝 `--model`。

也可以明确授权 agent 生成语音，并说明厂商、音色、输出文件和费用限制。Agent 不应仅为测试兼容性而发送付费请求。生成后还要检查实际音频：WAV 格式有效不代表每个字和表演指令都符合预期。

## 6. 修改和排查

可以继续提出这样的任务：

```text
保留全部台词，让陈没那么紧张，不要改变林的音色。
解释这里为什么 reset 恢复了场景的平静状态，而不是变成中性。
把脚本适配到 Gemini，不要悄悄删掉不支持的控制。
分析这个错误，不要再次发送付费请求。
```

Skill 会区分语言检查、适配器兼容性和实际声音效果。背景音乐、重叠发言、音素标签或音色克隆等未实现的需求，应明确说明并提供可行替代方案。详细规则见[教程](tutorial.zh-cn.md)、[DSL 规范](spec.zh-cn.md)和[厂商指南](providers.zh-cn.md)。
