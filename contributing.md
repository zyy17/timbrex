# Contributing to Timbrex

[简体中文](contributing.zh-cn.md)

Timbrex aims to be a small, understandable DSL for expressive TTS. Start with the [language reference](docs/spec.md) and [repository rules](AGENTS.md). The grammar is frozen; extend vocabulary and adapters without adding new syntax.

## Before opening a change

For a bug, include a minimal `.tbx` example, command, expected result, actual result, and tool version. For provider issues, include the model and region and redact credentials, private text, and signed audio URLs. API acceptance and audio quality are separate observations; describe both when relevant.

Discuss substantial syntax, architecture, or compatibility changes in an issue before implementing them. A focused bug fix or a documented provider addition can be proposed directly as a pull request.

## Development checks

Use the Go version in `go.mod` and golangci-lint **v2.14.0**. CI runs on Linux and macOS.

```sh
make test
make lint
make tidy
```

Tests must be offline by default. Use `httptest` for vendor contracts; never require credentials or spend money in the normal test suite. Add regression coverage for behavioral fixes and meaningful boundary cases. If changing the TextMate YAML source, regenerate its packaged JSON with:

```sh
go test ./editors/vscode -update
```

Live validation is separate and opt-in. Record the model, voice, region, request parameters, date, and result without recording secrets. Do not describe mock tests as proof that a vendor service works.

## Adding a provider

Read the official API documentation first and link it from the adapter and setup guide. Match the actual authentication, request schema, audio format, and completion/error responses. Put the implementation in `internal/providers/<name>`, register from `init`, and blank-import it in the CLI. Keep vendor code independent of parsing and Cobra.

Document supported models, environment variables, tag mappings, ranges, language hints, and limitations. Reject unrepresentable controls rather than dropping them. Test the full CLI-to-request path as well as malformed responses and failures. Shared code belongs in `internal/providers` only when it has a concrete shared use.

## Documentation and review

Use lowercase documentation filenames; `AGENTS.md` keeps its name for tool discovery. Update English and `.zh-cn.md` companions in the same change. Keep examples executable and distinguish implemented features, approximations, and plans. Cite primary sources for vendor-specific behavior; avoid unsupported price, licensing, quality, or deprecation claims.

Use short, idiomatic Go with clear package boundaries. Only `internal/cli` imports Cobra. Prefer existing patterns over new abstractions. Use English identifiers, errors, help text, comments, and Conventional Commit messages. Keep commits focused, include validation results, and never commit keys or `.env` files. Contributions are covered by the repository's [MIT license](license).
