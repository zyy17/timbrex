# 已实现的厂商能力

[English](capabilities.md) · [配置与官方参考](providers.zh-cn.md)

此表描述 Timbrex 当前适配器，不代表厂商提供的全部能力。**参数**表示数值 API 字段或 SSML 控制；**指令**表示期望的表演，模型可能近似理解；**不支持**表示用于语音片段时会在本地报错。音色和模型限制仍然适用。这些映射有离线契约测试，尚无真实声音验证。

| 控制 | OpenAI | ElevenLabs | MiniMax | Azure | Qwen |
| --- | --- | --- | --- | --- | --- |
| 文本与 cast 音色 | 支持 | 支持 | 支持 | 支持 | 支持 |
| `pause`、`silence`、`break`、非负 `gap` | 本地静音 | 本地静音 | 本地静音 | 本地静音 | 本地静音 |
| `speed`、`fast`、`slow` | 参数：0.25–4 倍 | 参数：0.7–1.2 倍；v4/v4 Turbo 不支持 | 参数：0.5–2 倍 | SSML：0.5–2 倍 | 参数：0.5–2 倍 |
| `pitch` | 不支持 | 不支持 | 整数 −12…+12 半音 | SSML −12…+12 半音 | 参数 −12…+12 半音 |
| `volume` | 不支持 | 不支持 | 参数 −60…+20 dB | SSML −60…+20 dB | 参数 −60…约 +6.02 dB |
| 标准情绪 | 指令 | Audio-tag 指令 | 七种映射情绪 | 映射到 style | 指令 |
| 情绪强度，如 `happy 0.8` | 文本指令 | 文本指令 | 不支持 | 不支持 | 文本指令 |
| `whisper` / `shout` | 指令 | Audio tags | 仅 2.6 支持耳语，不支持喊叫 | 依赖音色的 style | 指令 |
| `act`、`accent`、`dialect`、自定义 `emotion` | 指令 | Audio-tag 指令 | 不支持 | 不支持 | 指令 |
| `style`、`role` | 指令 | Audio-tag 指令 | 不支持 | SSML；role 需要 style | 指令 |
| `emph` | 指令 | Audio-tag 指令 | 不支持 | 依赖音色的强调 | 指令 |
| `laugh`、`cry`、`sigh` 的 **span** | 指令 | Audio-tag 指令 | 不支持 | 不支持 | 指令 |
| 独立非语言事件 | 不支持 | 部分事件 | 2.8 下的部分事件 | 不支持 | 部分事件 |
| `lang` span / front matter | 指令 | 语言代码 | Language boost | SSML locale | 语言提示 |
| `voice`、`as` span；`reset` | 共享规划器 | 共享规划器 | 共享规划器 | 共享规划器 | 共享规划器 |

OpenAI 指令要求模型支持 `instructions`，`tts-1` 和 `tts-1-hd` 会拒绝。ElevenLabs 表演和事件要求 `eleven_v3`、`eleven_v4` 或 `eleven_v4_turbo`。`eleven_v4` 和 `eleven_v4_turbo` 不支持语速设置，因此会拒绝显式语速控制；其他模型的适配器语速范围保守地采用所链接产品指南中的范围。其他范围和语言限制见[厂商接入](providers.zh-cn.md)。

## 具体映射

MiniMax 情绪映射为：`happy→happy`、`sad→sad`、`angry→angry`、`fear→fearful`、`disgust→disgusted`、`surprise→surprised`、`calm→calm`。`whisper` 与情绪占用相同 API 字段，不能同时使用。`speech-2.8-hd` 和 `speech-2.8-turbo` 支持的事件为：

```text
laugh chuckle cough clear-throat groan breath pant inhale exhale gasp
sniff sigh snort burp lip-smack hum hiss emm sneeze
```

Azure style 映射为：`happy→cheerful`、`sad→sad`、`angry→angry`、`fear→fearful`、`excited→excited`、`calm→calm`、`tender→gentle`、`serious→serious`、`whisper→whispering`、`shout→shouting`、`crying→crying`、`neutral→neutral`。`[style "vendor-style"]` 传递 style 名称。每个片段只能使用一个 style，且所选音色必须支持；Azure 可能忽略不支持的 style。Timbrex 当前不会探测音色能力。

ElevenLabs 事件：`laugh`、`sigh`、`gasp`、`clear-throat`、`giggle`、`cough`。Qwen 事件：`laugh`、`sigh`、`gasp`、`clear-throat`、`giggle`、`cough`、`snort`。事件参数和 note 尚不支持，包括 `[laugh nervously]`；请改用已支持的伴随台词 span 或表演指令。

`[laugh|Hello]` 请求边笑边说，`[laugh]` 请求独立事件，两者是不同能力。指令不能保证精确的非语言声音时长，也不能保证孤立音效。

音量通过 `10^(dB/20)` 将 dB 转为线性振幅比。MiniMax 接收该比值，Azure 接收相对百分比，Qwen 从默认音量 50 缩放并取整。Qwen 音高通过 `2^(st/12)` 将半音转为比值。这些是参数映射，不是对实际输出响度或音高精度的测量。

## 字面量文本

转义保护 Timbrex 语法，但不能关闭厂商的原生标记。为防止字面量文本变成意外指令，ElevenLabs 和 Qwen 拒绝台词中的 ASCII `[`、`]`。ElevenLabs 也拒绝表演指令中的方括号、旧模型中的字面量 `<break>` 标签，以及 `eleven_flash_v2` 和 `eleven_turbo_v2` 中的 `<phoneme>` 标签。数学比较及无关的类似标签词语仍被接受。

MiniMax 拒绝 `(laughs)` 等已识别的原生事件字符串，以及停顿前缀 `<#`。其他圆括号会原样传递，但服务可能将括号内的拼音、IPA 或粤拼解释为发音控制；适配器无法识别所有此类形式。需要保留这些符号为文字时，可以使用 `（ ）`、`［ ］` 等 Unicode 替代字符，或换用其他厂商。

Azure 对文本和属性进行 XML 转义，OpenAI 将台词和指令放在不同字段。这避免了编译器生成的结构注入，但不保证生成模型逐字朗读每个符号。

## 已保留，尚未实现

所有真实适配器当前都拒绝发音包装标签（`ipa`、`pinyin`、`sub`、`say` 等）、音频文件/音效/背景音乐、`overlap`、`cut`、`duration`、`rate`、`oral`、`emovec`、标记、音色转换和原始围栏。没有自动降级或静默丢弃不支持控制的行为。命名标签参数可以解析，但尚无适配器映射。

Stub 忽略表演语义，输出 8 kHz 测试音，不能验证厂商能力或语音质量。`check` 识别完整词表，但不选择适配器、不验证厂商参数范围，也不发出请求。
