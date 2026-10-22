# Timbrex 语言参考

[English](spec.md) · [厂商能力](capabilities.zh-cn.md) · [教程](tutorial.zh-cn.md)

**SPEC v0.3.4**。语法版本：**`timbrex: 1`**，语法已冻结。本参考描述语言规则，并单独说明当前的合成能力。

## 1. 文本与标签

`.tbx` 文件是 UTF-8 文本，`.timbrex` 是等价扩展名。开头的 BOM 会移除，CRLF/CR 换行统一为 LF。纯文本是合法脚本。

```tbx
Hello.
[happy|Hello again!] [pause 500ms] Welcome home.
```

标记只有 `[name args]` 和 `[name args|content]` 两种形式，可以嵌套。标签名、标识符和配置键使用英文 ASCII。台词、引号字符串、显示名称和自由表演指令可以使用任意语言。`［］`、`｜` 等全角字符是普通文本，不是替代语法。

标签头结束于引号参数之外第一个未转义的 `|` 或 `]`。内容中的双引号是普通台词标点，例如 `[whisper|She said "hello".]`。span 可以跨越非空行，但必须在同一段落内闭合。`[/whisper]` 这样的 BBCode 结束标签不合法。

| 形式 | 含义 |
| --- | --- |
| 发言中的 `[happy]` | 为本轮剩余台词设置情绪 |
| `[happy\|words]` | 只为这些文字设置情绪 |
| 独占一行的 `[happy]` | 为后续发言设置场景情绪 |
| `[pause 500ms]` | 在当前位置插入事件 |
| 独占一行的 `[whisper\|Hello]` | 将 span 作为旁白朗读 |

**场景行**只包含不带 span 的标签和可选空白。场景行中的事件发生在发言之间，场景状态持续到被替换或重置。

## 2. 角色与段落

空行结束段落。位于第零列且已声明的 `speaker_id:` 即使前面没有空行，也会开启新发言；后续普通行仍属于该发言。span 内看起来像角色前缀的文字仍是内容。

独立注释、场景行和原始围栏也会结束当前段落。后面的普通文字会成为旁白，除非带有已声明的角色前缀。若要在这些块之后继续同一角色的台词，应再次写出前缀。

```tbx
---
cast:
  lin: {name: 林}
  chen: {name: 陈}
---
lin: Hello.
Still Lin speaking.
chen: Hello back.

This is narration.
```

角色 ID 匹配 `[A-Za-z][A-Za-z0-9_.-]*`，区分大小写。只有 `cast` 中声明的 ID 才能开启发言，显示名称不是别名。未声明时，`note: hello` 是普通文本。`cast: auto` 接受所有行首 ASCII ID，`check` 会列出识别结果。`lin\: hello` 可转义可能的角色前缀。

旁白默认使用 `narrator`，也可以通过 front matter 的 `narrator:` 指定另一个 ASCII ID。在 cast 中为该 ID 配置音色以指定旁白音色。显示名称仅作元数据；适配器当前使用 ID 和音色映射，不会将显示名称作为表演指令。

## 3. 参数、转义与注释

参数以空格或制表符分隔。解析器识别位置参数和 `key=value`，但不套用厂商的参数 schema。

| 值类型 | 示例 |
| --- | --- |
| 数字 | `0.7`、`-1`、`+2.5` |
| 时长 | `500ms`、`1.5s`、`1min` |
| 百分比 | `120%` |
| 半音 | `+2st` |
| 分贝 | `-3dB` |
| 单词 | `warm`、`en-US` |
| 引号字符串 | `"holding back tears"`、`"轻轻地"` |
| 命名参数 | `intensity=0.5` |

数字后缀必须位于整个 token 末尾，`10stone` 因此是单词。引号值支持 `\"` 和 `\\`。注册标签可以允许尾随单词作为 note；当前允许 note 的标签是 `act` 和非语言事件。检查器尚不验证各标签的参数类型或范围。真实适配器在请求前验证自己实现的参数，当前都未实现命名参数映射。

合成中的数值控制要求使用类型匹配、不带引号的值。例如应写 `speed 1.2`，而非 `speed "1.2"` 或 `speed 1e0`。停顿支持 `ms`、`s`、`min` 时长单位；`1m`、`1s250ms` 等 Go 特有形式在此是单词，不是时长。零时长写为 `0ms`。

引号参数之外，`[[` 和 `\[` 表示字面量 `[`，`]]` 和 `\]` 表示字面量 `]`。反斜杠还可以转义 `|`、`\` 和 `:`，其他反斜杠按字面处理。打开两层或更多标签时，连续的 `]` 用于闭合嵌套标签，此时字面量右括号应写 `\]`。因此 `[whisper|say [pause 1s]]` 会闭合两层标签。

标签之外，一行第一个非空白字符起为 `//` 时是注释。行内 `//`、`#` 和 HTML 注释都是台词。圆括号和尖括号属于普通语言文本，但某些厂商将其保留为原生标记，详见 [能力表](capabilities.zh-cn.md#字面量文本)。

原始围栏以三个反引号和 info 字符串开始。info 第一个词必须是 ASCII 标识符，后面是原样保留的 info 文本；解析器当前只验证第一个词。仅含三个反引号及可选空白的一行结束围栏。围栏内部不解析 Timbrex。当前所有真实适配器都拒绝围栏；能够解析不代表支持透传。

## 4. 标签解析与严格模式

注册标签名不区分大小写，并把 `_` 视为 `-`：`[HAPPY]`、`[clear_throat]` 分别解析为 `happy`、`clear-throat`。角色 ID 和参数值不做此规范化。

显式表演指令写作 `[act "holding back tears"]`，span 形式为 `[act "holding back tears"|I'm fine.]`。也接受不带引号的 note：`[act holding back tears]`。所选厂商必须支持指令。

默认情况下，未知标签被视为隐式表演指令。`[thoughtfully]` 会成为保留原文的 `act` 指令。使用 `strict_tags: true` 或 `--strict-tags` 后，未知标签报错；`--no-strict-tags` 可反向覆盖 front matter。不能同时指定两个参数。对于提交到仓库的脚本，建议采用显式 `act` 和严格模式，防止拼写错误变成表演指令。

解析器识别 `ns:tag` 的形状，但尚未注册厂商命名空间或外部词表。这些名字当前遵循未知标签规则，不会直接调用厂商 API。

| 代码 | 当前诊断 |
| --- | --- |
| `V012` | 标签未闭合 |
| `V013` | 非法 BBCode 风格结束标签 |
| `V041` | 严格模式下未注册的标签 |
| `W040` | 非严格模式下接近注册名称的未知单词标签，附带建议 |
| `W103` | 使用了 cast 显示名称而非角色 ID |
| `I050` | 非 ASCII 自由表演文本 |

编辑距离不超过 2。`[paus]` 会建议 `[pause]`；`[laff]` 与 `[laugh]` 距离为 3，不会得到该建议。警告和信息不会使 `check` 失败，错误会。其他解析错误和位置错误可能没有诊断代码。没有自动翻译或拼写修复。

## 5. 作用域与词表

词表声明四种类型：

| 类型 | 行为 |
| --- | --- |
| State | 点标签持续到本轮结束，场景行中的标签持续生效，span 只作用于局部 |
| Event | 在当前位置发生，仅指定事件允许伴随台词的 span |
| Wrap | 必须带内容，例如 `[lang en-US\|Hello]` |
| Turn | 只能位于发言开头或场景行 |

内层控制覆盖同一特征的外层控制，不同特征可以在厂商能表达时共同生效。共享规划器分别将 `speed/fast/slow`、`voice/as`、`whisper/shout` 和情绪归为同一特征。退出 span 后恢复外层状态，每次新发言从场景状态开始。

发言中的 `[reset]` 恢复场景状态。`[reset speed]` 只恢复场景语速，没有场景语速时移除本轮语速。场景行的 reset 移除场景状态。`as` 选择另一角色的音色，必须写成 span。目标必须是 `cast` ID，或脚本中已经出现过的发言/旁白角色。已知角色没有对应厂商映射时仍使用默认音色；未知目标会报错，不会静默回退。`voice` 直接选择厂商音色 ID，也必须写成 span。

标准词表包含以下类别。**这是语言清单，不是厂商能力承诺。** 可执行列表见 [internal/vocab/vocab.go](../internal/vocab/vocab.go)。

| 类别 | 名称 |
| --- | --- |
| 情绪（state） | `neutral happy sad angry fear surprise disgust calm excited tender serious curious sarcastic nervous tired bored contempt confused proud embarrassed hopeful disappointed panicked crying` |
| 表演（state） | `emotion act speed fast slow pitch volume whisper shout emph style role accent oral dialect emovec` |
| 状态重置 | `reset`（无 span） |
| 停顿和时间（event） | `pause silence duration break rate` |
| 非语言声音（event，可带 span） | `laugh sigh breath cough gasp inhale exhale quick-breath pant huff groan grunt moan chuckle giggle cackle snort scoff cry sob sniffle sniff whimper wail clear-throat sneeze hiccup yawn burp snore swallow gulp lip-smack tongue-click kiss shush whistle hum hiss hmm uh um emm` |
| 发音和文本（wrap） | `pinyin ipa sub say arpabet jyutping romaji spell verbatim latex lang` |
| 音色（wrap） | `as voice` |
| 发言时间 | `overlap gap` |
| 音频和标记（event） | `sfx audio bgm ambience vc cut mark` |

## 6. Front matter

YAML front matter 可选，必须位于文件开头的两行 `---` 之间。开头和结尾的 `---` 行可以带有尾随空格或制表符。

```yaml
---
timbrex: 1
provider: openai
model: gpt-4o-mini-tts
voice: coral
lang: en-US
strict_tags: true
narrator: guide
cast:
  guide:
    name: Guide
    voices:
      openai: coral
      azure: en-US-JennyNeural
---
```

支持的键为 `timbrex`、`provider`、`model`、`voice`、`lang`、`strict_tags`、`narrator`、`cast`。`cast` 可以是 `auto`，也可以是 ID 映射，其中可选 `name` 和 `voices`。若提供 `timbrex`，必须为 `1`。未知 YAML 键会报错，包括拼错的 cast 字段；不会静默接受尚未实现的配置。

优先级见 [配置](configuration.zh-cn.md)。音色 ID 和模型属于具体厂商。`lang` 是由适配器解释的提示，不支持的提示可能报错。Azure 通过音色名称选择模型，因此拒绝 `model:`。凭证应放在环境变量中，不能写入脚本。

## 7. 合成契约

`check` 验证语言和词表，`synth` 还会验证当前适配器的映射，然后顺序渲染。用于语音片段的不支持控制、原始围栏和无效参数会在第一个付费请求前报错。发声前就被覆盖的状态没有听觉效果。

`pause`、`silence`、`break` 和非负的 `gap` 生成为本地静音。默认停顿 250 ms；`short`、`medium`、`long` 分别为 250 ms、500 ms、1 s。单次显式停顿支持 0 到 1 分钟。`0ms` 会被接受，但不产生采样。`fast` 为 1.2 倍速，`slow` 为 0.8 倍速。`speed 120%` 表示 1.2 倍速，不是增加 120%。音高使用半音，音量使用 dB，支持范围取决于适配器。仅在明确支持的适配器中，情绪强度才会被转成文本提示，并非校准后的测量值。

渲染器将兼容的原始 PCM 片段拼接成一个 WAV，不进行混音、重叠、重采样、归一化或淡入淡出。发言之间没有隐式停顿，需要时应明确添加。具体映射和限制见 [厂商接入](providers.zh-cn.md) 及 [能力表](capabilities.zh-cn.md)。

## 8. 后续方向，非当前行为

冻结语法为外部词表、注册厂商命名空间、原始内容透传、能力声明、锁文件，以及语音正文之外的数据集模板保留了空间。外部 YAML 配置已经实现，见 [配置](configuration.zh-cn.md)。发音控制、混音、降级规划、流式、原生对话和音色管理都需要独立实现与测试。语法不包含循环或条件分支。

`fix`、`fmt`、`expand`、`compile`、`caps`、`voices`、`tags`、`migrate`、LSP 和公开 Go SDK 均未交付。旧设计草图不构成兼容性承诺。后续工作须以中英文记录已实现行为。
