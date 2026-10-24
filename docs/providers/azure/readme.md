# Azure Speech

[简体中文](readme.zh-cn.md) · [Timbrex setup](../../providers.md) · [Supported controls](../../capabilities.md)

Developer quick reference, reviewed **2026-10-24**. This covers the vendor’s TTS offering only, not full Timbrex support; see the links above for the current adapter scope.

## Where it fits

Azure Speech is worth considering for multilingual applications that need explicit pronunciation controls, a broad voice catalog and regional deployment choices. Standard neural voices and custom voices serve different needs; feature availability depends on voice, model and region. [Overview](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/text-to-speech), [voice catalog](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/language-support?tabs=tts)

## Features and tradeoffs

- **Voice families:** standard and multilingual neural voices, expressive DragonHD/Omni families, and preview MAI-Voice-2.1/2.1-Flash. MAI Flash targets low-latency interaction; MAI targets expressive long-form speech. Preview features have no SLA and are not recommended for production. [HD voices](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/high-definition-voices), [MAI voices](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/mai-voices)
- **Pronunciation and delivery:** SSML provides phonemes, lexicons, substitutions, number/date interpretation and mathematical reading. Voice-dependent styles, roles, emphasis and prosody offer additional control. Not every voice supports every element. [Pronunciation](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-pronunciation), [delivery controls](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice)
- **Audio and timing:** multiple codecs and sample rates, incremental audio delivery, and a separate asynchronous batch API for long recordings. SDK workflows can expose timing/events and stream incoming text; input-text streaming does not support SSML. [REST audio](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/rest-text-to-speech), [batch](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/batch-synthesis), [input streaming](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/how-to-lower-speech-synthesis-latency#input-text-streaming)
- **Dialogue synthesis:** supported multi-talker voices synthesize a scripted conversation with context across turns. This is text-to-dialogue TTS, not a conversational agent API. [Multi-talker SSML](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-synthesis-markup-voice#multi-talker-voice-example)

## Limits worth knowing

Realtime synthesis produces at most 10 minutes per request. For longer output, split the text into requests or use Batch synthesis with a supported voice on S0. F0 allows 20 transactions per 60 seconds; S0 defaults to 30 transactions/second. A voice's regional capacity can still cause throttling below an account quota. [Quotas](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/speech-services-quotas-and-limits#text-to-speech-quotas-and-limits-per-resource)

Microsoft's HD support table and examples disagree on some `express-as` and `prosody` support. Validate the exact voice/version before relying on these controls. The MAI-specific guide documents 2.1 models while the broader catalog still lists some 2-series IDs. [HD support](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/high-definition-voices#supported-and-unsupported-ssml-elements-for-azure-speech-hd-voices), [MAI guide](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/mai-voices)

For new long-form work with batch-supported voices, prefer Batch synthesis. The older Long Audio API is scheduled to retire on **2027-04-01**. [Lifecycle and migration](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/batch-synthesis)

## Pricing

The official page lists **0.5 million Neural characters/month free on F0**. Paid synthesis is billed by characters, with separate categories for Neural/Neural HD Flash, Neural HD, MAI and custom voices. Commitment tiers have eligibility exclusions. [Official pricing](https://azure.microsoft.com/en-us/pricing/details/speech/)

As of 2026-10-24, the accessible pricing page showed paid amounts as `$-`; no regional paid rate was verified. Select the deployment region, currency and agreement in the official calculator before budgeting. Do not treat an old quoted rate as a universal Azure price.
