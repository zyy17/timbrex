> 中文版。英文主文档：[CAPABILITIES.md](CAPABILITIES.md)。

# TTS Provider 能力矩阵（v0.2，2026-10-20 查证）

图例：✅ 官方文档确认；⚠️ 有条件/部分确认；❓ 未查到官方说明，需实测；❌ 官方明确不支持。
"原生写法"列给出 provider 的真实语法，供适配器直接生成。

## 1. 商业 API

### Azure AI Speech（Neural / HD voices）
| 能力 | 状态 | 原生写法 |
|---|---|---|
| SSML | ✅ | `<speak>` 全套：`break` `prosody` `emphasis` `say-as` `phoneme`(ipa/sapi/ups/x-sampa) `sub` `lang` `voice` `audio` `bookmark` `lexicon` |
| 情绪/风格 | ✅ 仅部分音色 | `<mstts:express-as style="cheerful" styledegree="0.01–2" role="Girl">`，style/role 取值随音色而异 |
| HD 音色风格标记 | ✅ | 行内 `[whispering]` 等，持续到 `[Neutral]`/新句/换行 |
| 静音精控 | ✅ | `<mstts:silence type="Leading|Tailing|Sentenceboundary|comma-exact…" value="0–20000ms">` |
| 目标时长 | ✅ | `<mstts:audioduration value="20s">`（0.5–2× 自然时长） |
| 背景音 | ✅ | `<mstts:backgroundaudio src= volume= fadein= fadeout=>` |
| 多人 | ✅ | 单请求多个 `<voice>`；另有多人对话 HD 音色（❓细节待查） |
| 时间戳 | ✅ | WordBoundary / Viseme / Bookmark 事件 |
| 非语言声 | ❌ 无通用标签 | — |
| 声音定制 | ✅ | Custom Neural Voice / Personal Voice |
| 流式 | ✅ | SDK / WebSocket |
来源：https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice ；https://learn.microsoft.com/en-us/azure/ai-services/speech-service/high-definition-voices ；https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-structure

### Google Cloud TTS
| 能力 | 状态 | 原生写法 |
|---|---|---|
| SSML（Neural2/WaveNet/Standard/Studio） | ✅ | 标准 SSML |
| Chirp 3 HD SSML | ⚠️ Preview，仅同步请求 | `speak p s say-as phoneme sub break audio prosody voice`；流式不支持 SSML |
| Chirp 3 HD 停顿 | ✅ | `markup` 字段：`[pause short]` `[pause]` `[pause long]`（时长不固定） |
| Chirp 3 HD 语速/音高参数 | ❌（音色列表页说明） | — |
| Gemini TTS 风格 | ✅ | 自然语言 prompt；新版本用 `speech_metadata.style`（整轮）| 
| Gemini TTS 行内事件 | ✅ | 尖括号 `<sigh>` `<laughs>` `<short pause>` `<gasp>` 等（人声事件，非音效） |
| Gemini 多人 | ✅ | `speakers` 配置 + 每轮 metadata（多人上限以文档为准） |
| 强调 | ✅ | 大写单词 |
| 时间戳 | ⚠️ SSML `<mark>` 仅部分音色 | — |
来源：https://docs.cloud.google.com/text-to-speech/docs/chirp3-hd ；https://docs.cloud.google.com/text-to-speech/docs/list-voices-and-types ；https://ai.google.dev/gemini-api/docs/speech-generation ；https://docs.cloud.google.com/text-to-speech/docs/gemini-tts
注：Cartesia 汇总页称 Gemini 2.5/3.1 preview TTS 将于 2026-11-17 后下线（以 Google 公告为准）。

### AWS Polly
| 能力 | 状态 | 原生写法 |
|---|---|---|
| SSML | ✅（Generative 仅子集，不支持的标签报错） | `break emphasis lang mark p phoneme prosody s say-as sub w` |
| 情绪 | ❌ | — |
| 风格 | ⚠️ | `<amazon:domain name="news">` 仅部分 Neural 音色（Matthew/Joanna/Lupe/Amy） |
| 耳语 | ⚠️ 仅 Standard | `<amazon:effect name="whispered">` |
| 动态压缩 | ⚠️ 非 Generative | `<amazon:effect name="drc">` |
| 时间戳 | ✅ | Speech Marks（word/sentence/viseme/ssml） |
来源：https://docs.aws.amazon.com/polly/latest/dg/supportedtags.html ；https://docs.aws.amazon.com/polly/latest/dg/generative-voices.html ；https://docs.aws.amazon.com/polly/latest/dg/newscaster-voices.html

### ElevenLabs（v3 / v4）
| 能力 | 状态 | 原生写法 |
|---|---|---|
| Audio tags | ✅ 自由文本，非枚举 | `[laughs]` `[laughs harder]` `[whispers]` `[shouts]` `[sighs]` `[exhales]` `[clears throat]` `[sarcastic]` `[curious]` `[excited]` `[crying]` `[snorts]` `[giggling]` … |
| 音效 tags | ✅（实验） | `[applause]` `[gunshot]` `[explosion]` `[leaves rustling]` |
| 口音 | ✅（实验） | `[strong French accent]` |
| 唱 | ✅（实验） | `[sings]` |
| 停顿 | ⚠️ v3/v4 不支持 SSML `<break>`；用 `...` / 标点；v2 系列支持 `<break time="1.5s"/>` | — |
| 强调 | ✅ | 大写、标点 |
| 多人对话 | ✅ | Text to Dialogue API：`inputs:[{text, voice_id}]` |
| 速度/稳定度 | ✅ | `voice_settings`（stability: Creative/Natural/Robust 等） |
| 发音词典 | ✅ | pronunciation dictionaries（phoneme/alias） |
| 时间戳 | ✅ | `with-timestamps` 接口、强制对齐 |
| 声音设计/克隆 | ✅ | Voice Design / IVC / PVC |
来源：https://elevenlabs.io/docs/overview/capabilities/text-to-speech/best-practices ；https://elevenlabs.io/docs/overview/capabilities/text-to-dialogue.md ；https://elevenlabs.io/docs/help-center/product/core-capabilities/text-to-speech/how-do-audio-tags-work-with-eleven-v3-and-v4.md

### OpenAI gpt-4o-mini-tts
| 能力 | 状态 | 原生写法 |
|---|---|---|
| 风格/情绪/口音/语速/语气 | ✅ | `instructions` 自然语言 |
| 速度 | ✅ | `speed` 参数 |
| SSML / 行内标签 | ❌ | — |
| 多人 / 时间戳 | ❌ | 逐句合成 + STT 对齐 |
| 生命周期 | ⚠️ 已弃用，2027-01-06 关停，迁移 Realtime API | — |
来源：https://developers.openai.com/api/docs/guides/text-to-speech ；https://developers.openai.com/api/docs/deprecations

### Cartesia Sonic-3
| 能力 | 状态 | 原生写法 |
|---|---|---|
| 速度 | ✅ | `<speed ratio="0.6–1.5"/>` 或 `generation_config.speed` |
| 音量 | ✅ | `<volume ratio="0.5–2.0"/>` |
| 情绪（beta，仅英语） | ✅ | `<emotion value="angry"/>`；枚举 ~58 个，主情绪 neutral/calm/angry/content/sad/scared |
| 停顿 | ✅ | `<break time="1s"/>` |
| 拼读 | ✅ | `<spell>` |
| 笑 | ✅ | `[laughter]` |
| 发音词典 | ✅ | `pronunciation_dict_id` |
| 时间戳/流式 | ✅ | WebSocket word timestamps（❓细节） |
来源：https://docs.cartesia.ai/build-with-cartesia/capability-guides/ssml-tags ；https://docs.cartesia.ai/build-with-cartesia/capability-guides/volume-speed-emotion

### Hume Octave
| 能力 | 状态 | 原生写法 |
|---|---|---|
| 表演指令 | ✅ Octave 1；⚠️ Octave 2 文档称"coming soon" | 每 utterance `description`（≤100 字符推荐） |
| 语速 | ✅ | `speed` 0.5–2（非线性） |
| 句后静音 | ✅ | `trailing_silence`（秒） |
| 多段 | ✅ | `utterances[]`，每段可换 voice |
| 声音设计 | ✅ | 用描述生成音色 |
来源：https://dev.hume.ai/voice/docs/text-to-speech-tts/acting-instructions.md ；https://dev.hume.ai/voice/reference/text-to-speech-tts/synthesize-json.md

### Fish Audio
| 能力 | 状态 | 原生写法 |
|---|---|---|
| S2/S2.1-Pro 自由提示 | ✅ | `[whisper]` `[laugh]` `[emphasis]` `[sigh]` `[gasp]` `[pause]` `[inhale]` `[exhale]` `[angry]` … 任意自然语言，可放任意位置 |
| S1（2026-12-31 退役）固定标签 | ✅ | `(happy)` 等 24 基础 + 25 高级情绪；语气 `(whispering)(shouting)(screaming)(soft tone)(in a hurry tone)`；声音 `(laughing)(chuckling)(sobbing)(crying loudly)(sighing)(panting)(groaning)…`；`(break)(long-break)` |
| 克隆 | ✅ | reference_id / 参考音频 |
| 流式 | ✅ | HTTP stream / WebSocket |
来源：https://docs.fish.audio/developer-guide/models-pricing/models-overview ；https://docs.fish.audio/developer-guide/core-features/emotions

### MiniMax Speech（T2A v2）
| 能力 | 状态 | 原生写法 |
|---|---|---|
| 停顿 | ✅ | `<#0.5#>`（0.01–99.99s，不可连续） |
| 情绪 | ✅ | `voice_setting.emotion`: happy/sad/angry/fearful/disgusted/surprised/calm/fluent/whisper（fluent/whisper 仅 2.6；2.8 不支持 whisper） |
| 语速/音量/音高 | ✅ | speed 0.5–2；vol (0,10]；pitch −12..12 |
| 拼音/IPA/粤拼 | ✅ | `(he2)平`、`(lɪv)` |
| 语气词标签 | ✅ 仅 speech-2.8 | `(laughs)(chuckle)(coughs)(clear-throat)(groans)(breath)(pant)(inhale)(exhale)(gasps)(sniffs)(sighs)(snorts)(burps)(lip-smacking)(humming)(hissing)(emm)(sneezes)` |
| 混合音色 | ✅ | `timbre_weights` |
| 字幕/时间戳 | ✅ | subtitle 输出（❓粒度） |
| 克隆/声音设计 | ✅ | voice clone / design |
来源：https://platform.minimax.io/docs/api-reference/speech-t2a-http

### 火山引擎 / 豆包语音合成 2.0
| 能力 | 状态 | 原生写法 |
|---|---|---|
| 语音指令（情绪/语气） | ✅ 仅官方 2.0 音色；复刻音色不支持 | `req_params.context_texts: ["用悲伤、哽咽的语气说"]` |
| 情感枚举参数 | ❓ 1.0 部分多情感音色有 emotion 参数，2.0 文档未见 | — |
| 语速/音量 | ❓ | `audio_params` 下 speech_rate/loudness_rate（待核实） |
| SSML | ❓ | 1.0 文档曾支持，2.0 待核 |
| 克隆 | ✅ | seed-icl-2.0 |
来源：https://docs.volcengine.com/docs/DoubaoVoice/unidirectional-streaming-text-to-speech-http?lang=zh ；https://docs.volcengine.com/docs/DoubaoVoice/DoubaoTextToSpeech20CapabilityIntroduction?lang=zh

### 阿里云百炼 CosyVoice / Qwen-TTS
| 能力 | 状态 | 原生写法 |
|---|---|---|
| SSML | ✅ CosyVoice | `enable_ssml=true` + `<speak>…`（WebSocket 只能一次 continue-task） |
| instruction | ✅ CosyVoice（≤100 字符；格式受音色约束，如 v3-flash 要求"你说话的情感是happy。"） | `instruction` |
| Qwen 情绪/非语言标签 | ⚠️ Qwen-Audio-TTS：`[happy]` `[angry]` `[laughing]`… + instruction；Qwen3-TTS 不支持 instruction（据搜索摘要，需核实） | — |
| LaTeX 朗读 | ✅ | 见 SSML 与 LaTeX 文档 |
来源：https://help.aliyun.com/zh/model-studio/ssml-latex-user-guide ；https://help.aliyun.com/zh/model-studio/cosyvoice-tts-python-sdk ；https://help.aliyun.com/zh/model-studio/non-realtime-tts-user-guide

### 腾讯云 TTS
| 能力 | 状态 | 原生写法 |
|---|---|---|
| 情感 | ✅ 仅多情感音色 | `EmotionCategory`: neutral/sad/happy/angry/fear/news/story/radio/poetry/call/sajiao/disgusted/amaze/peaceful/exciting/aojiao/jieshuo；`EmotionIntensity` 50–200 |
| SSML | ✅ 中文为主 | `<speak>` + break/phoneme/say-as 等（❓完整列表） |
| 语速/音量 | ✅ | Speed / Volume 参数 |
来源：https://cloud.tencent.com/document/product/1073/37995 ；https://www.tencentcloud.com/zh/document/product/1154/47883

### 讯飞 超拟人合成
| 能力 | 状态 | 原生写法 |
|---|---|---|
| 口语化程度 | ✅ | `parameter.oral.oral_level`: high/mid/low |
| 语速/音量/音高 | ✅ | speed/volume/pitch（❓范围） |
| 情绪/非语言 | ❓ | — |
来源：https://www.xfyun.cn/doc/spark/%E8%B6%85%E6%8B%9F%E2%BC%88%E5%90%88%E6%88%90.html

## 2. 开源模型

| 模型 | 已确认控制 | 来源 |
|---|---|---|
| ChatTTS | 词级 `[laugh]` `[uv_break]` `[lbreak]`；句级 refine prompt `[oral_0-9][laugh_0-2][break_0-7]`；`[speed_N]`（效果不稳定）；说话人 embedding 采样 | https://github.com/2noise/ChatTTS ；https://github.com/2noise/ChatTTS/issues/769 |
| CosyVoice2/3 | 行内 `[laughter]` `[breath]` `[cough]` `[sigh]` `[noise]` `[quick_breath]` `[lipsmack]` `[mn]` `[clucking]` `[accent]` `[hissing]` `[vocalized-noise]`；区间 `<strong>…</strong>` `<laughter>…</laughter>`；instruct 文本（方言/情绪/语速）；v3 拼音/ARPAbet 纠音；零样本克隆；流式 | https://arxiv.org/html/2412.10117v3 ；https://github.com/FunAudioLLM/CosyVoice/blob/main/cosyvoice/tokenizer/tokenizer.py |
| IndexTTS2 | 8 维情绪向量 [happy, angry, sad, afraid, disgusted, melancholic, surprised, calm]；`emo_text` 自然语言情绪；情绪参考音频；`emo_alpha` 强度；时长控制（目标时长/倍率） | https://github.com/index-tts/index-tts |
| F5-TTS | 参考音频克隆、speed、nfe 步数；无行内标签 | https://github.com/SWivid/F5-TTS |
| Dia (nari-labs) | 说话人 `[S1]` `[S2]`；非语言 `(laughs)(coughs)(sighs)(gasps)(singing)(mumbles)(groans)(sniffs)(claps)(screams)(inhales)(exhales)(applause)(burps)(humming)(sneezes)(chuckle)(whistles)(beep)` | https://github.com/nari-labs/Dia |
| Kokoro | 发音 `[word](/IPA/)`；重音 `[word](+1)`/`(-1)`；speed | https://github.com/hexgrad/kokoro/issues/170 |

## 3. 横向总结

- **四种控制范式**：①SSML（Azure/Google/Polly/腾讯/阿里）②行内标签（ElevenLabs `[]`、Fish S2 `[]`、Fish S1 `()`、MiniMax `()`/`<#x#>`、Cartesia 类 SSML、Gemini `<>`、ChatTTS/CosyVoice/Dia token）③自然语言指令（OpenAI instructions、Gemini style、Hume description、豆包 context_texts、CosyVoice instruction、IndexTTS emo_text）④结构化参数（MiniMax emotion、腾讯 EmotionCategory、IndexTTS 向量、Cartesia generation_config）。
- **趋势**：从固定枚举走向自由文本（Fish S1→S2、ElevenLabs tags 非枚举、Gemini style）；两个主流 API 生命周期风险（OpenAI gpt-4o-mini-tts 2027-01 关停，Gemini preview TTS 下线公告）。→ DSL 必须把"自由文本表演指令"作为一等公民，并让适配器可插拔。
- **非语言声交集**：laugh、sigh、breath/inhale/exhale、cough、gasp、clear-throat、sniff 在 ≥4 家有原生支持，可作为核心标签。

## 4. Artificial Analysis TTS 榜单头部模型（2026-10）控制能力

榜单名单由上游提供，本文未独立核对排名；下表只核对控制能力。

| 模型 | 控制方式（已核实） | 多人 | 来源 |
|---|---|---|---|
| ElevenLabs Eleven v4 / v4 Turbo | 与 v3 相同的 `[...]` 自由文本 audio tags；v4 Turbo 实时也支持 tags；Text to Dialogue 推荐 v4；不支持 SSML break | ✅ Dialogue API | https://elevenlabs.io/docs/help-center/product/core-capabilities/text-to-speech/how-do-audio-tags-work-with-eleven-v3-and-v4.md |
| Alibaba Qwen-Audio-3.1-TTS-Plus | `input.instruction`（方言/情感/角色）；rate 0.5–2、pitch 0.5–2、volume 0–100；SSML（`enable_ssml`）；`hot_fix` 发音；`seed` 可复现；字级时间戳（流式）。**行内 `[excited]`/`[laughing]` 标签文档仅列 3.1-flash、3.0-plus、3.0-flash，未列 3.1-plus → 需实测** | ❓ | https://www.alibabacloud.com/help/en/model-studio/non-realtime-tts-user-guide ；https://docs.modelstudio.console.alibabacloud.com/zh/model-studio/qwen-audio-tts-http-api |
| Qwen-Audio 行内标签（适用型号见上） | 控制：`[sad][amazed][angry][excited][sarcastic][curious][bored][tired][scornful][shouting][asmr][panicked][mischievously][empathetic][whispers][reluctantly][crying][serious][trembling][very slowly][very fast][deep and loud shouting][like dracula]`（持续到下一个控制标签）；非语言：`[gasp][sighing][clears throat][giggles][laughing][cough][snorts]`… | — | 同上 |
| Cartesia Sonic 3.6 | 同 Sonic-3：`<speed/>` `<volume/>` `<emotion/>`(beta, 英语) `<break/>` `<spell>` `[laughter]`，generation_config | ❌ | https://docs.cartesia.ai/build-with-cartesia/capability-guides/ssml-tags |
| Google Gemini 3.8 Flash TTS | `text` 严格逐字；整轮 `speech_metadata.style` 自然语言（情绪/语速/音高/音量）；行内尖括号人声事件 `<sigh>` `<short pause>` 等；大写强调；Voice design；多说话人 | ✅ | https://ai.google.dev/gemini-api/docs/speech-generation |
| Inworld Realtime TTS-2 | `[...]` 英文自然语言 steering（持续到改变或 `[reset]`，仅 `inworld-tts-2`，flash 忽略）；固定非语言词表约 50 个：`[breathe][sigh][gasp][pant][huff][grunt][groan][moan][laugh][chuckle][giggle][cackle][snort][scoff][cry][sob][wail][whimper][whine][sniffle][sniff][shriek][squeal][howl][clear throat][cough][sneeze][hiccup][yawn][burp][snore][choke][gag][swallow][gulp][spit][tongue click][mouth click][mouth sound][lip smack][kiss][shush][raspberry][whistle][bleh][chew][slurp][babble][beatbox][growl]`；break 标签、自定义发音、verbatim；时间戳+viseme；`delivery_mode` STABLE/BALANCED/CREATIVE | ❓ | https://docs.inworld.ai/tts/capabilities/steering ；https://inworld.ai/realtime-tts-2 |
| Fish Audio Drama 3 | `[...]` 自由指令（任意语言，含中文）；成对词级标签 `<emphasis>` `<whisper>` 等 7 个（soft/faster/slower/stress/forceful，⚠️ 精确标签名待核）；语言标签 `<french>…</french>`（实验）；音素：英文 Arpabet、中文拼音、日文罗马字；temperature/top_p | ✅ `<|speaker:N|>` + `reference_id[]` | https://docs.fish.audio/features/text-to-speech/drama-3 ；https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech |
| MiniMax Speech 2.8 | 见上 MiniMax：19 个 `(laughs)…` 语气词、`<#x#>`、emotion 枚举（不支持 whisper）、拼音/IPA 注音 | ❌ | https://platform.minimax.io/docs/api-reference/speech-t2a-http |
| StepFun StepAudio 2.5 TTS（及 3） | 全局 `instruction`（2.5 ≤200 字，3 ≤500 字）；行内 `(...)` 圆括号指令不朗读，如 `(lowered voice)` `(short gasp)`；speed 0.5–2、volume 0.1–2；`pronunciation_map` 数字声调拼音；SSE/WebSocket 流式；克隆 | ❌ | https://platform.stepfun.ai/docs/en/guides/models/stepaudio-2.5-tts ；https://platform.stepfun.ai/docs/en/api-reference/audio/create-audio |

**榜单结论**：头部模型清一色采用"方括号/圆括号/尖括号 + 自由文本指令"，括号符号互相冲突（`[]` `()` `<>` 含义各异）。DSL 必须由编译器负责转义：Step/MiniMax 文本中的普通圆括号要转义或改写，否则会被当成指令吞掉（StepAudio 明确"括号内容不朗读"）。

## 5. 补充：更多 provider 与声音能力（详见 MARKET_SURVEY.md）

| Provider | 表达控制 | 声音能力 | 来源 |
|---|---|---|---|
| xAI / SpaceXAI Grok TTS | 语音标签（tone/pause/whisper/laughter，具体语法 ❓）、speed 0.7–1.5、`replace` 发音表（重拼或 /IPA/）、`text_normalization`、`with_timestamps` 字符级时间戳 | 28 内置音色；克隆约需 2 分钟，口令+声纹两步验证 | https://x.ai/api/voice ；https://groktexttospeech.com/grok-voice-api/（第三方，基于 docs.x.ai） |
| Mistral Voxtral TTS | 无标签，情绪/口音来自参考音频 | 零样本克隆 3 s（最佳 5–25 s）、跨语种；开放版 CC BY-NC，仅预设音色 | https://mistral.ai/news/voxtral-tts/ |
| Murf Falcon 2 | ❓ | 10 s 克隆（企业，需同意） | https://murf.ai/falcon |
| Deepgram Aura-2 | 少 | 无克隆🔸 | https://benchmarks.coval.ai/pricing |
| ElevenLabs 声音 | — | IVC（<2 min）、PVC（≥30 min）、Voice Design（20–1000 字描述）、Voice Remix（提示词+强度 low/medium/high/max）、Voice Library、voice-captcha | https://elevenlabs.io/docs/overview/capabilities/voices.mdx ；https://elevenlabs.io/docs/overview/capabilities/voice-remixing.mdx |
| MiniMax 声音 | — | 克隆 10 s–5 min + 可选 <8 s clone_prompt；voice design；`timbre_weights` 混音色；需实名认证 | https://platform.minimaxi.com/docs/api-reference/voice-cloning-clone |
| 豆包 声音复刻 2.0 | — | 宣称 5 s；最佳 14–30 s | https://docs.volcengine.com/docs/DoubaoVoice/SoundReplication20BestPractices?lang=zh |
| 阿里 CosyVoice / Qwen-Audio 声音 | — | 克隆 10–20 s（≤60 s）；voice design；`enable_aigc_tag` 隐性水印 | https://help.aliyun.com/en/model-studio/cosyvoice-clone-api-reference ；https://docs.modelstudio.console.alibabacloud.com/zh/model-studio/qwen-audio-tts-http-api |

**更新的映射**：MiniMax 克隆试听接口的语气词列表比 T2A 文档多 `(whistles)` `(crying)` `(applause)`（共 22 个），以克隆接口文档为准需复核。
