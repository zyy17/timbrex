# Review rounds

Earlier rounds 1–20 lived on main and were removed in `c18cb3e` as stale notes. This file records an independent review of current `main`. Those earlier conclusions were not treated as evidence.

## Independent review

- Tip before fixes: `6153f02` (origin/main, including merged PRs 1–9 and 11–14)
- Open drafts inspected: PR 10 (Azure 10-minute PCM), PR 15 (Fish Audio)
- Baseline: `make lint` clean; `go test ./...` and `go test -race ./...` clean on `6153f02` after installing Go 1.27.3 and golangci-lint v2.14.0
- Angles: SPEC v0.3.4 / `timbrex: 1` conformance, lexer and parser edges, diagnostics and exit codes, provider contracts versus official docs, readability, test quality, TextMate drift, CI and docs drift
- Method: read the tree and every PR; confirm each suspicion with a throwaway or failing probe. Previous self-review claims were re-checked from code

### Pull requests

| PR | State | Independent read |
| --- | --- | --- |
| 1 | Merged | W040 / I050 / V041, quoted args, lone backslash hang. Those behaviors remain on main and still match the spec |
| 2–8 | Merged | Escaped arguments, linear lexer/lower, audio validation, Azure pitch, config redaction, ElevenLabs model controls. Still present |
| 9 | Merged | Azure locale prefix and inner `<lang>` overrides. Tests cover custom multi-hyphen voices |
| 10 | Draft | Correct fix for Azure REST 10-minute truncation. Reproduced from official docs and implemented independently on this branch |
| 11–13 | Merged | Shared `Prepare`, WAV decoder, contract docs |
| 14 | Merged | Gemini `generateContent` adapter. Offline contract matches the GenerateContent `speechMetadata` / `AUDIO_WAV` schema. The current TTS guide leads with the Interactions API; both paths are still documented |
| 15 | Draft | Fish Audio adapter stacked on the WAV decoder. Not merged here |

### Findings

| Sev | Area | What | Probe | Result |
| --- | --- | --- | --- | --- |
| P1 | parse | Opening `--- ` / `---\t` is not recognized as front matter. Cast, provider, and strict mode are silently ignored; `check` still prints `ok`. TextMate already treated those lines as YAML | `Parse("--- \ncast: auto\n---\nlin: hi\n")` → narration containing the YAML | Fixed |
| P1 | plan | `[as ghost\|Hello]` with no such speaker used the default voice | `Plan` returned `voice=fallback` and no error | Fixed |
| P1 | azure | Official REST docs: audio longer than 10 minutes is truncated, possibly with HTTP 200. The adapter treated any successful PCM as complete | `azurePCMComplete` / synth mock at 28,800,000 bytes | Fixed |
| P2 | docs | Capabilities comparison table omitted Gemini; AGENTS.md still listed five HTTP providers; market survey said “all five” | File read vs `internal/providers/gemini` | Fixed |
| P2 | cli | `TestMain` unset vendor env prefixes but not `GEMINI_*`, so a runner key can leak into CLI tests | `cli_test.go` vs `gemini_test.go` | Fixed |
| P2 | editor | Named-argument highlighting allowed a leading `_`; the parser ident does not | Grammar `tag-arg` vs `identLen` | Fixed |
| P2 | http | Truncated bodies are already rejected when `Content-Length` is set (`unexpected EOF`). Bodies without a length cannot prove completeness | httptest probes | Not a bug; left as a documented PCM limit |
| P2 | check | `[pause 0]` is a number; `check` accepts it; `Plan` rejects it. Spec says per-tag types are not checked yet | parse + plan probe | Open |
| P2 | providers | Empty default voice still plans; OpenAI/Azure/MiniMax/Qwen can POST `voice=""` if config clears the default | `Plan(doc, "openai", "")` | Open |
| P2 | gemini | Official TTS guide now leads with `POST /v1beta/interactions`. GenerateContent remains documented. Lite model `gemini-3.8-flash-lite-tts` is rejected | Docs fetch 2026-10-22 | Open; see design notes |
| P2 | azure | PR 10’s exact-boundary reject also refuses a rare genuine 10:00.000 clip | Official wording “can’t exceed 10 minutes” | Accepted; documented |
| Nit | parse | `cast: Auto` is a hard error; `.5s` is a word | Spec examples use `auto` and `0.7` | Open |
| Nit | check | `[/hello world]` is implicit `act`, not V013 | `closing()` rejects spaces | Open |

No P0. Lexer escapes, nested `]]`, speaker column zero, scene lines, fence info, BOM/CRLF, and registry duplicate panics were re-checked and match the spec or existing tests.

### Design notes before more real providers

1. Keep `Provider` as unary `Synthesize(ctx, doc, Options)`. Add optional interfaces later (`Preflight`, stream) instead of growing `Options`.
2. Fail before the first HTTP call. `Prepare` is the right seam. Unknown `[as]` targets and Azure’s 10-minute cap belong there or immediately after the response, not in the renderer.
3. Do not guess request shapes. Official Gemini TTS examples now use Interactions; if that becomes the only supported path, replace the adapter rather than layering a second schema.
4. `check` is language-only by spec. A later `check --provider` / dry-run that runs `Prepare` without I/O would catch adapter errors earlier without changing the grammar.
5. Raw PCM cannot prove sample rate or that every word was spoken. Keep container/WAV checks strict; document the rest.
6. No retries, no streaming, no voice listing in v0.1. Surface `429` as a hard error.
