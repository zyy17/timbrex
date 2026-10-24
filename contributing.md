# Contributing to Timbrex

[简体中文](contributing.zh-cn.md)

This guide covers local development, tests, and pull requests. Read the [tutorial](docs/tutorial.md) to try the language, the [DSL Specification v1](docs/spec.md) for its rules, and [AGENTS.md](AGENTS.md) for package boundaries. The language is still developing; proposed rule changes should explain their effect on existing scripts.

## Set up a checkout

Install the Go version declared in [go.mod](go.mod), golangci-lint **v2.14.0**, and Node.js **24**, matching CI. Node.js is needed only for the editor runtime tests. Fork the repository if you plan to submit a pull request, then clone your fork. To explore the upstream checkout:

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
go mod download
make editor-deps
make build
./timbrex check examples/hello.tbx
./timbrex synth examples/hello.tbx --provider stub -o hello.wav
```

`check` prints `ok` for this example. The `stub` provider writes a short test tone without credentials or a network request. It exercises the output path, not speech quality. The entry point is repository-root `main.go`; you can also run `go run . check examples/hello.tbx`.

## Find the code to change

The [architecture guide](docs/architecture.md) follows one script through the compiler, request planning, and WAV output, with diagrams and source links.

The main path is `internal/lex` → `internal/parse` and `internal/ast` → `internal/check` → `internal/lower` and `internal/ir` → `internal/providers`. The standard tags are in `internal/vocab`. Command wiring is in `internal/cli`; YAML settings and environment bindings are in `internal/config`. Tests live beside the code they exercise.

Source-based lowering tests and benchmarks must follow `parse.Parse` → `check.Apply` → `check.Failed` → `lower.Script`, lowering only after checking succeeds. Build IR directly for planner tests that intentionally exercise invalid input. In IR, utterance speakers distinguish scene lines from speech, node kinds distinguish state from events, and paired span markers delimit local scope; there is no separate scope field.

Use short, idiomatic Go functions and follow existing patterns. Keep parsing independent of providers and I/O. Only `internal/cli` imports Cobra. Adapters receive resolved configuration and IR; they do not reload settings or parse scripts. Identifiers, errors, help text, and code comments use English. Comments should explain a constraint or a non-obvious decision rather than restate the code.

For a bug report, include a minimal `.tbx` script, the command, expected and actual results, and `timbrex version` output. Provider reports also need the model and region; remove credentials, private text, and signed audio URLs. Discuss substantial syntax, architecture, or compatibility changes in an issue before implementing them. A focused fix or a documented provider addition can go directly to a pull request.

## Run the checks

Run a relevant package while iterating, for example `go test ./internal/parse`. Before submitting, run the same checks as CI:

```bash
make verify
```

`make verify` runs `make tidy`, `make test`, `make lint`, and `make test-editor`. `make tidy` reports whether `go.mod` or `go.sum` would change without rewriting them. `make test` builds `./timbrex` and runs `go test ./...`. `make lint` runs golangci-lint and its formatter check. `make test-editor` runs the actual TextMate/Oniguruma tokenizer tests. Use `make fmt` to apply Go formatting. CI runs the Go test suite on Linux and macOS and the editor runtime tests on Linux.

Install dependencies during setup with `go mod download` and `make editor-deps`. Verification does not run `npm ci` or contact TTS services; with the Go modules and tools already cached, it can run offline. Repeat dependency installation only when dependencies change.

To collect Go coverage without skipping subprocess tests, run `go test -count=1 -coverprofile=coverage.out ./...`. The CLI input helper inherits an allowlist of runtime variables, including `GOCOVERDIR`, but no provider credentials or configuration. Go collects its child counters in the parent profile. Separately built CLI binaries, editor runtime tests, and live service behavior are not included; statement coverage does not establish branch coverage or audio quality.

Tests must be offline by default. Use `httptest` for vendor requests, and add regression coverage for behavioral fixes and boundary cases. Normal tests must not require credentials or spend money. If you edit the TextMate YAML grammar, regenerate its packaged JSON with `go test ./editors/vscode -update`; see the [editor guide](editors/vscode/readme.md).

Live validation is separate and opt-in. Record the model, voice, region, request parameters, date, and result without secrets. Distinguish API acceptance from the quality of the resulting audio; mocked requests establish neither.

## Add a provider

Read the official API documentation first and link it from the adapter and [setup guide](docs/providers.md). Match its authentication, request schema, audio format, and completion/error responses. Implement `Provider` in `internal/providers/<name>`, register it from `init`, and blank-import it in the CLI. Add defaults and environment bindings in `internal/config`.

Document supported models, credentials, tag mappings, ranges, language hints, and limitations in both the setup guide and [capability reference](docs/capabilities.md). Reject controls the adapter cannot represent. Test the full CLI-to-request path, malformed responses, and failure handling. Move code into `internal/providers` only when it has a concrete shared use.

## Write and review documentation

Keep each page focused: README for an overview and first run, tutorial for a guided script, specification for language rules, configuration and provider pages for setup, and design notes for tradeoffs. Link to the detailed reference instead of repeating it.

Write for a developer encountering the project for the first time. Explain what a feature does, show a small example, and state relevant limits where they affect its use. Avoid slogans, claims without evidence, and promises about future compatibility. Chinese text should read naturally rather than follow English sentence structure; keep identifiers and useful technical terms in English. Distinguish implemented behavior, approximations, and proposals. Cite official sources for vendor behavior.

Update English and `.zh-cn.md` companions together, with the same examples and technical meaning. Use lowercase documentation filenames except `AGENTS.md`. Documentation tests parse every fenced `tbx` example and `.tbx` file, check local file links and paired headings, and check Markdown fences, table escaping, and strong-label boundaries. Preview changed pages with a GitHub-compatible renderer as well, including their diagrams; syntax checks alone do not establish readability. These checks do not prove provider support or audio quality; verify those claims against adapter code and tests.

## Open a pull request

Keep the change focused and use English [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) messages, such as `fix: preserve speaker after nested span`. Explain the problem, the resulting behavior, and which checks passed. Include a small before/after example when it helps, and call out compatibility changes or missing validation. Do not commit keys, `.env` files, or generated audio. Contributions are covered by the repository's [Apache License 2.0](license).
