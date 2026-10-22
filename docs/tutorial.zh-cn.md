# 教程

[English](tutorial.md) · [语言参考](spec.zh-cn.md)

Timbrex 将 `.tbx` 脚本编译为 WAV 文件。本页对当前 CLI（`timbrex 0.1.0`）执行 `check` 和 `synth`。CLI 没有 `compile` 或 `inspect` 命令。

## 前置条件

需要 Go 1.27，版本见 [go.mod](../go.mod)。

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex version
```

```text
timbrex 0.1.0
```

在仓库根目录，`go run .` 与 `./timbrex` 是同一程序。没有 `cmd/timbrex` 包。后续步骤在 `make build` 之后使用 `./timbrex`。

## 编写脚本

创建 `hello.tbx`。纯文本是合法脚本。标签可选。

```tbx
你好。[pause 300ms] 欢迎回家。
```

标记形式为 `[name]`、`[name args]` 或 `[name args|content]`。标签名使用英文 ASCII。台词可用任意语言。

下列脚本与 [examples/expressive.tbx](../examples/expressive.tbx) 一致：

```tbx
---
timbrex: 1
strict_tags: true
---
[happy|很高兴见到你！] [pause 300ms] [slow|慢慢来。]
```

`timbrex: 1` 是冻结的语法版本（[SPEC v0.3.4](spec.zh-cn.md)）。`strict_tags: true` 拒绝未知标签名。front matter 可选。若使用，必须位于文件开头的两行 `---` 之间。开头和结尾的 `---` 行可以带有尾随空格或制表符。

## 检查脚本

`check` 解析文件并核对标签。不使用凭证或网络。

```bash
./timbrex check hello.tbx
```

```text
ok
```

仓库示例：

```bash
./timbrex check examples/hello.tbx
./timbrex check examples/expressive.tbx
./timbrex check examples/dialogue.tbx
./timbrex check examples/gemini.tbx
./timbrex check examples/fish.tbx
```

成功时在标准输出打印 `ok`。警告和信息写入标准错误，仍打印 `ok`。错误会使命令失败。

未知标签名会变为自由表演指令，除非开启严格模式。`--strict-tags` 与 `--no-strict-tags` 覆盖 front matter 和配置。

```bash
./timbrex check hello.tbx --strict-tags
```

下列内容在严格模式下不合法。该块使用 `text` 围栏，避免 `go test` 将其解析为脚本：

```text
[thoughtfully] Hello.
```

```text
script.tbx:1:1: error[V041]: unregistered tag "thoughtfully"; for free-form acting write [act "thoughtfully"]
error: check failed
```

自由表演写作 `[act "thoughtfully"]`。

## 查看配置

`check` 验证语言。`config` 打印合并后的文件与环境变量设置。已配置的 API key 显示为 `REDACTED`。`config` 不读取脚本，也不应用 `synth` 参数。

```bash
./timbrex config
```

未设置厂商环境变量时，内置默认值为：

```yaml
provider: stub
strict_tags: false
providers:
    azure:
        voice: en-US-JennyNeural
    elevenlabs:
        model: eleven_v3
        endpoint: https://api.elevenlabs.io/v1
    fish:
        model: s2.1-pro
        endpoint: https://api.fish.audio/v1
    gemini:
        model: gemini-3.8-flash-tts
        voice: Kore
        endpoint: https://generativelanguage.googleapis.com/v1beta
    minimax:
        model: speech-2.8-hd
        voice: English_expressive_narrator
        endpoint: https://api.minimax.io/v1
    openai:
        model: gpt-4o-mini-tts
        voice: coral
        endpoint: https://api.openai.com/v1
    qwen:
        model: qwen-audio-3.1-tts-flash
        voice: longanhuan_v3.1
    stub: {}
```

`check` 接受 `cast: auto`，并在标准错误中为每个识别出的角色列出一次：

```tbx
---
timbrex: 1
cast: auto
---
lin: 你好。
chen: 嗨。
```

```text
script.tbx:5:1: info: cast auto recognized speaker "lin"
script.tbx:6:1: info: cast auto recognized speaker "chen"
ok
```

## 用 stub 合成

`stub` 是默认厂商。它按脚本文字的 rune 数写出 8 kHz 测试音。它不是语音引擎，也不需要密钥。

```bash
./timbrex synth hello.tbx --provider stub -o tone.wav
```

```text
tone.wav
```

标准输出为输出路径。`-o` / `--out` 必须以 `.wav` 结尾。省略 `-o` 时，默认为脚本路径将扩展名换成 `.wav`（此处为 `hello.wav`）。不能覆盖源脚本。

```bash
./timbrex synth examples/expressive.tbx --provider stub -o tone.wav
```

```text
tone.wav
```

`file tone.wav` 报告 `WAVE audio, Microsoft PCM, 16 bit, mono 8000 Hz`。HTTP 适配器写出 24 kHz WAV。

## 用 OpenAI 合成

HTTP 适配器需要凭证。OpenAI 需要 `OPENAI_API_KEY`。未设置时，`synth` 在发请求前失败：

```bash
./timbrex synth examples/expressive.tbx --provider openai -o speech.wav
```

```text
error: set providers.openai.api_key or its documented environment variable
```

```bash
export OPENAI_API_KEY=your-key
./timbrex synth examples/expressive.tbx --provider openai --voice coral -o speech.wav
```

成功时标准输出为输出路径（`speech.wav`）。默认模型为 `gpt-4o-mini-tts`。本仓库未验证真实账号兼容性或声音质量。

厂商选择顺序：`--provider`，然后是 front matter 的 `provider:`，然后是配置（默认 `stub`）。[examples/dialogue.tbx](../examples/dialogue.tbx) 指定 `provider: openai`。若要对该文件写出本地测试音，传入 `--provider stub`。

其他适配器还有额外要求：ElevenLabs 和 Fish 需要音色 ID，Azure 需要区域或端点，Qwen 需要工作空间端点，Gemini 拒绝数值语速。见 [厂商接入](providers.zh-cn.md)。

## 声明角色

以 `id:` 开头的行只有在该 id 已列入 `cast`，或使用 `cast: auto` 时，才会开启发言。否则是旁白。旁白使用 `narrator`，除非 front matter 设置了 `narrator:`。

```tbx
---
timbrex: 1
cast:
  lin:
    name: Lin
    voices: {openai: coral}
  chen:
    name: Chen
    voices: {openai: onyx}
---
lin: 你来了！[pause 300ms] [whisper|进来吧。]
chen: [slow|我有事要告诉你。]
```

```bash
./timbrex check examples/dialogue.tbx
./timbrex synth examples/dialogue.tbx --provider stub -o dialogue.wav
```

```text
ok
dialogue.wav
```

`cast.<id>.voices.<provider>` 为该角色选择音色。`--voice` 是兜底值，不覆盖 cast 映射。显示名称不是角色别名。正文中的 `Lin:` 仍是旁白，并可能发出 `W103`。

## 插入停顿

`pause`、`silence`、`break` 和非负的 `gap` 由 HTTP 适配器使用的共享 PCM 渲染器生成本地静音，不会发送给厂商。`stub` 不插入静音，只按字符数写一段测试音。默认停顿为 250 ms。`short`、`medium`、`long` 分别为 250 ms、500 ms、1 s。显式时长使用 `ms`、`s` 或 `min`，范围为 `0ms` 到 `1min`。零时长写为 `0ms`。`[pause 0]` 是数字而不是时长，`check` 会拒绝。

```tbx
---
timbrex: 1
strict_tags: true
---
等一下。[pause 500ms] 现在走。[pause short] 再停一拍。
```

```bash
./timbrex check pauses.tbx
./timbrex synth pauses.tbx --provider stub -o pauses.wav
```

```text
ok
pauses.wav
```

发言之间没有隐式停顿。`overlap` 已保留。HTTP 适配器会拒绝该标签。

## 设置情绪与风格

点标签持续到本轮结束。span 只作用于其内容。一行只有不带 span 的标签时，为后续发言设置场景状态。`[reset]` 恢复场景状态。

```tbx
---
timbrex: 1
strict_tags: true
---
[happy]
[happy|很高兴见到你！] [pause 300ms] [slow|慢慢来。]
[act "holding back tears"|我没事。]
```

```bash
./timbrex check emotion.tbx --strict-tags
./timbrex synth emotion.tbx --provider stub -o emotion.wav
```

```text
ok
emotion.wav
```

独占一行的 `[happy]` 是场景状态，下一行旁白会继承它。`[happy|…]` 是局部 span。`[act "…"]` 是严格模式下要求的显式表演形式。

各适配器对这些标签的映射不同。OpenAI、Qwen 和 Gemini 将其中多数作为指令发送。MiniMax 映射七种情绪。Azure 映射 style。ElevenLabs 在 v3/v4 模型上使用 audio tags。Fish 映射部分 S2 标记。`stub` 忽略表演并写出测试音。面向 HTTP 适配器前见 [能力表](capabilities.zh-cn.md)。

## 批量合成

没有批量命令。使用 shell 循环。此处必须指定 `--provider stub`，因为 [examples/dialogue.tbx](../examples/dialogue.tbx)、[examples/gemini.tbx](../examples/gemini.tbx) 和 [examples/fish.tbx](../examples/fish.tbx) 指定了真实厂商。

```bash
mkdir -p out
for f in examples/*.tbx; do
  ./timbrex check "$f"
  ./timbrex synth "$f" --provider stub -o "out/$(basename "${f%.tbx}").wav"
done
```

```text
ok
out/dialogue.wav
ok
out/expressive.wav
ok
out/fish.wav
ok
out/gemini.wav
ok
out/hello.wav
```

`check` 不使用网络。不使用 `stub` 的 `synth` 需要对应厂商的凭证。

## 诊断

| 代码 | 级别 | 含义 |
| --- | --- | --- |
| `V012` | error | 标签未闭合 |
| `V013` | error | `[/name]` 不合法；span 写作 `[name\|...]` |
| `V041` | error | 严格模式下未注册的标签 |
| `W040` | warning | 接近注册名称的未知单词标签 |
| `W103` | warning | 把 cast 显示名称当成了角色前缀 |
| `I050` | info | 非 ASCII 自由表演文本 |

解析错误（如 `V012`）打印为 `file:line:col: …`。检查失败会再打印 `error: check failed`。

## 术语

| 术语 | 含义 |
| --- | --- |
| `.tbx` / `.timbrex` | 同一种语言的脚本扩展名 |
| 标签（tag） | `[name]`、`[name args]` 或 `[name args\|content]` |
| front matter | 位于首尾 `---` 之间的可选 YAML |
| cast | 已声明的角色 id；`cast: auto` 接受所有行首 ASCII id |
| 发言（turn） | 由已声明角色说出的一行及其后续行 |
| 旁白（narration） | 无前缀的段落；默认角色为 `narrator` |
| 场景行（scene line） | 只有不带 span 的标签的一行；在发言之间设置或触发 |
| span | `[name\|content]`，局部作用域 |
| `check` | 解析脚本并核对标签；打印 `ok` 或诊断 |
| `synth` | 先做同样的检查，再写出 WAV |
| `config` | 打印合并后的设置，密钥显示为 `REDACTED` |
| `stub` | 默认厂商；8 kHz 测试音，不是语音 |
| 严格模式 | 未知标签报 `V041`；应写成 `[act "…"]` |
| 厂商（provider） | `stub`、`openai`、`elevenlabs`、`minimax`、`azure`、`qwen`、`gemini` 或 `fish` |

## 另见

- [SPEC v0.3.4](spec.zh-cn.md) — 语法、词表、诊断
- [配置](configuration.zh-cn.md) — 参数、文件、环境变量
- [厂商接入](providers.zh-cn.md) — 凭证与请求形态
- [能力表](capabilities.zh-cn.md) — 适配器映射
