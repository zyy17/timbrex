# Configuration

[简体中文](configuration.zh-cn.md) · [Provider setup](providers.md)

Cobra handles every command. `internal/config` uses [koanf](https://github.com/knadh/koanf) to merge configuration and decode typed settings. The CLI loads them once; adapters receive resolved settings and never read environment variables or configuration files themselves.

## Choose a file

By default, `check`, `synth`, and `config` read `timbrex.yaml` from the **current working directory**, if present. They do not search the script's directory, parent directories, or your home directory. An explicit `--config` selects one file instead; a missing explicit file is an error.

```sh
cp examples/timbrex.yaml timbrex.yaml
./timbrex config
./timbrex --config examples/timbrex.yaml check examples/expressive.tbx
./timbrex synth examples/expressive.tbx --config timbrex.yaml --provider openai -o speech.wav
```

`--config` is a global flag and works before or after the command. Help, version, and shell completion do not load configuration. Timbrex does not load `.env` files, expand `${VARIABLE}` in YAML, or watch files for changes.

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
| `providers.<id>.api_key` | Credential for that provider; prefer its environment variable |
| `providers.<id>.model` | Default model; Azure selects models through its voice and rejects a model |
| `providers.<id>.voice` | Fallback voice for speakers without cast mappings |
| `providers.<id>.endpoint` | API base URL for OpenAI, ElevenLabs, and MiniMax; full request URL for Azure and Qwen |
| `providers.azure.region` | Azure region used when no endpoint is set |

Provider IDs are `stub`, `openai`, `elevenlabs`, `minimax`, `azure`, and `qwen`. Defaults and the supported environment variable names are listed in [provider setup](providers.md). The [complete example](../examples/timbrex.yaml) includes all five speech services. The stub ignores speech settings.

Endpoints must be HTTP(S) URLs without embedded credentials, query strings, or fragments. Credentials are sent to the configured endpoint; use a trusted service or gateway. Azure's explicit endpoint takes priority over its region. Checking scripts and displaying configuration do not require credentials; synthesis checks the selected provider's credentials and endpoint.

## Precedence

For each setting, the highest supplied value wins:

1. CLI flags (`--provider`, `--model`, `--voice`, strict-mode flags).
2. Script front matter (`provider`, `model`, `voice`, `strict_tags`).
3. Environment variables.
4. The selected YAML file.
5. Built-in defaults.

`TIMBREX_PROVIDER` overrides the file's default provider. `TIMBREX_STRICT_TAGS` overrides its strict mode and accepts Go boolean forms, including `true` and `false`. Existing vendor variables such as `OPENAI_API_KEY`, `OPENAI_MODEL`, and `OPENAI_BASE_URL` remain supported. Unset or empty environment variables leave lower-priority settings intact; unrelated variables are ignored.

A cast voice mapping still takes priority over the fallback voice, including `--voice`. A `voice` or `as` span overrides the current speaker within its scope. Model and voice IDs are not translated when switching providers. Endpoint and credential settings never come from scripts.

`--strict-tags=false` explicitly disables strict mode; `--no-strict-tags=false` enables it. The two flag names are mutually exclusive. This preserves the distinction between an omitted boolean and an explicit `false`.

## Inspect settings

`timbrex config` prints the merged defaults, file, and environment as YAML, with configured API keys replaced by `REDACTED`. It does not read a script, apply synthesis flags, test credentials, or contact a provider. Its output is for inspection, not a credential backup. The repository ignores the root `timbrex.yaml`; do not commit credentials in other files.
