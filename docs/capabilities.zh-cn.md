# 已实现的厂商能力

[English](capabilities.md) · [配置与官方参考](providers.zh-cn.md) · [教程](tutorial.zh-cn.md)

下表按厂商列出 Timbrex 当前适配器的能力，不代表厂商提供的全部能力。**参数**表示数值 API 字段或 SSML 控制；**指令**表示期望的表演，模型可能近似理解；**不支持**表示用于语音片段时会在本地报错。音色和模型限制仍然适用。这些映射有离线契约测试，尚无真实声音验证。

OpenAI 指令要求模型支持 `instructions`，`tts-1` 和 `tts-1-hd` 会拒绝。ElevenLabs 表演和事件要求 `eleven_v3`、`eleven_v4` 或 `eleven_v4_turbo`。`eleven_v4` 和 `eleven_v4_turbo` 不支持语速设置，因此会拒绝显式语速控制；其他模型的适配器语速范围保守地采用所链接产品指南中的范围。其他范围和语言限制见 [厂商接入](providers.zh-cn.md)。ElevenLabs 在 `eleven_multilingual_v2`、主要语言子标签为空，或仅支持英语的 `eleven_flash_v2` / `eleven_turbo_v2` 使用非 `en` 提示时，会在本地报错。适配器不会查询其他模型的语言支持，服务可能忽略不支持的代码。

## 单次请求配置

强类型 YAML 选项与下方的 DSL 控制分开。准确的键、范围、默认值、示例和本地校验规则见[配置说明](configuration.zh-cn.md#厂商合成选项)。

- [ElevenLabs](https://elevenlabs.io/docs/api-reference/text-to-speech/convert)：seed 和文本规范化、四个单次请求音色设置，以及最多三个有序的现有词典引用。不检查词典内容或模型实际效果；不把 UI 可用性当作 HTTP 约定。
- [Fish](https://github.com/fishaudio/docs/blob/main/api-reference/openapi.json)：支持的 S2 模型可设置文本/响度规范化、规范化语言、采样、分块长度、延迟、前块上下文和提前停止。规范化语言不是朗读语言覆盖项。[发音词典](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech#body-pronunciation-dictionary) 接受最多三个带版本的引用或内联词典；后端匹配/替换覆盖整个请求，不是局部发音 span。无法解析的引用可能被静默丢弃。[`quality-guard`](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech#body-features) 是强类型可选标记，可用性由后端决定。
- [Azure](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/rest-text-to-speech#custom-voices)：通过不含查询参数的显式端点和不透明的 `deployment_id`，将所有片段路由到同一现有自定义音色部署。需要托管资源密钥和已部署的音色名称；不含语言信息的名称须显式设置语言。不改写主机名，不调用部署/目录 API。
- [Qwen](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api)：每片段 seed 和有序的 `input.hot_fix` 发音/替换对。重复项或重叠项的匹配优先级尚未明确。
- [MiniMax](https://platform.minimax.io/docs/api-reference/speech-t2a-http)：`basic` 或 `quality` 文本规范化；quality 只适用于非流式 2.8/2.6 HD/Turbo，其他组合会被拒绝。

这些新增选项保留省略字段时的默认行为，不实现内联发音 span、资源创建/管理、原生对话或流式请求。最终文件格式由独立的[本地导出阶段](audio-output.zh-cn.md)处理，不是新增厂商请求格式。契约测试是离线验证，不是真实账户或声学验证。

## 按厂商查询控制

[OpenAI](#openai) · [ElevenLabs](#elevenlabs) · [MiniMax](#minimax) · [Azure](#azure) · [Qwen](#qwen) · [Gemini](#gemini) · [Fish](#fish)

### OpenAI

| 控制 | 支持情况 |
| --- | --- |
| 文本与 cast 音色 | 支持 |
| `pause`、`silence`、`break`、非负 `gap` | 本地静音 |
| `speed`、`fast`、`slow` | 参数：0.25–4 倍 |
| `pitch` | 不支持 |
| `volume` | 不支持 |
| 标准情绪 | 指令 |
| 情绪强度，如 `happy 0.8` | 文本指令 |
| `whisper` / `shout` | 指令 |
| `act`、`accent`、`dialect`、自定义 `emotion` | 指令 |
| `style`、`role` | 指令 |
| `emph` | 指令 |
| `laugh`、`cry`、`sigh` 的 **span** | 指令 |
| 独立非语言事件 | 不支持 |
| `lang` span / front matter | 指令 |
| `voice`、`as` span；`reset` | 共享规划器 |

### ElevenLabs

| 控制 | 支持情况 |
| --- | --- |
| 文本与 cast 音色 | 支持 |
| `pause`、`silence`、`break`、非负 `gap` | 本地静音 |
| `speed`、`fast`、`slow` | 参数：0.7–1.2 倍；v4/v4 Turbo 不支持 |
| `pitch` | 不支持 |
| `volume` | 不支持 |
| 标准情绪 | Audio-tag 指令 |
| 情绪强度，如 `happy 0.8` | 文本指令 |
| `whisper` / `shout` | Audio tags |
| `act`、`accent`、`dialect`、自定义 `emotion` | Audio-tag 指令 |
| `style`、`role` | Audio-tag 指令 |
| `emph` | Audio-tag 指令 |
| `laugh`、`cry`、`sigh` 的 **span** | Audio-tag 指令 |
| 独立非语言事件 | 部分事件 |
| `lang` span / front matter | 语言代码 |
| `voice`、`as` span；`reset` | 共享规划器 |

### MiniMax

| 控制 | 支持情况 |
| --- | --- |
| 文本与 cast 音色 | 支持 |
| `pause`、`silence`、`break`、非负 `gap` | 本地静音 |
| `speed`、`fast`、`slow` | 参数：0.5–2 倍 |
| `pitch` | 整数 −12…+12 半音 |
| `volume` | 参数 −60…+20 dB |
| 标准情绪 | 七种映射情绪 |
| 情绪强度，如 `happy 0.8` | 不支持 |
| `whisper` / `shout` | 仅 2.6 支持耳语，不支持喊叫 |
| `act`、`accent`、`dialect`、自定义 `emotion` | 不支持 |
| `style`、`role` | 仅 2.6 支持 `style fluent`；无 role |
| `emph` | 不支持 |
| `laugh`、`cry`、`sigh` 的 **span** | 不支持 |
| 独立非语言事件 | 2.8 下的部分事件 |
| `lang` span / front matter | 40 种语言/方言 boost 和显式 auto；受模型限制 |
| `voice`、`as` span；`reset` | 共享规划器 |

### Azure

| 控制 | 支持情况 |
| --- | --- |
| 文本与 cast 音色 | 支持 |
| `pause`、`silence`、`break`、非负 `gap` | 本地静音 |
| `speed`、`fast`、`slow` | SSML：0.5–2 倍 |
| `pitch` | SSML −12…约 +7.01955 半音 |
| `volume` | SSML −60…+20 dB |
| 标准情绪 | 映射到 style |
| 情绪强度，如 `happy 0.8` | 不支持 |
| `whisper` / `shout` | 依赖音色的 style |
| `act`、`accent`、`dialect`、自定义 `emotion` | 不支持 |
| `style`、`role` | SSML；role 需要 style |
| `emph` | 依赖音色的强调 |
| `laugh`、`cry`、`sigh` 的 **span** | 不支持 |
| 独立非语言事件 | 不支持 |
| `lang` span / front matter | SSML locale |
| `voice`、`as` span；`reset` | 共享规划器 |

### Qwen

| 控制 | 支持情况 |
| --- | --- |
| 文本与 cast 音色 | 支持 |
| `pause`、`silence`、`break`、非负 `gap` | 本地静音 |
| `speed`、`fast`、`slow` | 参数：0.5–2 倍 |
| `pitch` | 参数 −12…+12 半音 |
| `volume` | 参数 −60…约 +6.02 dB |
| 标准情绪 | 指令 |
| 情绪强度，如 `happy 0.8` | 文本指令 |
| `whisper` / `shout` | 指令 |
| `act`、`accent`、`dialect`、自定义 `emotion` | 指令 |
| `style`、`role` | 指令 |
| `emph` | 指令 |
| `laugh`、`cry`、`sigh` 的 **span** | 指令 |
| 独立非语言事件 | 部分事件 |
| `lang` span / front matter | 语言提示 |
| `voice`、`as` span；`reset` | 共享规划器 |

### Gemini

| 控制 | 支持情况 |
| --- | --- |
| 文本与 cast 音色 | 支持 |
| `pause`、`silence`、`break`、非负 `gap` | 本地静音 |
| `speed`、`fast`、`slow` | `fast`/`slow` 为指令；数值不支持 |
| `pitch` | 不支持 |
| `volume` | 不支持 |
| 标准情绪 | 指令 |
| 情绪强度，如 `happy 0.8` | 文本指令 |
| `whisper` / `shout` | 指令 |
| `act`、`accent`、`dialect`、自定义 `emotion` | 指令 |
| `style`、`role` | 指令 |
| `emph` | 指令 |
| `laugh`、`cry`、`sigh` 的 **span** | 指令 |
| 独立非语言事件 | 部分事件 |
| `lang` span / front matter | 原生语言代码 |
| `voice`、`as` span；`reset` | 共享规划器 |

### Fish

| 控制 | 支持情况 |
| --- | --- |
| 文本与 cast 音色 | 支持 |
| `pause`、`silence`、`break`、非负 `gap` | 本地静音 |
| `speed`、`fast`、`slow` | 参数：0.5–2 倍 |
| `pitch` | 不支持 |
| `volume` | 参数 −60…+20 dB（本地范围） |
| 标准情绪 | 部分 S2 标记 |
| 情绪强度，如 `happy 0.8` | 不支持 |
| `whisper` / `shout` | S2 标记 |
| `act` | 自然语言 S2 提示 |
| `accent`、`dialect`、自定义 `emotion` | 不支持 |
| `style`、`role` | 不支持 |
| `emph` | S2 标记 |
| `laugh`、`cry`、`sigh` 的 **span** | 不支持 |
| 独立非语言事件 | 部分事件 |
| `lang` span / front matter | 不支持 |
| `voice`、`as` span；`reset` | 共享规划器 |


## 具体映射

MiniMax 情绪映射为：`happy→happy`、`sad→sad`、`angry→angry`、`fear→fearful`、`disgust→disgusted`、`surprise→surprised`、`calm→calm`。`whisper` 和 `[style fluent]` 与情绪占用相同 API 字段，要求 2.6 模型，不能彼此组合，也不能与另一种情绪组合。`speech-2.8-hd` 和 `speech-2.8-turbo` 支持的事件为：

```text
laugh chuckle cough clear-throat groan breath pant inhale exhale gasp
sniff sigh snort burp lip-smack hum hiss emm sneeze
```

MiniMax 主要语言代码映射不区分大小写。Locale 处理、`auto` 和模型限制见[接入说明](providers.zh-cn.md#minimax)。右侧是精确的 `language_boost` API 字符串：

```text
af→Afrikaans ar→Arabic bg→Bulgarian ca→Catalan
cs→Czech da→Danish de→German el→Greek
en→English es→Spanish fa→Persian fi→Finnish
fil→Filipino fr→French he→Hebrew hi→Hindi
hr→Croatian hu→Hungarian id→Indonesian it→Italian
ja→Japanese ko→Korean ms→Malay nl→Dutch
nn→Nynorsk no→Norwegian pl→Polish pt→Portuguese
ro→Romanian ru→Russian sk→Slovak sl→Slovenian
sv→Swedish ta→Tamil th→Thai tr→Turkish
uk→Ukrainian vi→Vietnamese yue→Chinese,Yue zh→Chinese
```

Azure style 映射为：`happy→cheerful`、`sad→sad`、`angry→angry`、`fear→fearful`、`excited→excited`、`calm→calm`、`tender→gentle`、`serious→serious`、`whisper→whispering`、`shout→shouting`、`crying→crying`、`neutral→neutral`。`[style "vendor-style"]` 传递 style 名称。每个片段只能使用一个 style，且所选音色必须支持；Azure 可能忽略不支持的 style。Timbrex 当前不会探测音色能力。

ElevenLabs 事件：`laugh`、`sigh`、`gasp`、`clear-throat`、`giggle`、`cough`。Qwen 事件：`laugh`、`sigh`、`gasp`、`clear-throat`、`giggle`、`cough`、`snort`。事件参数和 note 尚不支持，包括 `[laugh nervously]`；请改用已支持的伴随台词 span 或表演指令。

`[laugh|Hello]` 请求边笑边说，`[laugh]` 请求独立事件，两者是不同能力。指令不能保证精确的非语言声音时长，也不能保证孤立音效。

音量通过 `10^(dB/20)` 将 dB 转为线性振幅比。MiniMax 接收该比值，Azure 接收相对百分比，Qwen 从默认音量 50 缩放并取整。Qwen 音高通过 `2^(st/12)` 将半音转为比值。Azure 音高限制在其文档规定的 0.5–1.5 倍频率范围内，上限为 `12*log2(1.5)` 半音，避免请求可能被服务钳制或替代的数值。这些是参数映射，不是对实际输出响度或音高精度的测量。

## 字面量文本

转义保护 Timbrex 语法，但不能关闭厂商的原生标记。为防止字面量文本变成意外指令，ElevenLabs、Qwen 和 Fish 拒绝台词中的 ASCII `[`、`]`。Fish 还拒绝原生说话人 token 前缀 `<|`。ElevenLabs 也拒绝表演指令中的方括号、旧模型中的字面量 `<break>` 标签，以及 `eleven_flash_v2` 和 `eleven_turbo_v2` 中的 `<phoneme>` 标签。数学比较及无关的类似标签词语仍被接受。

MiniMax 拒绝 `(laughs)` 等已识别的原生事件字符串，以及停顿前缀 `<#`。其他圆括号会原样传递，但服务可能将括号内的拼音、IPA 或粤拼解释为发音控制；适配器无法识别所有此类形式。需要保留这些符号为文字时，可以使用 `（ ）`、`［ ］` 等 Unicode 替代字符，或换用其他厂商。

Eleven v4 还会将[斜杠包围的 IPA](https://elevenlabs.io/docs/overview/capabilities/text-to-speech/best-practices#ipa-with-eleven-v4) 解释为发音指引。包含斜杠的文本会原样传递；适配器不会一概拒绝斜杠，也不保证所有原生音标形式都保持字面语义。

Azure 对文本和属性进行 XML 转义，OpenAI 将台词和指令放在不同字段。这避免了编译器生成的结构注入，但不保证生成模型逐字朗读每个符号。

## 已保留，尚未实现

所有真实适配器当前都拒绝发音包装标签（`ipa`、`pinyin`、`sub`、`say` 等）、音频文件/音效/背景音乐、`overlap`、`cut`、`duration`、`rate`、`oral`、`emovec`、标记、音色转换和原始围栏。没有自动降级或静默丢弃不支持控制的行为。命名标签参数可以解析，但尚无适配器映射。

Stub 忽略表演语义，输出 8 kHz 测试音，不能验证厂商能力或语音质量，也不插入本地停顿。`check` 识别完整词表，并验证共享规划器使用的 pause/silence/break/gap 时长类型。它不选择适配器、不验证厂商参数范围，也不发出请求。

## Gemini TTS

Gemini 适配器支持 `gemini-3.8-flash-tts` 和 `gemini-3.8-flash-lite-tts`，可使用文本、预置角色音色、共用 voice/as/reset 作用域和本地停顿。标准情绪（含数值强度）、`act`、`style`、`role`、`accent`、`dialect`、自定义 `emotion`、`whisper`、`shout`、`emph` 及 laugh/cry/sigh span 会转换为 `speechMetadata.style` 中的**指令**。`fast` 和 `slow` 请求更快/更慢的语音，不承诺精确倍率。数值 speed、pitch 和 volume **不支持**。`lang` 是原生语言代码参数，允许的子集见接入说明。

两个模型均支持下表中的独立事件，映射依据官方 [TTS 原生发声标记](https://ai.google.dev/gemini-api/docs/generate-content/speech-generation#vocal-bursts)。

| DSL 事件 | 原生台词标记 |
| --- | --- |
| `[laugh]` | `<laugh>` |
| `[sigh]` | `<sigh>` |
| `[cough]` | `<cough>` |
| `[breath]` | `<breath>` |
| `[gasp]` | `<gasp>` |
| `[exhale]` | `<exhales>` |
| `[chuckle]` | `<chuckle>` |
| `[giggle]` | `<giggle>` |
| `[cackle]` | `<cackle>` |
| `[snort]` | `<snort>` |
| `[cry]` | `<cry>` |
| `[sob]` | `<sob>` |
| `[whimper]` | `<whimper>` |
| `[pant]` | `<pant>` |
| `[groan]` | `<groan>` |
| `[grunt]` | `<grunt>` |
| `[moan]` | `<moan>` |
| `[sneeze]` | `<sneeze>` |
| `[yawn]` | `<yawn>` |
| `[hiss]` | `<hiss>` |
| `[clear-throat]` | `<throat-clearing>` |

事件参数、注释和未列出的独立事件会被拒绝；不猜测近义映射。`whisper`、`shout` 仍为风格指令，`pause` 等精确时长事件仍在本地生成静音，不输出原生 pause 标记。原生标记请求表演效果，不保证精确的独立声音。文本中的字面 ASCII `<`、`>` 会被保守拒绝，即使添加 DSL 转义也不能绕过此限制；需要保留字面含义时请用 Unicode 替代。指令与台词分开传递。前述保留但未实现的功能同样适用。

## Fish Audio

Fish 适配器支持 `s2.1-pro`、`s2.1-pro-free` 和 `s2-pro`，可使用文本、显式音色 reference ID、共用 voice/as/reset 作用域和本地停顿。数值语速是**参数**（0.5–2 倍）；`fast`/`slow` 分别使用 1.2/0.8 倍。音量是 **dB 参数**，采用本地 −60…+20 范围。`act` 是**自然语言提示**，接受引号中的指令或 bare note，支持 scene/turn/span/reset 作用域。音高、显式语言/口音、style/role 指令、数值情绪强度和 laugh/cry/sigh span **不支持**。

原生 S2 标记映射为 `happy→[happy]`、`sad→[sad]`、`angry→[angry]`、`fear→[scared]`、`surprise→[surprised]`、`disgust→[disgusted]`、`calm→[calm]`、`excited→[excited]`、`whisper→[whispering]`、`shout→[shouting]`、`emph→[emphasis]`；其他专用情绪标签不支持；描述性指令可使用 `act`。事件映射为 `laugh→[laughing]`、`chuckle→[chuckling]`、`sigh→[sighing]`、`groan→[groaning]`、`pant→[panting]`、`gasp→[gasping]`、`yawn→[yawning]`、`snore→[snoring]`、`clear-throat→[clear throat]`。这些固定标记/事件的参数和注释不支持。这些是请求的表演效果，不保证精确独立声音或声学数值。

台词和表演指令中的字面 ASCII `[`/`]` 及原生 token 前缀 `<|` 会被拒绝，以防意外控制。表演指令不能为空，不能混用位置参数与 note，也不能使用命名参数。普通圆括号仍作为文本，不生成旧 S1 圆括号标记。前述保留但未实现的功能同样适用。
