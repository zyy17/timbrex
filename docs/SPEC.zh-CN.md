> 中文版。英文主文档：[SPEC.md](SPEC.md)。

# Timbrex 规范 v0.3.4（最终设计，语法版本 `timbrex: 1`）

> 一种括号、一条规则、一个开放注册表。**DSL 本身全英文 ASCII；台词内容可以是任何语言。**
> 一份 `.tbx`（亦接受 `.timbrex`）→ 拟人化语音 / 多角色对话 / 批量训练数据，编译到任意 TTS provider。
> 设计依据见 [REFLECTION.zh-CN.md](REFLECTION.zh-CN.md)；能力依据见 [CAPABILITIES.zh-CN.md](CAPABILITIES.zh-CN.md)、[MARKET_SURVEY.zh-CN.md](MARKET_SURVEY.zh-CN.md)。英文主文档 [SPEC.md](SPEC.md) 与本文件同为 v0.3.4。

## 变更记录

### v0.3.4（2026-10-21）
- **项目更名为 Timbrex**：取自 timbre（音色），造词后缀 -x；短、好读，包注册表与 pkg.go.dev 上轮核查为空（详见命名调研）。
- 命名方案（最终）：
  - CLI / Go module / 配置 / 锁 / 语法版本键 / JSON 字段 / 适配器前缀 / Go API：`timbrex`（例：`timbrex synth`、`timbrex.yaml`、`timbrex.lock`、`timbrex: 1`、`timbrex-provider-<name>`、`timbrex.Parse`）
  - 脚本扩展名：**`.tbx`**（首选）；**`.timbrex`** 为等价别名（实现应两者都认，`timbrex fmt` 默认写出 `.tbx`）
  - 批处理数据集：**`*.tbxset.yaml`**
  - 推荐仓库与 module：`github.com/<user>/timbrex`；安装 `go install github.com/<user>/timbrex/cmd/timbrex@latest`
- 由 Kouji（v0.3.3）改名：`kouji`→`timbrex`，`.kouji`→`.tbx`，`*.koujiset.yaml`→`*.tbxset.yaml`，`kouji: 1`→`timbrex: 1`。语法与语义不变；`timbrex migrate` 批量改文件名与版本键。诊断码前缀（V/W/I）与 `voices.lock.yaml` 不变。
- 英文主文档是 `SPEC.md`、`CAPABILITIES.md`、`MARKET_SURVEY.md`、`REFLECTION.md`，与对应 `.zh-CN.md` 说同一件事。更早的 v0.1 / v0.2 草案不在本仓库。

### v0.3.3（2026-10-21）
- **项目更名为 Kouji（口技）**：取自中国传统口技艺术——一人模仿万物之声，正是本项目"一份脚本驱动任意声音"的定位。
- 改名对照：命令行 `vox` → `kouji`；脚本扩展名 `.vox` → `.kouji`；数据集文件 `*.voxset.yaml` → `*.koujiset.yaml`；配置 `vox.yaml` → `kouji.yaml`；锁文件 `vox.lock` → `kouji.lock`（`voices.lock.yaml` 不变）；语法版本键 `vox: 1` → `kouji: 1`；`items.jsonl` 字段 `vox` → `kouji`；外部适配器可执行文件 `vox-provider-<name>` → `kouji-provider-<name>`；Go API `vox.Parse/Builder` → `kouji.Parse/Builder`。
- 推荐仓库 `github.com/<user>/kouji`，Go module 同名，CLI 安装为 `go install github.com/<user>/kouji/cmd/kouji@latest`。
- 诊断码前缀（V/W/I）不变；语法与语义无任何变化，`kouji migrate` 可把旧名文件批量改名并改写版本键。

### v0.3.2（2026-10-20）
- **tbxset（批量模式）默认开启严格标签**：新增 tbxset 字段 `strict_tags: true`（默认值）。未注册的 `[…]` 报错 **V041**，并附近似标签建议；它不会再静默变成自由表演。
- **显式自由表演写法统一为 `[act "…"]`**（区间写 `[act "…"|内容]`）。`act` 本身就是已注册的标准标签，带一个字符串参数，所以完全符合 5 条核心规则，不新增任何语法。所有模式下都合法，是严格模式下唯一的自由表演写法。
- 单脚本模式行为不变：未注册的 `[…]` 是隐式自由表演，加近似拼写警告（W040）。可在配置中设 `strict_tags: true` 主动开启严格模式；CLI 提供 `--strict-tags` / `--no-strict-tags`。
- `timbrex fix --explicit-acts`：把隐式自由表演 `[x]` 改写为 `[act "x"]`。`timbrex fmt` 会把 `[act holding back tears]` 规范为 `[act "holding back tears"]`。

### v0.3.1（2026-10-20）
- **DSL 仅英文**：标签名、命名空间、参数名、关键字、YAML 键、情绪集、标准库词表全部是英文 ASCII。删除 `zh` 别名包。
- **删除全角等价**：语法不再识别 `【】` `｜` `：` `“”`。它们在任何位置都是普通文本。改为由 `timbrex fix` 自动转换为规范英文写法，`timbrex check` 给出 lint 警告（§3.5）。
- **说话人 ID 必须是 ASCII 标识符**，并且在 cast 中声明；中文等显示名放到 cast 的 `name:`（§3.4）。说话人前缀只认 ASCII `:`。
- **自由表演（未注册的 `[…]`）允许任何语言**，推荐英文；新增 provider 语言偏好表和 `act_translate` 配置（§3.3）。
- **字符串值**（引号内）可以是任意 Unicode，比如文件路径、音效描述、音色设计提示。
- 重新评估了区间写法：**保留 `[x|content]` 作为唯一规范形式**，BBCode `[x]…[/x]` 由 `timbrex fix` 改写（§2.4）。
- 全部示例改为英文标签；EBNF、迁移表、工具链、待决问题同步更新。

### v0.3.0（2026-10-20）
- 首版最终设计：单一标签语法、位置决定作用域、开放 IR、五级能力触达、tbxset 数据集、三轴版本化、未来扩展演示。

---

## 0. 设计原则与语言边界

| 原则 | 落地 |
|---|---|
| **P1 极简易用** | 纯文本即合法；作者只需学 1 种标记 `[ ]`；作用域只由位置决定；与 ElevenLabs v3 写法几乎同形，LLM 天然会写；词表单一语言（英文），没有别名歧义 |
| **P2 高度可扩展** | **语法冻结**；所有增长发生在词表包（注册表）、命名空间、能力声明（caps）、适配器。IR 开放（feature id 是字符串），新能力不改核心 |
| **P3 覆盖全部能力** | 五级触达：标准标签 → 包标签 → 命名空间标签 → 命名空间原文/参数 → 原文围栏（§8 覆盖表） |
| **P4 可复现、可审阅** | 解析严格确定；宽容修复只在显式的 `timbrex fix`；词表与 caps 版本锁定 |

**语言规则**：凡是"代码"（标记、标识符、配置键、枚举值）都用英文 ASCII；凡是"内容"（台词、旁白、自由表演描述、引号字符串、显示名）用任意语言。

三层分离：

| 层 | 文件 | 内容 |
|---|---|---|
| 表演文本 | `*.tbx` 正文 | 谁说、说什么、怎么说、何时发生。**无控制流** |
| 配置 | `.tbx` 的 YAML front-matter，或 `timbrex.yaml` / `cast.yaml` | 说话人与音色（预置/克隆/设计/改造/混合）、provider、合规、输出、请求级透传参数 |
| 数据生成 | `*.tbxset.yaml`（声明式），或任意程序产出的 `items.jsonl` | 数据源、组合与采样、seed、切分、模板（只有 `{{var}}` 替换） |

---

## 1. 核心规则（这就是全部语法）

```
1. Plain text is a valid script. Markup is one form only:
     [name args…]  or  [name args…|content]        (ASCII [ ] | only)
2. Position decides scope:
     [x] inside a line     → event happens here / state lasts to end of this turn
     [x|content]           → applies to content only
     [x] alone on a line   → whole scene from here, until overridden or [reset]
3. name = registered ASCII tag, or ns:tag. Args: positional or key=value;
   values: 0.7  500ms  1.5s  120%  +2st  -3dB  word  "any unicode string".
   Unknown trailing words = note.
   Free-form acting: explicit [act "any language"] (always valid);
   implicit [anything unregistered] is allowed in single scripts only (tbxset = error V041).
4. Turns: `speaker_id: text` (speaker_id is an ASCII id declared in cast).
   Unprefixed paragraphs = narrator. A blank line ends a turn.
5. `//` at line start = comment. ``` fence = raw passthrough (e.g. ```ssml azure).
   Literal brackets: [[ ]] or \[ \] \|. Other brackets in text — () <> 【】 （） — are plain
   text, escaped per provider by the compiler.
```

最小示例：
```
---
cast:
  lin:  { name: 小林 }
  chen: { name: 老陈 }
---
lin: [laugh] 真的吗？[pause 500ms] 太好了！
chen: [sigh] [whisper|我早就知道了。]
```

---

## 2. 词法与语法（EBNF）

### 2.1 预处理
解析前只做两件事：统一换行为 `\n`；去除 BOM。**没有全角映射**：`【】` `｜` `：` `“”` 都按普通字符处理（转换交给 `timbrex fix`，见 §3.5）。

### 2.2 EBNF
```ebnf
script        = [ front_matter ] , { block } ;
front_matter  = "---" , NL , yaml_text , "---" , NL ;          (* 只能出现在文件开头 *)

block         = blank_line | comment_line | fence | scene_line | turn | narration ;
blank_line    = { WS } , NL ;
comment_line  = { WS } , "//" , { ANY } , NL ;
fence         = "```" , fence_info , NL , { raw_line } , "```" , NL ;
fence_info    = ident , { WS , ( ident | kv ) } ;             (* ssml azure / abc id=s1 / json minimax *)

scene_line    = { WS } , tag , { { WS } , tag } , { WS } , NL ; (* 一行只有标签 → 场景级 *)
turn          = speaker_id , ":" , content , NL , { cont_line } ;
narration     = content , NL , { cont_line } ;
cont_line     = content , NL ;                                 (* 非空、无说话人前缀、非注释的后续行 *)

speaker_id    = ident ;                                        (* 必须在 cast 中声明，见 §3.4 *)

content       = { text_run | tag | escape } ;
text_run      = TEXT_CHAR , { TEXT_CHAR } ;                    (* 除 "[" 与 "\" 外的任意 Unicode *)
escape        = "[[" | "]]" | "\" , ( "[" | "]" | "|" | "\" | ":" ) ;

tag           = "[" , { WS } , tag_body , { WS } , [ "|" , content ] , "]" ;
tag_body      = structured | free_act ;
structured    = head , { WS , arg } ;                          (* head 已注册时才成立，见 §2.3 *)
head          = [ ns , ":" ] , ident ;
ns            = ident ;
arg           = ident , "=" , value | value ;
value         = number | duration | percent | semitone | decibel | quoted | word ;
number        = [ "+" | "-" ] , DIGIT , { DIGIT } , [ "." , DIGIT , { DIGIT } ] ;
duration      = number , ( "ms" | "s" | "min" ) ;
percent       = number , "%" ;
semitone      = number , "st" ;
decibel       = number , "dB" ;
quoted        = '"' , { UNICODE - '"' | '\"' } , '"' ;        (* 内容可为任意语言 *)
word          = WORD_CHAR , { WORD_CHAR } ;                    (* 非空白、非 [ ] | = " *)
free_act      = ACT_CHAR , { ACT_CHAR } ;                      (* 任意 Unicode，除未转义的 [ ] | *)

ident         = ASCII_LETTER , { ASCII_LETTER | DIGIT | "_" | "-" | "." } ;
```

### 2.3 解析一个 tag 的确定性步骤
1. 取 `[` 后、第一个未转义的 `|` 或 `]` 之前的文本，称为 *body*。
2. 若 body 的第一个词形如 `ns:ident`，且 `ns` 是已注册命名空间 → 命名空间标签：
   - 适配器注册了 `ident` → 按其 TagSpec 解析参数（第 3 级）；
   - 否则 → **命名空间原文**：`:` 之后的全部文本原样交给该 provider（第 4 级）。
3. 否则，若第一个词是 ASCII ident 且在**已加载词表**中注册（大小写不敏感，`-`/`_` 等价）→ 按 TagSpec 匹配参数：先 `key=value`，再按类型填位置值；剩余词拼成 `note`（标签声明 `note: allow` 时才允许，否则报错）。
4. 否则 → **隐式自由表演** `act`，`note = body 原文`（任何语言）。严格模式下（tbxset 默认，或 `strict_tags: true`）这一步改为报错 V041，并附编辑距离 ≤2 的已注册名建议，以及"如需自由表演请写 `[act "…"]`"。命名空间原文（第 2 步）和 `[act "…"]` 不受严格模式影响。
5. 有 `|content` 时，内容递归解析（可嵌套），必须在**同一段落内**闭合。

### 2.4 区间写法的再评估：`[x|content]` 还是 `[x]…[/x]`
中文输入法下 `/` 变 `、` 的问题依然存在：作者在中文输入状态下写台词，切出来打标签时仍可能打出 `、`。但这个问题现在交给 `timbrex fix` 处理，已经不是决定性因素。去掉这一条之后，重新比较：

| 维度 | `[x|content]` | `[x]…[/x]`（BBCode） |
|---|---|---|
| 与"状态标签"区分 | 一眼可分：有 `|` 就是区间 | **开标签与状态标签完全同形**。`[whisper]` 是到轮末的状态，还是等一个远处的 `[/whisper]`？要向后看才能确定；漏写闭合会静默变成另一种语义 |
| 嵌套/交叉 | 不可能交叉（括号天然配对） | 可能写出 `[a][b][/a][/b]`，需要额外报错规则 |
| 闭合名匹配错误 | 不存在 | 常见（`[/whisp]`、`[/]` 歧义） |
| 局部性（LLM 流式生成、按段提交） | 段内自闭合 | 需要跨距离配对 |
| 长区间（多句） | 略笨重 | 自然 |
| LLM 熟悉度 | 高（形似 Markdown 链接、wiki 管道） | 更高 |

**结论：保持 `[x|content]` 为唯一规范形式。** 核心理由不是输入法，而是 **BBCode 的开标签和本语言的状态标签同形**，会让"位置决定作用域"这条核心规则出现例外。长区间用"状态标签 + 位置作用域"表达（到轮末，或独占一行作用于整个场景）。`timbrex fix` 会把同段落内配对的 `[x]…[/x]` 改写为 `[x|…]`；`timbrex check` 遇到 `[/x]` 时报错 V013 并给出修复建议。

---

## 3. 语义

### 3.1 文档结构
- **段落** = 被空行分隔的连续行。
- **轮（turn）** = 以 `speaker_id:` 开头的段落；后续无前缀行属于同一轮。
- **旁白段** = 不以说话人开头的段落，说话人为 `narrator`（配置 `narrator: host` 可改指向）。只有一个音色的纯文本脚本全部是旁白，零配置可用。
- **场景行** = 只含标签的行：不发声，只修改场景状态，或在时间线上放置事件（停顿、音效、BGM 等）。

### 3.2 标签种类（由注册表声明，作者无需记忆）
| kind | 在台词中间 `[x]` | 带内容 `[x|…]` | 独占一行 `[x]` |
|---|---|---|---|
| `event`（laugh、pause、sfx） | 在此处发生 | TagSpec 定义的"伴随"语义（`[laugh|while laughing]`）；未定义则报错 | 在两轮之间的时间线上发生 |
| `state`（emotion、speed、whisper） | 从此处到本轮结束 | 只作用于内容 | 到场景结束 / 被覆盖 / `[reset]` |
| `wrap`（pinyin、say、sub、lang） | 报错（需要内容） | 作用于内容 | 报错 |
| `turn`（overlap、gap） | 只能出现在轮首 | — | 作用于下一轮 |

**叠加规则**：同一 feature 内层覆盖外层；不同 feature 叠加；`[reset]` 清除轮内状态，`[reset speed]` 只清该 feature；独占一行的 `[reset]` 清除场景状态。

### 3.3 自由表演（act）与语言
- **显式写法（规范形式，所有模式通用）**：`[act "holding back tears"]`、`[act "强忍着眼泪"|我没事。]`。`act` 是 std 中的 state 标签，唯一参数是字符串；未加引号的词也会被收为 note（`[act holding back tears]`），`timbrex fmt` 会补上引号。
- **隐式写法（仅单脚本模式）**：未注册的方括号内容 = 自由表演，**允许任何语言**：`[strong French accent]`、`[强忍着眼泪|我没事。]`。与已注册名相近时给出 W040。
- **严格模式**（tbxset 默认，单脚本可用 `strict_tags: true` 或 `--strict-tags` 开启）：隐式写法报错 V041，必须改用 `[act "…"]`。理由：批量生成中，一个拼错的 `[laff]` 会悄悄污染成千上万条样本的标签和音频；显式写法让数据集中每一处自由表演都可以 grep、可审计，manifest 的 `acts` 字段也更干净。
- **推荐英文**，原因：ElevenLabs、Inworld 等官方文档偏好英文；英文也最不容易与已注册名混淆。非 ASCII 的自由表演会得到 `info[I050]`，不报错。
- provider 语言偏好（写在 caps 的 `act.lang` 中，以下为当前判断，需随实测更新）：

| provider | 自由文本指令通道 | 偏好语言 |
|---|---|---|
| ElevenLabs v3/v4 | 行内 `[…]` | 英文 |
| Inworld TTS-2 | 行内 `[…]` steering | 英文（文档明确） |
| Fish S2 / Drama 3 | 行内 `[…]` | 任意（文档称支持中文） |
| Gemini TTS | `speech_metadata.style` | 任意，英文更稳 ❓ |
| OpenAI mini-tts | `instructions` | 任意 |
| Hume | `description` | 英文 ❓ |
| Qwen-Audio / CosyVoice | `instruction` | 中文/英文；CosyVoice 部分音色有格式模板 |
| 豆包 2.0 | `context_texts` | 中文 |
| StepAudio | `instruction` + 行内 `(…)` | 中/英 |
| IndexTTS2 | `emo_text` | 中/英 |

- 配置 `act_translate: off | lexicon | llm`（默认 `lexicon`）：自由表演的语言与目标 provider 偏好不一致时，`lexicon` 查内置短语表翻译，查不到就原样发送并给 info；`llm` 调用配置的翻译模型，结果写入 `timbrex.lock` 缓存以保证可复现；`off` 一律原样发送。
- 降级：provider 支持自由文本 → 原样（或翻译后）发送；不支持 → 映射到最近的标准情绪/非语言声 → instruction 汇总 → drop 并警告。
- 严格开关的优先级：CLI `--strict-tags/--no-strict-tags` > tbxset `strict_tags` / 配置 `strict_tags` > 模式默认值（tbxset=true，单脚本=false）。

### 3.4 说话人
- 前缀 = `speaker_id:`，其中 `speaker_id` 是 ASCII 标识符，冒号是 ASCII `:`，且 `speaker_id` 在 cast 中声明。不满足就是普通文本。所以 `他说：你好` 和 `Note: this` 都不会被误认为说话人。
- 显示名放在 cast：`lin: { name: 小林 }`。显示名用于 manifest、报告、字幕，以及需要称呼说话人的指令类 provider。
- 配置 `cast: auto` 时，任何行首 `ident:` 都视为说话人，`timbrex check` 会列出识别结果。
- 字面冒号：`lin\: not a speaker`。

### 3.5 中文输入与 `timbrex fix`
语法只认 ASCII。中文作者用输入法打出的全角符号或中文词，由工具层处理，**不进入语法**：

| 作者写了 | 语法解释 | `timbrex check` | `timbrex fix` 改写 |
|---|---|---|---|
| `【笑】` | 普通文本 | `warning[W101] full-width bracket looks like a tag` | `[laugh]` |
| `【慢｜谁都没有说话】` | 普通文本 | W101 | `[slow|谁都没有说话]` |
| `[笑]` | 自由表演（未注册的中文） | `warning[W102] "笑" matches std tag laugh` | `[laugh]` |
| `[开心 0.7]` | 自由表演 | W102 | `[happy 0.7]` |
| `[停 500ms]` | 自由表演 | W102 | `[pause 500ms]` |
| `[强忍着眼泪]` | 自由表演 | `info[I050] non-English acting; target elevenlabs prefers English` | 不改（`--translate-acts` 时才翻译） |
| `小林：台词`（cast 有 `name: 小林`） | 普通文本（旁白） | `warning[W103] looks like speaker lin (display name 小林)` | `lin: 台词` |
| `[speed 1.2｜文本]`、`[act “…”]` | 全角 `｜` `“”` 被当作 body 的一部分 → 自由表演 | W104 | 转换为 `|`、`"` |
| `（笑）`、`(laughs)` | 普通文本 | `info[I060] looks like a stage direction` | 仅 `--parens` 时改为 `[laugh]` |
| `【重要通知】` | 普通文本 | 无（不匹配任何标签） | 不改 |
| `[whisper]…[/whisper]` | `[/whisper]` 是自由表演文本 → 报错 V013 | V013 | `[whisper|…]` |
| tbxset 中的 `[thoughtfully]` | 严格模式：错误 | `error[V041]` | `--explicit-acts` → `[act "thoughtfully"]` |

- 中文→英文的对照表（`笑→laugh`、`开心→happy`、`停/停顿→pause`、`慢→slow`、`小声→whisper`、`重读→emph`、`拼音→pinyin`……）是**工具数据**（`fix/zh.yaml`），不是词表的一部分；LSP 也用它支持"输入中文搜索英文标签"的补全。
- `timbrex fix --check` 用于 CI：存在可修复项时退出码非 0。

### 3.6 参数约定
- 情绪强度 0–1（默认 0.6）：`[happy 0.8]`。
- 语速：倍率或词 `[speed 1.2]` `[fast]`；音高：`+2st` 或 `high/low`；音量：`+3dB` 或 `loud/soft`。
- 程度词作为 note：`slightly`、`very`，适配器可映射到数值。

---

## 4. 标准词表（`std` 包，默认加载，全英文）

`timbrex tags --search laugh` 查询；LSP 支持中文关键词搜索（基于 `fix/zh.yaml`）。

### 4.1 核心
| tag | kind | 例 |
|---|---|---|
| `pause` | event | `[pause 500ms]` `[pause short]` `[pause]`（模型自然停顿） |
| 情绪 24 个：`neutral happy sad angry fear surprise disgust calm excited tender serious curious sarcastic nervous tired bored contempt confused proud embarrassed hopeful disappointed panicked crying` | state | `[sad 0.5]` `[happy|太好了]` |
| `emotion` | state | `[emotion wistful 0.4]`（标准集外的情绪名，走映射或自由文本） |
| `act` | state | `[act "holding back tears"]` |
| `speed` `fast` `slow` | state | `[slow|谁都没有说话]` |
| `pitch` `volume` | state | `[pitch +2st]` `[volume -3dB|……]` |
| `whisper` `shout` | state | `[whisper|就是想来看看你]` |
| `emph` | state | `[emph|进来！]` |
| `laugh` `sigh` `breath` `cough` `gasp` | event | `[laugh]` `[laugh nervously]` |
| `pinyin` `ipa` `sub` `say` | wrap | `[pinyin chong2|重]庆` `[sub "世界卫生组织"|WHO]` `[say digits|2026]` |
| `sfx` | event | `[sfx door_creak.wav]` `[sfx "distant thunder" dur=3s]` |
| `reset` | state | `[reset]` |

### 4.2 扩展（std 内）
- **韵律**：`silence leading|trailing|sentence 300ms`、`duration 20s`、`break`、`rate`。
- **风格**：`style newscaster|customer-service|chat|storytelling|poetry|asmr…`、`role girl|boy|elder…`、`accent French strong`、`oral high|mid|low`、`dialect sichuanese|cantonese|shanghainese…`（值为英文 ID，如需可用 `dialect "四川话"` 字符串）、`emovec happy=0.6 sad=0.2`。
- **非语言全集**：breath inhale exhale quick-breath pant gasp sigh huff groan grunt moan / laugh chuckle giggle cackle snort scoff / cry sob sniffle sniff whimper wail / cough clear-throat sneeze hiccup yawn burp snore swallow gulp lip-smack tongue-click kiss shush whistle hum hiss / hmm uh um emm。参数 `2s`、`intensity=`、note。带内容 = 伴随：`[laugh|边笑边说]`。
- **发音/文本**：`arpabet jyutping romaji spell verbatim latex lang mark`。`[lang en-US|open source]`、`[mark climax]`。
- **多人与时间线**：`overlap -300ms`、`gap 1s`、`cut`（在此处被打断）、`as <speaker_id>|…`（行内换人）、`voice id="…"|…`、`voice design="raspy old man"|…`。
- **音频**：`sfx`、`audio file`、`bgm file vol= fade= loop duck`、`bgm stop fade=3s`、`ambience "rain"`。
- **声音转换**：`vc in.wav to=lin`。

### 4.3 词表治理
1. **名字语义稳定**：新名字在"无原生支持 → 自由文本"降级下的效果必须与其英文字面意思一致；否则只能放进命名空间。
2. 只增不改义；改名 = 新名 + 旧名 `deprecated`（旧名作为别名保留，别名同样必须是英文）。
3. 新标签有 `since:`；锁定在旧版本的脚本中，新名字仍按自由表演解释，并提示可升级。

---

## 5. 配置（YAML，front-matter 或 `timbrex.yaml`；键全英文）

```yaml
---
timbrex: 1
use: [std@1, film@0.3]
lang: zh-CN                # 内容语言（影响文本归一化、指令模板语言）
provider: azure
fallback: [qwen, minimax]
dialogue: auto             # native | stitch | auto
narrator: narrator
act_translate: lexicon     # off | lexicon | llm
strict_tags: false         # 单脚本默认 false；tbxset 默认 true
parens: text               # text | act（仅影响 timbrex fix --parens 的默认）
seed: 42
cast:
  narrator: { name: 旁白, desc: "沉稳男声，中年", voices: { azure: zh-CN-YunjianNeural } }
  lin:
    name: 小林                          # 显示名，任意语言
    desc: "二十多岁女生，活泼"            # 内容字段，任意语言；指令类 provider 使用
    gender: f
    age: 25
    voices: { azure: zh-CN-XiaoxiaoNeural, fish: { reference_id: "<id>" } }
    clone:  { ref: voices/lin.wav, ref_text: "…", consent: consents/lin.pdf, cache: true }
    design: { prompt: "温柔的南方女孩，略带鼻音", seed: 7 }
    remix:  { base: clone, prompt: "more mature", strength: medium }
    mix:    [ { voice: "<a>", weight: 0.7 }, { voice: "<b>", weight: 0.3 } ]
    match:  { gender: f, age: 20-30, style: lively }
lexicon: { 重庆: chong2 qing4, GPT: "G P T" }
params:
  el: { stability: 0.3 }
  mm: { timbre_weights: [ { voice_id: a, weight: 70 } ] }
  inw: { delivery_mode: CREATIVE }
compliance: { require_consent_for_clone: true, watermark: provider, disclose_ai: true }
output: { format: wav, sample_rate: 24000, mode: merge }
---
```
音色解析优先级：`voices.<provider>` → `clone` → `design` → `remix` → `mix` → `match` → 默认音色 + 警告。`timbrex voices clone|design|remix` 预建音色并写入 `voices.lock.yaml`。克隆缺 consent 直接报错；各家的验证流程由 `timbrex check` 提示人工完成，不代为绕过。

---

## 6. 数据集生成：`*.tbxset.yaml`

```yaml
timbrex: 1
seed: 20261020
provider: minimax
fallback: [qwen]
config: cast.yaml
strict_tags: true                  # 默认即 true；未注册标签 → V041
inputs:
  row: csv(sentences.csv)
grid: [row, k]                     # 笛卡尔积维度
vars:                              # 每个 item 独立采样
  k:   range(3)
  spk: pick(cast)                  # pick(cast where gender=f)
  emo: weighted({neutral: 2, happy: 1, sad: 1, angry: 1, surprise: 1})
  spd: uniform(0.9, 1.15, step=0.05)
  nv:  pick(["", "[laugh] ", "[sigh] ", "[breath] "])
  act: pick(["", "[act \"holding back a smile\"] ", "[act \"slightly out of breath\"] "])
id: "{{row.id}}_{{k}}"
split: { train: 0.9, dev: 0.05, test: 0.05, by: spk }
template: "{{spk}}: {{{nv}}}{{{act}}}[speed {{spd}}][{{emo}} 0.7]{{row.text}}"
output: { mode: per-item, manifest: manifest.jsonl, export: [hf] }
qc: { asr_cer_max: 0.1, max_silence: 1.5s }
```
- 严格标签检查发生在 `timbrex expand` 之后、合成之前，**覆盖每个展开后的 item**（包括 `{{{x}}}` 插入的片段和 `items.jsonl` 中的 `timbrex` 字段）。报告按"模板位置 + 变量取值"聚合，同一个错误只报一次，并列出受影响的 item 数。
- `{{x}}` **默认转义**（数据里的 `[` `]` `|` 不会变成标签），`{{{x}}}` 原样插入；模板中没有条件和循环。
- 每个变量的 RNG = hash(seed, item id, 变量名)：增删一个变量不影响其他变量的取值。
- `timbrex expand` 输出 `items.jsonl` 供审阅和复现；复杂逻辑直接用任意语言生成 `items.jsonl`（每行 `{id, timbrex, meta}`）。
- manifest 字段同 v0.2 §13，另加 `vars`、`vocab`（词表锁版本）、`caps_verified`、`speaker_name`。

---

## 7. 扩展架构

```
.tbx → parse（冻结语法）→ AST（通用 Tag 节点）
     → resolve（注册表 / 命名空间 / 版本锁）→ lower → 开放 IR
     → plan（caps + 降级链）→ requests + postops → synth → render → 音频 + manifest
```

### 7.1 注册表：TagSpec（YAML 零代码，或 Go）
```yaml
package: film
version: 0.3.0
namespace: film                 # [film:door-slam]；use 后也可直接写 [door-slam]
tags:
  - name: door-slam
    kind: event
    feature: audio.sfx
    lower: { sfx: assets/door_slam.wav }
    describe: "a door slamming shut"        # 进入 LLM 能力卡片
  - name: hold-back-tears
    kind: state
    expand: '[act "holding back tears, voice tight"]'
    per_provider: { el: "[holding back tears]" }
  - name: breathy
    since: 0.3.0
    kind: state
    feature: voice.breathiness
    params: [ { name: level, type: number, range: [0, 1], default: 0.5 } ]
    note: allow
    fallback: [ native, { instruction: "more breathy, {level|degree}" }, drop ]
```
校验：`name`、`namespace`、`params[].name`、`aliases` 必须匹配 ASCII ident；`describe` 与 `expand` 中的引号内容不限语言。

```go
type TagSpec struct {
    Name, Namespace, Since, Deprecated, ReplacedBy string
    Aliases  []string                      // ASCII only
    Kind     Kind                          // Event | State | Wrap | Turn
    Feature  string                        // 开放 feature id
    Params   []ParamSpec
    Note     NotePolicy
    Lower    func(*LowerCtx, *ast.Tag) ([]ir.Node, error) // 可选
    Fallback []Strategy
    Describe string
}
func Register(pkg Package)
```

### 7.2 开放 IR
```go
type Node struct {
    Kind    NodeKind         // Text | Event | StateSet | StateUnset | SpanBegin | SpanEnd | TurnAttr | Raw
    Feature string           // "prosody.speed", "emotion", "nv.laugh", "act", "pass.el", ...
    Args    map[string]Value
    Note    string           // 自由描述（任意语言）
    NoteLang string          // 检测出的语言，供 act_translate 使用
    Text    string
    Src     Pos
}
type Utterance struct { Speaker string; Nodes []Node; Offset time.Duration; Src Pos }
```
核心只理解结构性 feature（timeline、speaker、text、raw），其余 feature 都由 caps 和适配器解释。

### 7.3 能力声明（caps）
```yaml
provider: minimax
models: [speech-2.8-hd, speech-2.8-turbo]
verified: 2026-10-20
source: https://platform.minimax.io/docs/api-reference/speech-t2a-http
features:
  pause:         { native: "<#{sec}#>", range: [0.01, 99.99], no_consecutive: true }
  emotion:       { param: voice_setting.emotion, values: [happy, sad, angry, fearful, disgusted, surprised, calm, fluent], map: { fear: fearful }, scope: request }
  nv.laugh:      { native: "(laughs)" }
  pron.pinyin:   { native: "({py}){text}" }
  prosody.speed: { param: voice_setting.speed, range: [0.5, 2], scope: request }
  act:           { strategy: [map-nearest, drop] }
text_escape: { "(": "（", ")": "）" }
---
provider: elevenlabs
features:
  act: { native: "[{note}]", lang: en }      # 偏好语言，驱动 act_translate
```
策略链：`native → param → ssml → instruction → postprocess → asset → map-nearest → drop`；`scope: request` 的特性自动切分请求；`--strict`、`--report` 同 v0.2。

### 7.4 能力触达五级（同一语法）
| 级 | 写法 | 可移植 |
|---|---|---|
| 1 标准 | `[sad 0.6]` `[laugh]` | ✅ |
| 2 包 | `[door-slam]` `[film:door-slam]` | ✅（由包定义降级） |
| 3 命名空间标签 | `[az:express-as style=cheerful degree=1.5|…]` `[cart:emotion triumphant]` | 仅该 provider |
| 4 命名空间原文 / 参数 | `[el:strong French accent]` → 原样输出 `[strong French accent]`；`[fish:(sobbing)]`；`[mm:set timbre_weights=…]`；或配置 `params:` | 仅该 provider |
| 5 原文围栏 | ```` ```ssml azure ```` | 仅该 provider |

非目标 provider 遇到第 3–5 级：`on_other: drop | warn | act | error`（默认 warn+drop）。

### 7.5 适配器
Go 接口（`ID/Aliases/Namespace/Caps/Voices/Tags/Plan/Synthesize`）；**声明式适配器**（只需 caps + `request:` 模板，零 Go 代码）；**外部进程适配器**（`timbrex-provider-<name>`，JSON-RPC over stdio，可用任意语言实现）。

---

## 8. 能力覆盖表

| Provider 能力 | v0.3.1 写法 | 级 |
|---|---|---|
| SSML break/prosody/emphasis/say-as/phoneme/sub/lang/voice/audio/mark | `[pause]` `[speed]/[pitch]/[volume]` `[emph|]` `[say|]` `[ipa|]/[pinyin|]` `[sub|]` `[lang|]` `id:` `[audio]` `[mark]` | 1 |
| Azure express-as style/styledegree/role | `[style cheerful]` + `[happy 0.8]` + `[role girl]`；精确值 `[az:express-as …|…]` | 1/3 |
| Azure HD `[whispering]`、silence、audioduration、backgroundaudio、viseme | `[whisper]`、`[silence sentence 300ms]`、`[duration 20s]`、场景行 `[bgm …]`、`--visemes` | 1 |
| Azure 任意 SSML | ```` ```ssml azure ```` | 5 |
| Google Chirp `[pause short/long]`、SSML 子集 | `[pause short]` `[pause long]` | 1 |
| Gemini style、`<sigh>` 类事件、多人、大写强调、voice design | emotion/`[act]` → style；非语言声；`dialogue: native`；`[emph|]` → 大写；cast `design` | 1 |
| Polly domain news / whispered / drc / Speech Marks | `[style newscaster]` / `[whisper]` / `[polly:set drc=true]` / manifest | 1/4 |
| ElevenLabs v3/v4 自由 audio tags、音效、口音、唱 | 原样同形：`[laughs harder]`；`[sfx "applause"]`；`[accent French strong]`；`[sing|]` | 1 |
| ElevenLabs Dialogue / voice_settings / 发音词典 / IVC/PVC/Design/Remix | `dialogue: native`；`[el:set stability=0.3]`；`lexicon`；cast | 1/4 |
| OpenAI instructions、speed | 样式汇总为 instructions；`[speed]` | 1 |
| Cartesia speed/volume/emotion/break/spell/`[laughter]`、58 种情绪 | `[speed]` `[volume]` emotion `[pause]` `[spell|]` `[laugh]`；长尾 `[cart:emotion triumphant]` | 1/3 |
| Hume description/speed/trailing_silence | `[act]` / `[speed]` / 轮末 `[pause]` | 1 |
| Fish S2/Drama3 自由 `[…]`、词级标签、`<|speaker:N|>`、语言标签、音素 | 自由表演原样（中文亦可）；`[emph|]` `[whisper|]`；`dialogue: native`；`[lang|]`；`[pinyin|]` `[arpabet|]` `[romaji|]` | 1 |
| Fish S1 `(emotion)` | 标准情绪；长尾 `[fish:(in a hurry tone)]` | 1/4 |
| MiniMax `<#x#>`、emotion、speed/vol/pitch、注音、语气词、timbre_weights、latex | `[pause]`、情绪、韵律、`[pinyin|]/[ipa|]/[jyutping|]`、非语言声、cast `mix` 或 `[mm:set …]`、`[latex|]` | 1/4 |
| 豆包 2.0 context_texts、复刻 | 样式汇总为中文指令；cast `clone` | 1 |
| 阿里 CosyVoice SSML/instruction、Qwen-Audio 行内标签、hot_fix、seed、AIGC 标识、voice design | SSML 同上；情绪/`[act]`/`[style asmr]`/`[speed]`；非语言声；`lexicon`；`seed`；`compliance`；cast `design` | 1 |
| 腾讯 EmotionCategory+Intensity（sajiao/aojiao/jieshuo） | 情绪；`[tc:emotion aojiao]` | 1/3 |
| 讯飞 oral_level | `[oral high]` | 1 |
| Inworld steering、约 50 个非语言词、`[reset]`、delivery_mode、break、verbatim | 自由表演（英文）；非语言全集；`[reset]`；`[inw:set delivery_mode=CREATIVE]`；`[pause]`；`[verbatim|]` | 1/4 |
| StepAudio instruction + 行内 `(…)`、pronunciation_map | 场景行 `[act …]` → instruction；轮内 `[act]` → `(…)`；`lexicon` | 1 |
| xAI Grok 标签、speed、replace、时间戳、克隆 | 情绪/非语言/`[whisper]`；`[speed]`；`lexicon`；manifest；cast `clone` | 1 |
| Mistral Voxtral（参考音频驱动） | cast `clone`（可加 `clone.by_emotion`） | 1 |
| ChatTTS / CosyVoice2/3 / IndexTTS2 / Dia / Kokoro | `[laugh]` `[pause]` `[oral]` / `[breath]` `[laugh|…]` `[emph|]` `[act]` / `[emovec]` `[duration]` / `id:` + native / `[ipa|]` `[emph|]`；精确 token `[chattts:oral_2]` | 1/4 |
| GPT-SoVITS、F5、Spark、Chatterbox、Higgs、VibeVoice | 外部进程适配器 + caps | — |
| 克隆/设计/Remix/混合/S2S/水印 | cast `clone/design/remix/mix`、`[vc]`、`compliance` | 1 |
| 流式/实时 | 运行时（§10.4） | — |

---

## 9. 版本化（三条独立轴）

| 轴 | 标识 | 规则 |
|---|---|---|
| 语法 | `timbrex: 1` | 目标永不变；EBNF 变化才升主版本，并配套 `timbrex migrate` |
| 词表包 | `use: [std@1]` → `timbrex.lock` | semver；只增不改义；改名 = 新名 + 旧名 deprecated |
| 能力声明 | caps `verified` / `deprecated` / `sunset` | 随适配器发布；`timbrex caps --probe` 实测 |

`timbrex.lock` 记录词表版本、caps 版本、音色锁、`act_translate=llm` 的翻译缓存。

---

## 10. 未来扩展演示（都不改语法、不改核心）

### 10.1 新模态：带乐谱的唱歌
`music` 包新增 `sing`（state，feature `music.sing`，参数 `key` `tempo` `notes` `score`），围栏 info 新增 `abc` / `musicxml`；Acme caps 声明 `music.sing: { param: singing.score }`；其他 provider 降级为 `instruction: "sing it"`，在 ElevenLabs 上是 `[sings]`。
````
lin: [sing key=C tempo=90 notes="C4/4 D4/4 E4/2"|小星星]
lin: [sing score=s1|一闪一闪亮晶晶]
```abc id=s1
X:1
K:C
CC GG|AA G2|
```
````

### 10.2 provider 新参数：当天可用 → 逐步标准化
- 第 0 天：`[mm:set breathiness=0.4]`、`[fish:breathy]`（第 4 级，无人改代码）。
- 第 7 天：minimax 适配器注册 `[mm:breathiness 0.4]`（第 3 级，带校验与补全）。
- 第 30 天：`std@1.3` 新增 `breathy`（§7.1），caps 中 minimax 映射为 param、fish 映射为 native，其他 provider 走 instruction。锁定在 `std@1.2` 的旧脚本里，`[breathy]` 仍是自由表演，行为不变。

### 10.3 生成式音效与空间音频
`sfx` 接受文件或描述；带 `audio.sfx.generate` 能力的音效适配器负责生成并缓存，否则查素材库，再不行就 drop。新增 `spatial` 包 `pos`（feature `render.spatial`），由 render 的 HRTF 插件实现；provider 将来原生支持时，只需在 caps 里声明 native。
```
[ambience "night rain" vol=-24dB]
chen: [pos az=-60 dist=3m] 有人吗？
[sfx "distant thunder rolling" dur=3s pos="az=30"]
```

### 10.4 实时
`timbrex serve --ws` 以段落为提交单位增量解析与合成；需要时可加 `realtime` 包事件 `[await user]`，离线编译时 drop 并警告。

---

## 11. 工具链与 LLM 写作

| 工具 | 作用 |
|---|---|
| `timbrex prompt [-p fish]` | 生成 LLM 能力卡片：§1 规则 + 当前英文词表（含 describe）+ provider 支持度与自由表演偏好语言 |
| `timbrex check` | 语法、未知标签（编辑距离建议）、说话人、降级预览、合规、§3.5 的 lint |
| `timbrex fix` | 全角括号与符号 → ASCII；中文标签词 → 英文规范名；`[x]…[/x]` → `[x|…]`；`<laugh>`、`*laughs*` → `[laugh]`；显示名前缀 → 说话人 ID；`--parens`、`--translate-acts`、`--explicit-acts`（隐式自由表演 → `[act "…"]`）；输出 diff，`--check` 用于 CI |
| `timbrex fmt` | 规范化空白、参数顺序、别名 → 规范名 |
| `timbrex expand/compile/synth/diff/caps/voices/tags/migrate` | 同 v0.2 |
| LSP / MCP / HTTP / Go 库 | 补全（支持中文关键词搜索英文标签）、悬停显示降级链；`timbrex.Parse`、`timbrex.Builder` |

错误信息示例：
```
scene.tbx:12:9  error[V012]   unclosed span: [slow|谁都没有说话
                              hint: close it with ] in the same paragraph (fixable)
scene.tbx:13:5  error[V013]   [/whisper] is not valid; spans use [whisper|…] (fixable)
scene.tbx:15:1  warning[W103] "小林：" looks like speaker lin (display name 小林); treated as narration (fixable)
scene.tbx:16:7  warning[W101] full-width 【笑】 is plain text; did you mean [laugh]? (fixable)
scene.tbx:18:14 warning[W040] unknown [laff] used as free-form acting; did you mean [laugh]?
emo.tbxset.yaml:template:21 error[V041] unregistered tag [laff] (strict_tags); did you mean [laugh]?
                              for free-form acting write [act "laff"]   (affects 3,000 items)
scene.tbx:19:3  info[I050]    non-English acting "强忍着眼泪"; elevenlabs prefers English (act_translate=lexicon: no match, sent as-is)
```

---

## 12. 示例：v0.2 → v0.3.1

### 12.1 有声书场景
v0.2：
```
@bgm "rain.wav" vol=-22dB loop
> 雨下了一整夜。[停 600ms] 门被推开的时候，<慢>谁都没有说话</慢>。
[sfx "door_creak.wav"]
lin(惊讶 0.6): 陈叔？[停 300ms] 你怎么……[呼吸] 这么晚了。
chen(难过 0.5): [叹气] 没事。<耳语>就是想来看看你。</耳语>
lin: <温柔 0.6>先进来吧，外面冷。</温柔>
@overlap -200ms
chen: 不用了，<interrupt>我马上就</interrupt>
lin(生气 0.7): <重读>进来！</重读>
@bgm stop fade=3s
```
v0.3.1：
```
[bgm rain.wav vol=-22dB loop]
雨下了一整夜。[pause 600ms] 门被推开的时候，[slow|谁都没有说话]。
[sfx door_creak.wav]
lin: [surprise 0.6] 陈叔？[pause 300ms] 你怎么……[breath] 这么晚了。
chen: [sad 0.5][sigh] 没事。[whisper|就是想来看看你。]
lin: [tender 0.6] 先进来吧，外面冷。
chen: 不用了，我马上就[cut]
lin: [overlap -200ms][angry 0.7][emph|进来！]
[bgm stop fade=3s]
```

### 12.2 播客 + 自由表演 + 透传
v0.2：
```
@dialogue native
host(兴奋): 欢迎收听科技夜聊！[笑] 我是老王。
guest: 大家好。[若有所思] 今天聊点什么？
host: [el:strong Sichuan accent] 今天聊开源语音合成。
guest: [轻笑] 好啊。
```
v0.3.1（`dialogue: native` 移到配置）：
```
host: [excited] 欢迎收听科技夜聊！[laugh] 我是老王。
guest: 大家好。[thoughtfully] 今天聊点什么？
host: [accent Sichuan strong] 今天聊开源语音合成。
guest: [chuckle] 好啊。
```
`[thoughtfully]` 是隐式自由表演（单脚本模式合法；推荐英文）。显式写法是 `[act "thoughtfully"]`，放进 tbxset 模板时必须用这种写法，否则报 V041。`[若有所思]` 在单脚本中也合法，但会得到 info I050。

### 12.3 批量情绪数据集
v0.2 在 `.tbx` 中写 13 行控制流 → v0.3.1 用 `emo.tbxset.yaml`（§6），核心一行：
```yaml
template: "{{spk}}: {{{nv}}}{{{act}}}[speed {{spd}}][{{emo}} 0.7]{{row.text}}"
```
其中 `act` 变量的取值都是 `[act "…"]` 形式；如果写成 `"[thinking hard] "`，`timbrex expand` 后会报 V041。

---

## 13. 迁移（`timbrex migrate --to 1`，包含 `timbrex fix` 的全部改写）
| v0.2 / v0.3.0 | v0.3.1 |
|---|---|
| `<x args>文本</x>`、`</>` | `[x args|文本]` |
| `角色(情绪 强度):` | `id: [emotion 0.x]` |
| `> 旁白` | 无前缀段落 |
| 中文标签 / 别名（`[笑]` `[停 500ms]` `<温柔>`） | 英文规范名（`[laugh]` `[pause 500ms]` `[tender|…]`） |
| `【】` `｜` `：` `“”`（v0.3.0 的语法等价） | ASCII `[]` `|` `:` `"` |
| 中文说话人前缀 `小林:` | cast 中 `lin: { name: 小林 }` + `lin:` |
| `@overlap/@gap` | 轮首 `[overlap …]` `[gap …]` |
| `@bgm` | 场景行 `[bgm …]` |
| `<interrupt>…</interrupt>` | 断点处 `[cut]` |
| `@dialogue` `@param` `@provider` | 配置 `dialogue:` `params:`；`[ns:set …]`；拆分文件或 `--provider` |
| `<raw azure>…</raw>` | ```` ```ssml azure ```` |
| `@for/@loop/@pick/@rand/@weighted/@if/@macro/@item`、`${x}` | `*.tbxset.yaml`、`{{x}}` |
| `#`、`<!-- -->` 注释 | `//` |
| `[py "…"]` | `[pinyin …|…]` |

## 14. 实现路线（Go）
| 阶段 | 内容 |
|---|---|
| M0 | 冻结语法 parser（含错误恢复）、注册表（ASCII 校验）、std、`check/fmt/fix(含 zh.yaml)/tags/prompt`、golden tests |
| M1 | 开放 IR、caps、plan 降级、render 基础；provider：azure、elevenlabs、minimax、gemini、qwen |
| M2 | tbxset（含 strict_tags/V041）、manifest、缓存/并发/fallback、voices.lock、act_translate |
| M3 | fish、cosyvoice、doubao、inworld、stepfun、cartesia、tencent、polly、xai；声明式适配器；外部进程适配器；本地模型 |
| M4 | 原生对话、LSP、MCP/HTTP、实时 `serve --ws`、`caps --probe` |

## 15. 待用户决定
1. **`act_translate` 的默认值**：`lexicon`（当前默认，可复现、不花钱）还是 `llm`（翻译质量高，需缓存锁定）？
2. ~~tbxset 是否默认严格~~ → 已决定（v0.3.2）：默认严格，显式写 `[act "…"]`。
3. **cast 是否强制声明**：当前强制，`cast: auto` 可选。
4. **front-matter 是否保留**：当前与 `timbrex.yaml` 两种方式并存。
5. 情绪集规模（24 个）；是否把 valence/arousal 作为 `emovec` 的标准轴。
6. 项目名。
7. CAPABILITIES 中标 ❓ 的项目，以及 §3.3 语言偏好表中标 ❓ 的项，需要实测。
