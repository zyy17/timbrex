> 中文版。英文主文档：[MARKET_SURVEY.md](MARKET_SURVEY.md)。

# TTS 市场与声音能力调研（2026-10-20）

图例：✅ 本次查到官方/一手来源；🔸 来自第三方汇总（价格聚合站、评测博客），需复核；❓ 未查证（凭已知信息或空缺）。
价格为标价，随时变动；"$/1M 字符"是归一化口径。表达控制细节见 `CAPABILITIES.md`，本文侧重**市场全景 + 声音（克隆/设计/转换）能力**。

## 1. 排行榜快照（Artificial Analysis，2026-10）

- **Provider Voice Arena 头部**（据 orcarouter 转述 AA 数据 🔸）：ElevenLabs Eleven v4 Turbo ≈1330、Eleven v4 ≈1328、Qwen-Audio-3.1-TTS-Plus ≈1300、Cartesia Sonic 3.6 ≈1286、Gemini 3.8 Flash TTS ≈1274、Qwen-Audio-3.0-TTS-Plus ≈1268、Speechify Simba 3.2 ≈1245、Fish Audio Drama 3 Preview ≈1243（第 10）。共 ~97 个模型，17 个开放权重。
  来源：https://artificialanalysis.ai/text-to-speech/leaderboard/provider-voice ；https://www.orcarouter.ai/blog/fish-audio-drama-3-tops-open-weights-speech-arena
- **开放权重榜前 5**（✅ AA 页面）：Fish Audio Drama 3 Preview 1247、Breeze TTS 2 1221、Ming-3.0-tts 1188、Fish Audio S2 Pro 1113、Step Audio EditX (Mar 2026) 1099。
  来源：https://artificialanalysis.ai/text-to-speech/leaderboard/provider-voice/open-weights
- **克隆（Controlled Voice）榜**：Qwen-Audio-3.1 位列前二（1184±14）🔸，完整名次未取到。来源：https://artificialanalysis.ai/text-to-speech/leaderboard/controlled-voice
- 注：Breeze TTS 2、Ming-3.0-tts 在榜上出现，但本次未查到其官方仓库/许可 ❓。

## 2. 商业 API 总表

| Provider / 主力模型 | 中文/方言 | 表达控制（摘要） | 流式 | 价格（$/1M 字符） | 状态 | 来源 |
|---|---|---|---|---|---|---|
| ElevenLabs Eleven v4 / v4 Turbo / v3 / Flash v2.5 | 中文✅（70+ 语）；方言❓ | 自由 `[tags]`、Dialogue API | ✅（v4 Turbo 实时） | v4 Turbo/Flash $40；v4 $80（🔸） | 活跃 | https://elevenlabs.io/pricing ；https://silma.ai/blog/text-to-speech-price-comparison-2026 |
| OpenAI gpt-4o-mini-tts | 中文✅ | instructions | ✅ | $12/1M 音频 token（🔸） | **弃用，2027-01-06 关停** | https://developers.openai.com/api/docs/deprecations |
| Google Gemini 3.8 Flash / Flash-Lite TTS | 中文✅ | style + `<tag>`，多人 | ✅ | $9 / $6 每 1M 音频 token（预览价，2027-01-01 起翻倍 🔸） | 预览 | https://ai.google.dev/gemini-api/docs/speech-generation |
| Google Chirp 3 HD | 中文✅ | `[pause]`，SSML 子集 | ✅ | $30 | GA | https://docs.cloud.google.com/text-to-speech/docs/chirp3-hd |
| Azure Neural / Neural HD | 中文+多方言（粤/吴/川等 ❓具体列表待核） | 全 SSML、express-as | ✅ | Neural 承诺价低至 $7.5；HD $22（🔸） | 活跃 | https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice |
| AWS Polly Neural / Generative | 中文✅（普通话、粤语 ❓） | SSML 子集 | ✅ | Neural $16；Generative $30（🔸） | 活跃 | https://docs.aws.amazon.com/polly/latest/dg/supportedtags.html |
| Cartesia Sonic 3.6 | 中文✅ | 类 SSML + emotion（英） | ✅ 低延迟 | ≈$37（Scale 计划折算 🔸） | 活跃 | https://docs.cartesia.ai/build-with-cartesia/capability-guides/ssml-tags |
| Hume Octave 2 | ❓ | description（O2 待上线）、speed | ✅ | $50（🔸） | 活跃 | https://dev.hume.ai/voice/docs/text-to-speech-tts/acting-instructions.md |
| Inworld Realtime TTS-2 / Flash | 中文✅（70+ 语 🔸） | `[steering]` + ~50 非语言 | ✅ | TTS-2 $12.5–25；Flash $7–15（🔸） | 活跃 | https://docs.inworld.ai/tts/capabilities/steering |
| Fish Audio S2.1-Pro / Drama 3 | 中文✅ | 自由 `[...]`、词级标签、多人 | ✅ | S2.1 Pro $15；Drama 3 $15（🔸） | 活跃；S1 2026-12-31 退役 | https://docs.fish.audio/features/text-to-speech/drama-3 |
| MiniMax Speech 2.8 HD/Turbo | 中文+粤语✅（language_boost 40 种） | `(laughs)` 等、`<#x#>`、emotion、注音 | ✅ | Turbo $60（🔸） | 活跃 | https://platform.minimax.io/docs/api-reference/speech-t2a-http |
| 火山 豆包语音合成 2.0 | 中文✅；方言❓ | context_texts 指令 | ✅ | ❓（人民币计费） | 活跃 | https://docs.volcengine.com/docs/DoubaoVoice/DoubaoTextToSpeech20CapabilityIntroduction?lang=zh |
| 阿里 Qwen-Audio-3.x / CosyVoice | 中文✅、方言 via instruction | SSML、instruction、标签 | ✅ | Qwen-Audio-3.0-Plus $20（新加坡 🔸）；Qwen3 TTS Flash Realtime $13（🔸） | 活跃 | https://www.alibabacloud.com/help/en/model-studio/non-realtime-tts-user-guide |
| 腾讯云 TTS | 中文✅、粤语等❓ | SSML、EmotionCategory | ✅ | ❓ | 活跃 | https://cloud.tencent.com/document/product/1073/37995 |
| 讯飞 超拟人合成 | 中文✅、方言❓ | oral_level | ✅ | ❓ | 活跃 | https://www.xfyun.cn/doc/spark/%E8%B6%85%E6%8B%9F%E2%BC%88%E5%90%88%E6%88%90.html |
| 百度 智能云 TTS | 中文✅ | ❓（本次未查） | ❓ | ❓ | ❓ | ❓ |
| StepFun StepAudio 2.5 / 3 TTS | zh/en（ja/ko/fr/es 预览） | instruction + 行内 `(…)` | ✅ SSE/WS | ❓ | 活跃 | https://platform.stepfun.ai/docs/en/api-reference/audio/create-audio |
| Speechify Simba 3.2 | ❓（60+ 语 🔸） | ❓ | ✅ | $6–10（🔸） | 活跃 | https://speechify.com/blog/best-text-to-speech-api-voice-quality-price/ |
| Deepgram Aura-2 / Flux TTS | **无中文**（de/en/es/fr/it/ja/nl 🔸） | 少 | ✅ 低延迟 | Aura-2 $27–30；Flux $40.5（🔸） | 活跃 | https://benchmarks.coval.ai/pricing |
| Rime Mist v3 / Coda | ❓ | ❓ | ✅ | $30 / $50（🔸） | 活跃 | 同上 |
| LMNT | ❓ | ❓ | ✅ | 未公开（🔸） | ❓ | ❓ |
| Murf Falcon 2 | 中文✅（35+ 语） | ❓ | ✅ | $10（1¢/分钟） | 活跃 | https://murf.ai/falcon |
| Resemble | ❓ | ❓ | ❓ | ❓ | ❓ | ❓（本次未查） |
| Smallest.ai Lightning v3.1 | 15 语（中文❓） | ❓ | ✅ | $19.5（🔸） | 活跃 | https://benchmarks.coval.ai/pricing |
| Mistral Voxtral TTS | **无中文**（en/fr/de/es/nl/pt/it/hi/ar） | 情感表现（参考音频驱动） | ✅ | $16 | 活跃（2026-03 发布） | https://mistral.ai/news/voxtral-tts/ |
| Gradium | 无中文（de/en/es/fr/ja/ko/pt 🔸） | ❓ | ✅ WS 优先 | ≈$0.01/分钟（🔸） | 活跃 | https://slng.mintlify.app/models |
| Soniox TTS RT v1/v2 | 中文✅（🔸） | ❓ | ✅ | $14.1（🔸） | 活跃 | https://benchmarks.coval.ai/pricing |
| SpaceXAI(xAI) Grok TTS | 中文✅（20+ 语） | 语音标签（tone/pause/whisper/laugh）、speed、replace 发音、字符级时间戳 | ✅ | $15 | 活跃 | https://x.ai/api/voice ；https://groktexttospeech.com/grok-voice-api/（第三方指南） |
| PlayHT | — | — | — | — | **API 已于 2025-12-31 关停**（🔸） | https://www.versusref.com/tts/alternatives/voxtral-tts/ |
| Kling 等视频厂商 | 配音功能附属于视频产品，未作为独立 TTS API 调研 ❓ | | | | | |

## 3. 声音能力对比（商业）

| Provider | 零样本克隆（最短参考） | 专业/微调克隆 | 文本设计音色 | 音色改造 / S2S | 跨语种克隆 | 音色库 | 同意/安全 | 水印 |
|---|---|---|---|---|---|---|---|---|
| ElevenLabs | ✅ IVC，约 1–2 分钟最佳（<2 分钟） | ✅ PVC ≥30 分钟，理想 3 小时，训练 3–6 h，Creator 起 | ✅ Voice Design（描述 20–1000 字，出 3 个预览） | ✅ Voice Remix（提示词改性别/口音/风格，强度 Low–Max）；Speech-to-Speech ✅ | ✅（多语种） | ✅ 17,000+，PVC 可共享并分成 | PVC 用 voice-captcha 验证本人 | ❓（AI 语音分类器 ❓） |
| MiniMax | ✅ 10 s–5 min（mp3/m4a/wav ≤20MB），可附 <8 s 示例 | ❓ | ✅ voice design | ❓ | ✅（language_boost） | 系统音色 ✅ | 需个人/企业认证；7 天未用删除 | ❓ |
| 豆包 声音复刻 2.0 | ✅ 产品页称 5 s；最佳实践 14–30 s | ❓ | ❓ | ❓ | 建议 prompt 中英都覆盖 | ✅ | ❓ | ❓ |
| 阿里 CosyVoice（百炼） | ✅ 10–20 s（≤60 s，含 ≥5 s 连续段） | ❓ | ✅（Qwen-Audio voice design） | ❓ | ❓ | ✅ | 用户自负版权；每账号 ≤1000 个，1 年未用下线 | Qwen-Audio 可嵌入 AIGC 隐性标识（`enable_aigc_tag`）✅ |
| Fish Audio | ✅（参考音频 / reference_id）❓最短 | ❓ | ❓ | ❓ | ✅ | ✅ 大型社区库 | ❓ | ❓ |
| Mistral Voxtral | ✅ 3 s（最佳 5–25 s），仅 API | ❌ | ❌ | ❌ | ✅ 零样本跨语种（涌现） | 20 预设 | ❓ | ❓ |
| xAI Grok | ✅ 约 2 分钟自然语音 | ❌ | ❌ | S2S 对话 API（非变声） | ✅ 继承多语种 | 28 内置 | 口令 + 声纹匹配两步验证 ✅ | ❓ |
| Murf Falcon 2 | ✅ 10 s（企业） | ❓ | ❓ | ❓ | ❓ | 150+ | 声称需说话人明确同意 | ❓ |
| Inworld | ✅ 3 s；专业克隆 beta | ✅ beta | ❓ | ❓ | ❓ | ✅ | ❓ | ❓ |
| Gemini TTS | ❓ | ❌ | ✅ Voice design（文档提及） | ❌ | — | 30 风格 | ❓ | SynthID ❓ |
| Cartesia | ✅ IVC（❓最短） | ✅ PVC（文档提及 PVC 不支持请求级调速） | ❓ | ✅ Voice changer ❓ | ❓ | ✅ | ❓ | ❓ |
| Hume | ❓ | ❓ | ✅ 用描述生成音色 | ❓ | ❓ | ✅ | ❓ | ❓ |
| Azure | ✅ Personal Voice（需同意录音 ❓细节） | ✅ Custom Neural Voice（受限访问，需审批） | ❓ | ❓ | ✅ | ✅ 数百音色 | 受限访问 + 说话人同意声明 ❓ | 自动水印 ❓ |
| Google Chirp 3 | ✅ Instant Custom Voice（受限）❓ | ❓ | ❓ | ❓ | ❓ | ✅ | ❓ | ❓ |
| AWS Polly | ❌（Brand Voice 定制项目 ❓） | ❓ | ❌ | ❌ | — | ✅ | — | — |
| Speechify / Smallest / Rime / LMNT / Resemble / Deepgram / Gradium / 腾讯 / 讯飞 / 百度 / StepFun | Speechify 支持克隆🔸；Deepgram Aura-2 无克隆🔸；Gradium 即时克隆🔸；StepFun 有克隆预览✅；其余 ❓ | ❓ | ❓ | Resemble 有 S2S/水印产品（❓） | ❓ | ❓ | ❓ | ❓ |

## 4. 开源模型

| 模型 | 许可 | 中文/方言 | 表达控制 | 克隆（最短参考） | 备注 / 来源 |
|---|---|---|---|---|---|
| Fish Audio Drama 3 Preview | 开放权重（许可 ❓） | ✅ | 自由 `[...]`、词级标签、多人 | ✅ | 35B MoE（3B 激活），AA 开源第 1 🔸 https://www.orcarouter.ai/blog/fish-audio-drama-3-tops-open-weights-speech-arena |
| Fish Audio S2 Pro | 开放权重（许可 ❓） | ✅ | 自由 `[...]` | ✅ | AA 开源第 4 |
| Step Audio EditX | ❓ | ✅ | 情绪/风格编辑 ❓ | ✅ ❓ | AA 开源第 5 |
| Breeze TTS 2 / Ming-3.0-tts | ❓ | ❓ | ❓ | ❓ | AA 开源 #2/#3，未查到一手资料 |
| CosyVoice 2/3 | Apache-2.0 ❓ | ✅ 多方言（instruct） | `[laughter]` `[breath]` `<strong>`、instruct、拼音 | ✅ 3–10 s ❓ | https://github.com/FunAudioLLM/CosyVoice |
| IndexTTS2 | 自定义许可（❓商用条款） | ✅ | 8 维情绪向量、emo_text、时长控制 | ✅ 音色/情绪参考分离 | https://github.com/index-tts/index-tts |
| F5-TTS | 代码 MIT，权重 CC-BY-NC ❓ | ✅ | speed | ✅ 参考音频+文本 | https://github.com/SWivid/F5-TTS |
| ChatTTS | 代码 AGPL ❓ / 权重 CC-BY-NC ❓ | ✅ | `[laugh]` `[uv_break]` `[oral_N]` | 说话人采样（非克隆） | https://github.com/2noise/ChatTTS |
| Dia | Apache-2.0 ❓ | ❌（英语） | `[S1][S2]` + 19 种 `(laughs)` | ✅ 音频提示 | https://github.com/nari-labs/Dia |
| Kokoro 82M | Apache-2.0 ❓ | ✅（有限） | IPA、重音 | ❌ | https://github.com/hexgrad/kokoro |
| Chatterbox | MIT🔸 | 多语 ❓ | 情绪夸张度参数 ❓ | ✅ | 输出带水印🔸 https://www.cartesia.ai/learn/open-source-tts |
| Orpheus | Apache-2.0🔸 | ❓ | `<laugh>` 等 ❓ | ✅ ❓ | 同上 |
| Spark-TTS | Apache-2.0🔸 | ✅ | 性别/音高/语速参数 ❓ | ✅ | 同上 |
| MegaTTS3 | Apache-2.0🔸 | ✅ | ❓ | ✅（官方曾限制 encoder 发布 ❓） | 同上 |
| Higgs Audio v2 / v3 | v2 Apache-2.0；v3 仅研究🔸 | ✅ | 多人、音效 ❓ | ✅ | Higgs Audio V3 在 open-weights 小榜 1044🔸 https://theopenweights.com/leaderboards/text-to-speech |
| VibeVoice | MIT 但声明仅研究🔸 | ✅ | 长时多人播客（最多 4 人 ❓） | ✅ | https://www.cartesia.ai/learn/open-source-tts |
| Kyutai TTS / Pocket TTS | ❓ | ❓ | 流式 | Pocket TTS 约 5 s，CPU 可跑🔸 | https://kyutai.org/pocket-tts/ |
| Sesame CSM | Apache-2.0 ❓ | ❌ | 对话上下文 | ✅ | ❓ |
| XTTS v2 (Coqui) | CPML 非商用 ❓；Coqui 公司已关闭 | ✅ | — | ✅ 6 s ❓ | ❓ |
| GPT-SoVITS | MIT ❓ | ✅ 中/粤/日/英 | — | ✅ 5 s 零样本，1 分钟微调 ❓ | https://github.com/RVC-Boss/GPT-SoVITS |
| Mistral Voxtral TTS（开放版） | CC BY-NC 4.0 ✅ | ❌ | — | 仅 20 预设，开放版不能克隆 | https://mistral.ai/news/voxtral-tts/ |
| Maya 2 Flash | ❓ | ❓ | ❓ | ❓ | open-weights 小榜第 1🔸 |

## 5. 结论（对 VoxScript 的意义）

1. **中文能力第一梯队**：Qwen-Audio、MiniMax、豆包、Fish、CosyVoice、Inworld、ElevenLabs、Azure；Deepgram、Mistral Voxtral、Gradium 无中文。
2. **克隆门槛已降到 3–30 秒**（Voxtral 3 s、Inworld 3 s、豆包 5 s 宣称、MiniMax 10 s、CosyVoice 10–20 s、Murf 10 s），而 ElevenLabs IVC/Grok 要 1–2 分钟；专业克隆仍需 30 分钟以上。
3. **"声音设计"成标配**：ElevenLabs、MiniMax、Qwen-Audio、Gemini、Hume 都能用文本描述生成音色；ElevenLabs 还有 Remix（在现有音色上改）。
4. **合规差异大**：ElevenLabs voice-captcha、xAI 口令+声纹、MiniMax 实名、Azure 受限访问、阿里 AIGC 隐性标识、Chatterbox 水印 → DSL 必须把 consent / 水印声明作为一等元数据。
5. **生命周期风险**：OpenAI gpt-4o-mini-tts（2027-01-06）、Fish S1（2026-12-31）、PlayHT（已关）、Gemini 预览价 2027 翻倍 → 适配器必须可插拔、caps 带 deprecated。
6. **开源**：Fish Drama 3/S2 Pro 已领跑开放权重；许可是选型关键（Voxtral、F5、ChatTTS、XTTS、Higgs v3、VibeVoice 都有非商用/研究限制）。
