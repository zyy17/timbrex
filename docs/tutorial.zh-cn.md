# 教程

[English](tutorial.md) · [DSL 规范 v1](spec.zh-cn.md)

本教程先用一份简短脚本生成 WAV 文件，再加入角色和表演指令。本地操作不需要账号；生成真实语音的示例需要配置 TTS 厂商，并可能产生费用。

## 构建 CLI

安装 Go 1.27 和 `make`，在终端运行：

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex version
```

后续命令均在仓库根目录执行，使用刚构建的 `./timbrex`。

## 编写并检查脚本

创建 `hello.tbx`，写入：

```tbx
你好。[pause 300ms] 欢迎回家。
```

普通文本是要朗读的台词，`[pause 300ms]` 表示停顿 300 毫秒。标签是可选的，只写 `你好。` 也合法。

```bash
./timbrex check hello.tbx
```

```text
ok
```

`check` 解析脚本并检查标签，不联系厂商。错误会使命令失败；警告和提示写入 stderr，不影响 stdout 上的 `ok`。通过检查不表示某个厂商支持脚本中的全部指令。

## 生成测试文件

```bash
./timbrex synth hello.tbx --provider stub -o tone.wav
```

```text
tone.wav
```

用音频播放器打开 `tone.wav`，你会听到一段测试音，而不是脚本中的台词。`stub` 用于验证文件生成，不合成语音，也不插入指定的停顿。它是默认 provider，不需要凭证。

`-o` / `--out` 指定输出路径，必须以 `.wav` 结尾，且所在目录需要事先存在。省略 `-o` 时，`hello.tbx` 会在脚本所在目录生成 `hello.wav`。如果输出路径会覆盖源脚本，Timbrex 会报错。

## 生成语音

从[厂商接入](providers.zh-cn.md)中选择一个 provider。以 OpenAI 为例，通过环境变量 `OPENAI_API_KEY` **或**配置字段 `providers.openai.api_key` 提供密钥，然后运行：

```bash
# 运行前先配置 OpenAI 凭证。
./timbrex synth hello.tbx --provider openai --voice coral -o speech.wav
```

成功后打印 `speech.wav`。两处都没有配置凭证时，会在发送请求前报错：

```text
error: set providers.openai.api_key or its documented environment variable
```

不要把密钥写进脚本或提交到版本库。文件和环境变量的配置方式见[配置](configuration.zh-cn.md)。OpenAI 适配器默认使用 `gpt-4o-mini-tts`；使用前请查看[服务限制与生命周期说明](providers.zh-cn.md#openai)。其他适配器对模型、音色和端点的要求不同。

真实语音适配器生成 24 kHz、单声道、16 位 PCM WAV，默认串行合成台词，并在本地插入停顿。请求约定已有离线测试，但本仓库尚未验证真实账号兼容性和声音效果。

脚本含有多个语音片段时，可以为真实厂商尝试 `--concurrency 2`，让请求并行进行。默认值为 `1`，允许范围是 `1`–`8`。请遵守账号限额：更高的值会增加内存占用，也可能触发限流。并发不会改变片段文本、音色、控制、停顿或最终音频顺序。`stub` 只接受 `1`。失败处理与计费限制见[并发生成](providers.zh-cn.md#并发生成)。

## 添加指令与严格检查

标签有三种写法：`[name]`、`[name args]` 和 `[name args|content]`。标签名使用 ASCII 标识符，台词可以使用任意语言。

创建 `expressive.tbx`，写入以下内容。它与 [examples/expressive.tbx](../examples/expressive.tbx) 使用相同的标签结构：

```tbx
---
timbrex: 1
strict_tags: true
---
[happy|很高兴见到你！] [pause 300ms] [slow|慢慢来。]
```

```bash
./timbrex check expressive.tbx
```

文件开头的 YAML 部分称为 *front matter*，可以省略。使用时需要放在文件最前面，以两行 `---` 包围。这里的 `timbrex: 1` 指定当前 DSL 版本，`strict_tags: true` 拒绝未注册的标签名。[DSL 规范 v1](spec.zh-cn.md) 描述当前规则，开发期间可能调整。

不开启严格模式时，未注册的标签会被当作自由表演指令。这方便编写，也可能让拼写错误被忽略。需要检查标签名时，请开启严格模式，并把自由指令明确写成 `[act "thoughtfully"]`。

可以创建 `unknown.tbx`，写入下面这行，观察区别：

```text
[thoughtfully] Hello.
```

```bash
./timbrex check unknown.tbx --strict-tags
```

```text
unknown.tbx:1:1: error[V041]: unregistered tag "thoughtfully"; for free-form acting write [act "thoughtfully"]
error: check failed
```

将 `[thoughtfully]` 改为 `[act "thoughtfully"]` 后，就能通过这项严格检查。`--strict-tags` 与 `--no-strict-tags` 会覆盖 front matter 和配置中的设置。

## 理解作用范围

`happy`、`slow` 这类指令可以作用于三个范围：

- **Span**，例如 `[slow|慢慢来。]`，只影响括号中的台词。嵌套 span 结束后，恢复外层状态。
- **台词中的点标签**，例如 `你好。[sad] 我很想你。`，从标签位置持续到本次发言结束。没有角色前缀的段落算一次旁白发言，空行会结束该段落。
- **场景行**，即只有点标签的一行。在这里设置的状态影响后续发言，直到另一条场景行修改它。`[pause 1s]` 这类事件只执行一次，不会成为持续状态。

创建 `emotion.tbx`：

```tbx
---
strict_tags: true
---
[happy]
你好。[sad] 我很想你。[reset] 欢迎回来。

[slow|慢慢来。] [act "quietly relieved"|你终于到了。]
[reset]
再见。
```

```bash
./timbrex check emotion.tbx
./timbrex synth emotion.tbx --provider stub -o emotion.wav
```

第一行 `[happy]` 设置场景情绪。第一段中的 `[sad]` 改变当前情绪，`[reset]` 则恢复场景的 `happy`。第二段同样继承 `happy`，其中的 span 添加局部指令。最后独占一行的 `[reset]` 清除场景控制，因此“再见。”不再带有前面的指令。

在台词中，`[reset]` 恢复的是场景默认值，即使它位于嵌套 span 内，也不是恢复到紧邻的外层 span。`[reset speed]` 这类指定名称的写法只重置对应控制，详见[作用域参考](spec.zh-cn.md)。

上面的 stub 命令只验证文件生成。要听到表演效果，需要选择支持这些指令的厂商。数值控制与表演提示并不相同：例如，Gemini 将 `slow` 作为近似风格要求，但不支持数值 `speed`。切换厂商前请查阅[能力表](capabilities.zh-cn.md)。

## 使用多个角色

创建 `cast.tbx`：

```tbx
---
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
./timbrex check cast.tbx
./timbrex synth cast.tbx --provider stub -o cast.wav
```

配置凭证后，改用 `--provider openai` 就可以请求真实对话。每个 `voices.openai` 值指定对应角色的音色。`--voice` 只供没有 cast 映射的角色使用，不会覆盖这两项设置。

行首的 `id:` 只有在该 ID 已列入 `cast`，或 front matter 使用 `cast: auto` 时，才会开始角色发言。否则，前缀会保留为普通文字：在已有发言中仍由当前角色说出，另起一段时则成为旁白。显示名称不是别名：这里应写 `lin:`，而不是 `Lin:`。无前缀旁白默认使用 `narrator`，也可以通过 front matter 的 `narrator:` 指定其他 ID。

`cast: auto` 无需预先声明就能识别角色前缀，并在 `check` 时列出各角色 ID，但不会自动分配不同音色。需要区分音色时，应明确配置 cast 映射。`[as lin|台词]` 使用已知角色的映射，`[voice "vendor-id"|台词]` 则直接选择厂商音色。

## 插入停顿

创建 `pauses.tbx`：

```tbx
等一下。[pause 500ms] 现在走。[pause short] 再停一拍。
```

```bash
./timbrex check pauses.tbx
./timbrex synth pauses.tbx --provider stub -o pauses.wav
```

使用真实厂商时，`pause`、`silence`、`break` 和非负的 `gap` 会插入本地静音。`gap` 必须位于已识别的角色发言开头，在台词或状态标签之前，也可以放在场景行中。无角色前缀的旁白开头不能使用 `gap`，可改用 `pause`。默认时长为 250 ms；`short`、`medium`、`long` 分别为 250 ms、500 ms、1 s。显式时长使用 `ms`、`s` 或 `min`，范围为 `0ms` 到 `1min`。零时长写作 `0ms`：`[pause 0]` 是数字而非时长，无法通过 `check`。

发言之间不会自动添加停顿，需要时应显式写出。重叠发言尚未实现。上面的 stub 输出不会表现停顿；要听到停顿效果，需要切换到真实厂商。

## 复用配置

Provider、模型、默认音色和严格模式可以写入配置，不必每次都在命令中指定。优先级依次为：命令行参数、脚本 front matter、环境变量、选中的 YAML 文件、内置默认值。Cast 音色映射仍然优先于默认音色。

默认配置文件是当前工作目录下的 `timbrex.yaml`，也可以用 `--config` 选择其他文件。查看合并后的文件与环境变量设置：

```bash
./timbrex config
```

已配置的 API key 显示为 `REDACTED`。该命令不读取脚本、不应用 `synth` 参数，也不验证凭证。[配置参考](configuration.zh-cn.md)说明全部字段和覆盖规则，[厂商接入](providers.zh-cn.md)列出各适配器的默认值与环境变量。

部分仓库示例在 front matter 中选择了真实厂商。如果只想检查全部示例并生成本地测试文件，而不发送语音请求，可以运行：

```bash
mkdir -p out
for f in examples/*.tbx; do
  ./timbrex check "$f"
  ./timbrex synth "$f" --provider stub -o "out/$(basename "${f%.tbx}").wav"
done
```

## 排查结果

| 现象 | 检查方法 |
| --- | --- |
| `check` 打印 `ok`，`synth` 却提示不支持某个控制 | 语言检查与适配器支持范围不同。请在[能力表](capabilities.zh-cn.md)中核对所选模型。 |
| 输出只有一段测试音 | 当前使用的是默认 provider `stub`。请配置并选择真实厂商。 |
| `--voice` 没有改变角色音色 | 该角色有 cast 映射，或当前存在 `voice`/`as` 指令。`--voice` 只提供默认音色。 |
| 冒号前的名字被读了出来 | 在 `cast` 中声明完全一致的 ID，或使用 `cast: auto`。显示名称不能当作别名。 |
| 已导出环境变量，合成仍提示缺少密钥 | 核对当前 provider 及其变量名，用 `config` 查看合并结果。一个厂商的密钥不能配置另一个厂商。 |
| 没有生成输出文件 | 查看 stderr，检查输出目录、脚本诊断和厂商错误。真实语音合成失败会保留已有目标文件，但此前完成的请求可能已经计费。 |

能确定位置时，诊断会给出文件名、行号和列号。`V041` 表示严格模式下出现未注册标签，`V012` 表示标签未闭合，`W103` 表示把显示名称当成了角色前缀。完整诊断见 [DSL 规范](spec.zh-cn.md)，服务错误和限制见[厂商接入](providers.zh-cn.md)。
