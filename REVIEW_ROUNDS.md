# Review rounds

> Historical M0 review notes. These are not current capability or validation claims; see [README](README.md) and [CONTRIBUTING](CONTRIBUTING.md).
> 历史 M0 审查记录，不代表当前能力或验证结果；现状见 [中文说明](README.zh-CN.md)和[贡献指南](CONTRIBUTING.zh-CN.md)。

Strict self-review of the M0 tip. P0 and P1 were fixed before the next round. P2 was recorded and left.

## Round 1

- Tip: `cc4f0c2`
- Angle: grammar and parser coverage against SPEC v0.3.4
- P0: 0. P1: 1
- Fixed: a quoted tag argument is opaque. `|`, `[`, and `]` inside quotes no longer split or close the tag. The `|` after the closing quote still starts span content. Regression: `TestQuotedArgHidesMarkup`.

## Round 2

- Tip: `437200a`
- Angle: error paths and partial or malformed input
- P0: 1. P1: 0
- Fixed: a backslash that is not an escape (`\`, `\q`) used to leave the scanner on the same byte, so `Parse` hung. It is now one literal text byte. A defined `\\` escape still yields one backslash. Regression: `TestLoneBackslashIsText` (timed out before the fix).

## Round 3

- Tip: `437200a` (no code change)
- Angle: CLI flags and exit codes
- P0: 0. P1: 0
- Checked: no-args help exits success; unknown command errors; `--strict-tags` with `--no-strict-tags` is rejected; a missing file errors; `synth --strict-tags` on an unregistered tag prints V041, returns `check failed`, and does not create the wav; a flag after the path still applies; `--version` exits success.
- P2, not fixed: `timbrex version` prints `timbrex 0.1.0`, while `--version` prints `timbrex version 0.1.0`.

## Round 4

- Tip: `437200a` (no code change)
- Angle: provider interface boundaries
- P0: 0. P1: 0
- Checked: `Synthesize` takes a context, a document, and an output path; unknown ids error; the stub writes a WAV only after `ctx.Err()` is nil; a failed check does not call the provider. One stub implementation is registered from `init`.
- P2, not fixed: the CLI passes `context.Background()`, and `Register` overwrites a duplicate id.

## Round 5

- Tip: `96129c9`
- Angle: vocabulary check against SPEC tag placement
- P0: 0. P1: 2
- Fixed: `overlap` and `gap` are rejected in narration, including when they lead the paragraph. A closing tag reports V013 only, so `[/pinyin]` does not also say that pinyin needs content. Regressions: `TestNarrationRejectsLeadingTurnTag`, `TestClosingWrapReportsOnlyV013`.

## Round 6

- Tip: `0f61a65`
- Angle: docs and code drift
- P0: 0. P1: 1
- Fixed: SPEC says `timbrex check` lists the speakers `cast: auto` recognized. It now emits one info per id and still exits successfully. Regression: `TestCastAutoListsSpeakers`. AGENTS.md now calls `.timbrex` an alias, matching the spec and the editor contribution.

## Round 7

- Tip: `0f61a65` (no code change)
- Angle: concurrency and cancellation
- P0: 0. P1: 0
- Checked: vocabulary and the provider registry are written from `init` before requests. `Parse` and `Apply` do not share mutable maps across calls. The stub checks `ctx.Err` before it writes a file.
- P2, not fixed: the CLI still passes `context.Background()`, and `Register` overwrites a duplicate id.

## Round 8

- Tip: `0f61a65` (no code change)
- Angle: module layout and readability
- P0: 0. P1: 0
- Checked: the path is lex, parse, check, lower, then a provider. The CLI does not reimplement the tag grammar. Go sources, comments, and CLI text stay English.

Rounds 7 and 8 were clean. Review continues through round 20.

## Round 9

- Tip: `0055d16`
- Angle: escape sequences and Unicode / CR-LF normalization in the lexer
- P0: 0. P1: 1
- Fixed: `]]` was always one literal `]`, so a nested tag that ended on the next `]` never closed (`[whisper|say [pause 1s]]`, `[whisper|[emph|hi]]`). `]]` stays a literal `]` in plain text and inside a single open tag. A deeper `]` closes that tag. A literal `]` inside the inner tag is `\]`. Regressions: `TestScanNestedCloseIsTwoBrackets`, `TestNestedSpansCloseTogether`.
- Checked: BOM strip and CR/CRLF folding happen before lex; full-width brackets and colons stay text; `\[ \] \| \\ \:` and `[[` still fold; a lone `\` is one byte; fence bodies keep escaped brackets raw.

## Round 10

- Tip: `ab8c4de`
- Angle: cast and speaker resolution edge cases
- P0: 0. P1: 2
- Fixed: front matter `narrator:` now sets the speaker of unprefixed paragraphs (default remains `narrator`). A narration paragraph that starts with a cast display name plus an ASCII or full-width colon stays narration and warns W103. The longest name wins; equal lengths use the smaller id. `cast: auto` still treats an ident prefix as a speaker, so that line is not W103. Regressions: `TestNarratorConfig`, `TestDisplayNamePrefixWarns`.
- P2, not fixed: a declared id followed by a full-width colon is narration with no W103. Leading spaces before `id:` stay narration, matching the turn production.

## Round 11

- Tip: `5e7974a`
- Angle: capability lowering against SPEC 3.2 and the capability notes
- P0: 0. P1: 2
- Fixed: `overlap` and `gap` were scene-scoped on a scene line and here-scoped inside a turn. They are turn attributes (the next turn, when alone on a line), so the scope is `turn`. Registered names are written in canonical form, so `[PAUSE]` lowers as `pause`. Regressions: `TestLowerTurnTagsAreTurnScope`, `TestLowerCanonicalName`.
- Checked: events stay `here` in a line and `scene` on a scene line; state uses the block scope; spans stay `span`; a scene-line `pause` is still an event on the scene timeline. Per-tag parameter schemas and provider capability plans stay out of M0.

## Round 12

- Tip: `5a3bd63`
- Angle: TextMate grammar against the parser's tokens
- P0: 0. P1: 2
- Fixed: a nested tag treated `]]` as one escape, so `[whisper|say [pause 1s]]` never closed in the highlighter. Inner tags now close on the next unescaped `]`. A `[` or `]` after an even run of backslashes is real, so `\\[pause]` is a tag, and `\[x\]` is not a second tag name. Regression: `TestGrammarNestedTagClose`, checked by tokenizing the grammar with vscode-textmate.
- Checked: quotes still hide `|` and brackets; `]]` inside one tag stays an escape; `10stone` and `3minutes` are one argument; fences swallow markup; front matter stays embedded YAML; `lin\:` and an indented `lin:` are not speakers.

## Round 13

- Tip: `ded8df9`
- Angle: CI, Makefile, AGENTS.md, and Go version consistency
- P0: 0. P1: 2
- Fixed: `go test` never compiled package main, and `golangci-lint fmt` was configured but not run. `make test` now builds `./timbrex` before the tests. `make lint` runs `golangci-lint run` and `golangci-lint fmt --diff`. CI calls `make test` and `make lint` with Go from `go.mod` (1.27) and golangci-lint v2.14.0. Regression: `TestCIMatchesMakefile`.
- Checked: `go mod tidy -diff` is clean. README and AGENTS name the same Go and linter versions.

## Round 14

- Tip: `992f5c1`
- Angle: error message quality and exit codes
- P0: 0. P1: 1
- Fixed: `main` prefixed every failure with `error:`, so a parse error was `error: scene.tbx:1:7: error[V012]: ...` and the location was no longer at the start of the line. Parse errors are printed as-is. `check failed` and other failures still use `error:`. Exit status stays 1 for any failure and 0 when the only findings are warnings or info. Regression: `TestFormatErrorKeepsParseLocation`.
- Checked: V012 after front matter is line 4 column 7; a tag after a non-ASCII character uses a rune column; an inner unclosed tag points at that tag. Wording that differs from the SPEC sample (`unclosed tag` versus `unclosed span`) is left as P2.

## Round 15

- Tip: `81e9ee8`
- Angle: fuzz and adversarial malformed scripts
- P0: 0. P1: 0
- Checked: `FuzzParse` ran about 124k inputs with no panic. Four thousand nested spans parse. A one-megabyte tag parses. Eighty thousand unclosed nested tags return V012. A modest YAML alias bomb returns immediately. Invalid UTF-8 does not spin. Regression: `FuzzParse`, `TestDeepNestAndHugeTagDoNotPanic`.
- P2, not fixed: millions of nested tags could still exhaust the goroutine stack. No depth cap was added, because a few thousand nests already succeed and the spec states no limit.

## Round 16

- Tip: `7abefed`
- Angle: synth temp files, partial WAV on failure, overwrite
- P0: 0. P1: 1
- Fixed: the stub used `os.WriteFile`, which truncates the destination before the bytes are written. A failed write left a partial RIFF header in place of a previous wav. It now writes a sibling temp file and renames it over the destination only after the write and close succeed. A failed write removes the temp file. Regressions: `TestOverwriteKeepsPreviousWavOnFailure`, `TestOverwriteReplacesWav`, `TestSynthCheckFailureWritesNoWav`.
- Checked: a failed check does not call the provider, so an existing wav is left unchanged.

## Round 17

- Tip: `2c45d2e`
- Angle: version text, cancelable CLI context, duplicate provider ids
- P0: 0. P1: 3
- Fixed: `timbrex version` and `timbrex --version` both print `timbrex 0.1.0`. `Run` uses `signal.NotifyContext` for interrupt and SIGTERM, and synth passes `cmd.Context()` into the provider. Tests call `run` with their own context so they do not install a process-wide handler. `Register` panics on an empty or duplicate id instead of overwriting. Regressions: `TestVersion`, `TestSynthCanceledContextWritesNoWav`, `TestRegisterDuplicatePanics`, `TestRegisterEmptyIDPanics`.

## Round 18

- Tip: `2c45d2e` (no code change)
- Angle: package API readability and dead code
- P0: 0. P1: 0
- Checked: `peekAhead` is used by blank-line skipping. Production functions are referenced. `Options.Path` is part of the provider contract; the stub does not read it. `Front.Lang` is kept on the front-matter struct and is not used by lowering. No unused code was removed.

## Round 19

- Tip: `7e5dd41`
- Angle: check then synth, strict tags with cast auto, bilingual editor install
- P0: 0. P1: 0
- Checked: `--strict-tags` with `cast: auto` lists each speaker once, rejects `[thoughtfully]` with V041, keeps `[act "quietly"]`, and `synth` writes no wav. Regression: `TestStrictCastAutoCheckThenSynth`. The English and Chinese README install steps match: `editors/vscode`, `cursor` or `code --install-extension .`, reload, language id `timbrex`, extensions `.tbx` and `.timbrex`.

## Round 20

- Tip: `05a19ec`
- Angle: final sweep against SPEC v0.3.4
- P0: 0. P1: 1
- Fixed: a fence info string must start with an ASCII identifier. A fence whose info is `123` is now a parse error. `ssml azure` is unchanged. Regression: `TestFenceInfoMustStartWithIdent`.
- Left out of M0, recorded earlier: `timbrex fix` / `fmt`, W101 W102 W104 I060, tbxset expand, `timbrex.yaml` loading, per-tag parameter schemas, provider namespaces, tree-sitter, and real provider APIs.
- P2 still open: an ASCII id plus a full-width colon is narration without W103; a speaker line with leading spaces stays narration; V012 text says `unclosed tag`; very deep nesting has no stack cap; the stub ignores `Options.Path`.

Rounds 9 through 20 are on main. `make lint` and `go test ./...` were green after each fix.
