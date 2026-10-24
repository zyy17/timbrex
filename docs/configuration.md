# Configuration

[简体中文](configuration.zh-cn.md) · [Provider setup](providers.md) · [Tutorial](tutorial.md)

Use configuration to set a default provider, model, voice, and strict mode. `check`, `synth`, and `config` load one YAML file and the documented environment variables. Scripts and command-line flags can override the relevant defaults; credentials and endpoints come only from configuration or the environment.

## File discovery

By default, `check`, `synth`, and `config` read `timbrex.yaml` from the **current working directory**, if present. They do not search the script directory, parent directories, or the home directory. `--config` selects one file. A missing explicit file is an error.

Scripts and configuration inputs must be regular files; symlinks to regular files are supported. Named pipes (FIFOs), devices, and directories are rejected.

Interrupt and SIGTERM cancel the command context. The CLI checks cancellation between configuration loading, reading, parsing, checking, and lowering, and before reporting check success or starting synthesis. These checkpoints do not interrupt an in-progress parser, checker, or filesystem operation. Once synthesis has successfully committed the output, a later cancellation does not turn that completed operation into a failure.

```bash
./timbrex --config examples/timbrex.yaml config
./timbrex --config examples/timbrex.yaml check examples/expressive.tbx
./timbrex --config examples/timbrex.yaml synth examples/expressive.tbx --provider stub -o tone.wav
```

`--config` is a global flag and works before or after the command. Help, version, and shell completion do not load configuration. Timbrex does not load `.env` files, expand `${VARIABLE}` in YAML, or watch files for changes.

## Local environment file

[.env.example](../.env.example) lists every supported environment variable, grouped by provider. All assignments are commented out and empty, so the unchanged template preserves existing environment variables, YAML settings, and built-in defaults. It does not select a paid provider. The `stub` provider needs no credentials; configure only the speech services you intend to use. Required credentials, voices, and endpoints are explained in [provider setup](providers.md).

From the repository root, create a local copy (keep an existing `.env` rather than overwriting it):

```bash
cp -i .env.example .env
chmod 600 .env
```

Edit `.env` locally, uncomment only the variables you need, and put each value inside single quotes. Do not add spaces around `=`. Single quotes keep characters such as `$`, spaces, and `#` literal; a value containing a single quote needs shell escaping. Nonempty exported values override YAML settings; leaving an unused line commented also preserves an existing shell value.

Timbrex has no automatic dotenv loading or dotenv dependency. In a POSIX-compatible shell such as Bash or Zsh, manually export the edited file before running Timbrex:

```bash
set -a
. ./.env
set +a
./timbrex config
```

The `.` command executes shell code: source only your own edited file after reviewing it, not an untrusted downloaded file. These exports apply to the current shell and its child processes. `config` redacts API keys and does not contact providers; a later `synth` command using a real provider can incur charges. The repository ignores `.env` and `.env.*` local variants while retaining `.env.example`. Never put real credentials in the template, commit them, or paste them into chat.

## File format

```yaml
provider: openai
strict_tags: true
providers:
  openai:
    model: gpt-4o-mini-tts
    voice: coral
    endpoint: https://api.openai.com/v1
  azure:
    region: eastus
    voice: en-US-JennyNeural
```

The file contains one YAML mapping. Keys are case-sensitive. Unknown keys, unknown provider IDs, incorrect value types, duplicate keys, multiple YAML documents, and null values are errors. An empty file is valid. Invalid file values fail even when an environment variable would override them.

| Key | Meaning |
| --- | --- |
| `provider` | Default provider; initially `stub` |
| `strict_tags` | Default tag checking mode; initially `false` |
| `providers.<id>.api_key` | Credential for that provider; its environment variable is an alternative |
| `providers.<id>.model` | Default model; Azure selects models through its voice and rejects a model |
| `providers.<id>.voice` | Fallback voice for speakers without cast mappings |
| `providers.<id>.endpoint` | API base URL for OpenAI, ElevenLabs, MiniMax, Gemini, and Fish; full request URL for Azure and Qwen |
| `providers.azure.region` | Azure region used when no endpoint is set |
| `providers.<id>.synthesis` | Typed per-request options described below; not stored voice settings |

Provider IDs are `stub`, `openai`, `elevenlabs`, `minimax`, `azure`, `qwen`, `gemini`, and `fish`. Defaults and the supported environment variable names are listed in [provider setup](providers.md). The [complete example](../examples/timbrex.yaml) includes the speech services. The stub ignores speech settings.

Endpoints must be HTTP(S) URLs without embedded credentials, query strings, or fragments. Credentials are sent to the configured endpoint; use a trusted service or gateway. Azure's explicit endpoint takes priority over its region. Checking scripts and displaying configuration do not require credentials; synthesis checks the selected provider's credentials and endpoint.

## Precedence

For each setting, the highest supplied value wins:

1. CLI flags (`--provider`, `--model`, `--voice`, strict-mode flags).
2. Script front matter (`provider`, `model`, `voice`, `strict_tags`).
3. Environment variables.
4. The selected YAML file.
5. Built-in defaults.

`TIMBREX_PROVIDER` overrides the file's default provider. `TIMBREX_STRICT_TAGS` overrides its strict mode and accepts Go boolean forms, including `true` and `false`. Existing vendor variables such as `OPENAI_API_KEY`, `OPENAI_MODEL`, and `OPENAI_BASE_URL` remain supported. Unset or empty environment variables leave lower-priority settings intact; unrelated variables are ignored.

A cast voice mapping still takes priority over the fallback voice, including `--voice`. A `voice` or `as` span overrides the voice selection within its scope; it does not change the speaker ID. Model and voice IDs are not translated when switching providers. Endpoint and credential settings never come from scripts.

`--strict-tags=false` explicitly disables strict mode; `--no-strict-tags=false` enables it. The two flag names are mutually exclusive. This preserves the distinction between an omitted boolean and an explicit `false`.

## Provider synthesis options

The selected YAML file supports typed per-request settings under `providers.elevenlabs.synthesis`, `providers.fish.synthesis`, `providers.azure.synthesis`, `providers.qwen.synthesis`, and `providers.minimax.synthesis`. There are no CLI, environment, or script-front-matter overrides for these settings. Other providers reject a `synthesis` section; this is not arbitrary API passthrough and adds no DSL syntax.

Omitted fields preserve existing requests. Explicit `0`, `false`, empty strings where permitted, and empty lists remain explicit; lists are not merged or deduplicated. An empty mapping supplies no overrides. Null, unknown or dotted schema keys, incorrect scalar types, non-finite numbers, and out-of-range values fail locally. Every segment is prepared before the first HTTP request or output change. These settings neither create resources nor modify saved voices/dictionaries.

### ElevenLabs

[Create Speech](https://elevenlabs.io/docs/api-reference/text-to-speech/convert) supports the following request overrides; see also the [voice settings schema](https://elevenlabs.io/docs/api-reference/voices/settings/update).

```yaml
providers:
  elevenlabs:
    synthesis:
      seed: 42
      apply_text_normalization: auto
      apply_language_text_normalization: false
      voice_settings:
        stability: 0.5
        similarity_boost: 0.75
        style: 0
        use_speaker_boost: false
      pronunciation_dictionary_locators:
        - pronunciation_dictionary_id: YOUR_EXISTING_DICTIONARY_ID
          version_id: YOUR_EXISTING_VERSION_ID
```

- `seed`: integer `0`–`4294967295`, sent unchanged to each segment. Determinism is best effort; changing segmentation, text, model, or voice changes the request.
- `apply_text_normalization`: `auto`, `on`, or `off`. `apply_language_text_normalization` is a boolean; ElevenLabs currently documents Japanese support and potentially substantial added latency. It does not replace the existing `lang` rules.
- `voice_settings.stability` and `similarity_boost`: numbers `0`–`1`. `style` is a finite number for vendor style exaggeration, not the textual DSL `style`; the linked API schema publishes no numeric bounds, so Timbrex does not invent them. `use_speaker_boost` is boolean. These settings coexist with existing DSL speed without overriding it.
- `pronunciation_dictionary_locators`: at most three existing dictionaries, applied in supplied order. Each entry requires `pronunciation_dictionary_id`; `version_id` is optional. Timbrex locally rejects empty or invalid UTF-8 IDs and IDs/explicit versions containing whitespace or control characters, preserves duplicates, and sends `[]` when explicitly supplied. It does not fetch dictionary contents or validate model compatibility of their individual pronunciation rules.

The [Playground guide](https://elevenlabs.io/docs/eleven-creative/playground/text-to-speech) describes model-dependent UI availability that is not a complete HTTP eligibility contract. Passing local schema checks does not prove every setting is effective on every model. No new model gate is inferred from hidden UI controls. See [the configuration example](../examples/elevenlabs-synthesis.yaml); set your accessible voice and API key before synthesis.

### Fish

The [TTS API](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech) and [OpenAPI schema](https://github.com/fishaudio/docs/blob/main/api-reference/openapi.json) define these S2 request controls:

```yaml
providers:
  fish:
    synthesis:
      normalize: true
      normalization_language: en
      prosody:
        normalize_loudness: false
      temperature: 0.7
      top_p: 0.7
      chunk_length: 200
      min_chunk_length: 0
      latency: balanced
      condition_on_previous_chunks: false
      early_stop_threshold: 0
      features: [quality-guard]
      pronunciation_dictionary:
        - items:
            - key: Timbrex
              value: "tɪmbɹɛks"
              case_sensitive: false
```

- `normalize` and `prosody.normalize_loudness`: booleans. Omission retains Timbrex's explicit `normalize: false` and loudness normalization `true`, not new service defaults. Loudness normalization is effective on the three supported S2 models.
- `normalization_language`: maps to API `language` for normalization/alignment, not spoken language or accent. At most 35 ASCII letters, digits, `_`, or `-`; `""` and `auto` are permitted. It does not make DSL `lang` supported.
- `temperature`, `top_p`, `early_stop_threshold`: numbers `0`–`1`.
- `chunk_length`: integer `100`–`300`; `min_chunk_length`: integer `0`–`100`. These control server-side chunks within a request, not Timbrex script segmentation.
- `latency`: `low`, `normal`, or `balanced`; `condition_on_previous_chunks`: boolean.
- [`features`](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech#body-features): a list accepting only `quality-guard`; unknown flags fail locally. `[]` explicitly requests no flags; omission adds no field. Backend availability determines whether the feature is effective; this is not a quality guarantee.
- [`pronunciation_dictionary`](https://docs.fish.audio/api-reference/endpoint/openapi-v1/text-to-speech#body-pronunciation-dictionary): a direct list of at most three dictionaries, all inline or all references, never mixed. Inline dictionaries require `items` with at most 5000 entries. Each entry requires `key` (1–256 Unicode characters) and phonetic `value` (1–1024); optional `case_sensitive` is boolean. Both strings must be valid UTF-8 and exclude `<|phoneme_start|>` and `<|phoneme_end|>`.

Alternatively, reference existing dictionaries by both `id` and immutable `version`, each matching `[A-Za-z0-9_-]{1,128}`; there is no implicit latest version:

```yaml
providers:
  fish:
    synthesis:
      pronunciation_dictionary:
        - id: YOUR_EXISTING_DICTIONARY_ID
          version: YOUR_IMMUTABLE_VERSION
```

The [official dictionary schemas](https://github.com/fishaudio/docs/blob/main/api-reference/openapi.json) specify request-wide plain substring matching, leftmost-longest, with ASCII case folding unless `case_sensitive: true`. The backend substitutes phonetic forms, not ordinary spelling aliases. Timbrex does not rewrite script/IR text or fetch dictionary contents. Unresolvable references are silently dropped by the service, so a successful request does not prove a dictionary was applied. Lists retain order and duplicates; explicit `[]`, `items: []`, and `case_sensitive: false` are preserved. Omitted fields stay omitted; `null` is rejected locally. Dictionary data is copied into a private snapshot once per synthesis invocation and reused for its segments.

`max_new_tokens` and `repetition_penalty` remain outside this schema. See [the configuration example](../examples/fish-synthesis.yaml).

### Azure

[`deployment_id`](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/rest-text-to-speech#custom-voices) routes synthesis to one existing custom voice deployment per invocation:

```yaml
providers:
  azure:
    endpoint: https://eastus.voice.speech.microsoft.com/cognitiveservices/v1
    voice: YOUR_DEPLOYED_CUSTOM_VOICE_NAME
    synthesis:
      deployment_id: YOUR_EXISTING_DEPLOYMENT_ID
```

Use the deployment's actual hosting endpoint without its query, not the standard regional `.tts.speech` endpoint. An explicit endpoint is required: Timbrex does not guess the hosting region or rewrite a standard host. It adds the opaque ID as an encoded `deploymentId` query parameter; it does not require UUID syntax. Blank, invalid UTF-8, or control-character IDs fail locally. The endpoint still rejects credentials, query strings (including a bare `?`), and fragments.

Set `AZURE_SPEECH_KEY` to the key of the resource hosting that deployment, and `voice` to its actual deployed voice name, not the deployment ID. For an opaque voice name without a recognizable locale prefix, use explicit script language, for example:

```tbx
---
timbrex: 1
lang: en-US
---
Hello from my custom voice.
```

See Microsoft's [custom voice usage](https://learn.microsoft.com/en-us/azure/ai-services/speech-service/professional-voice-deploy-endpoint#use-your-custom-voice) and [the configuration example](../examples/azure-synthesis.yaml). This only invokes synthesis; it does not create, train, deploy, manage, or discover voices. Voice/SSML restrictions still apply. Omitting `deployment_id` preserves ordinary endpoint/region routing; no new model field or DSL/IR syntax is introduced.

### Qwen

The [Qwen-Audio-TTS HTTP API](https://www.alibabacloud.com/help/en/model-studio/qwen-audio-tts-http-api) accepts `input.seed` and `input.hot_fix`:

```yaml
providers:
  qwen:
    synthesis:
      seed: 0
      hot_fix:
        pronunciation:
          - text: 天气
            replacement: tian1 qi4
        replace:
          - text: 今天
            replacement: 金天
```

`seed` is an integer `0`–`65535`, unchanged for each segment. Each hot-fix list uses typed `text`/`replacement` string pairs, serialized as ordered single-entry objects under `input.hot_fix.pronunciation` or `input.hot_fix.replace`, never `input.replace`. Both keys are required. Timbrex locally rejects empty source text or invalid UTF-8 pair strings; it preserves literal whitespace, Unicode, empty replacement strings, duplicate source keys, and overlapping words. The documentation does not establish duplicate/overlap precedence or empty-replacement behavior: order preservation is not a promise of a matching algorithm. An explicit list `[]` is sent; omitted lists and an empty `hot_fix` mapping add no fields. See [the configuration example](../examples/qwen-synthesis.yaml).

### MiniMax

The [synchronous TTS API](https://platform.minimax.io/docs/api-reference/speech-t2a-http) supports top-level `text_normalization_mode`:

```yaml
providers:
  minimax:
    synthesis:
      text_normalization_mode: quality
```

Allowed values are `basic` and `quality`. `quality` requires non-streaming `speech-2.8-hd`, `speech-2.8-turbo`, `speech-2.6-hd`, or `speech-2.6-turbo`; Timbrex rejects it for other model IDs rather than accepting a silent downgrade. The current adapter is non-streaming. Omission leaves normalization disabled as before; no deprecated `voice_setting.text_normalization` field is added. See [the configuration example](../examples/minimax-synthesis.yaml).

Billing caveat: an existing designed/cloned voice can incur an additional voice fee on its first synthesis use, even though Timbrex creates no voice ([pricing](https://platform.minimax.io/docs/pricing/overview#audio)). For normalized text, do not estimate usage from the original script alone: the [response schema](https://platform.minimax.io/docs/api-reference/speech-t2a-http.md) distinguishes billable `usage_characters`, reference-only `word_count`, and high-quality normalization `norm_text_char_count` (also reference-only, not currently billed). Timbrex does not yet expose this usage metadata or `usage_voice_count`.

## Synthesis concurrency

`synth --concurrency N` controls the maximum number of in-flight speech segment jobs for real providers. It defaults to `1` (serial), accepts integers from `1` through `8`, and preserves script order in the output. `stub` only accepts `1`. This is deliberately a per-invocation CLI option, not a YAML key, front matter field, or environment variable; `timbrex config` does not display it. See [concurrent generation](providers.md#concurrent-generation) before increasing it.

## Inspect

`timbrex config` prints the merged defaults, file, and environment as YAML. Configured API keys are replaced by `REDACTED`. It does not read a script, apply `synth` flags, test credentials, or contact a provider. Deployment/dictionary identifiers, inline dictionary contents, and hot-fix text are not redacted; do not publish configuration containing private text. The output is not a credential backup. The repository ignores the root `timbrex.yaml`. Do not commit credentials.
