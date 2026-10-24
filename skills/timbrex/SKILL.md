---
name: timbrex
description: Write, edit, validate, configure, synthesize, and troubleshoot Timbrex dialogue and narration scripts (.tbx or .timbrex). Use for turning natural-language briefs into expressive speech scripts, adapting scripts to a TTS provider, and producing requested audio files. Not for implementing Timbrex itself.
---

# Timbrex

Turn the user's brief into a readable script, then carry out the requested parts of the script-to-audio workflow. Authoring does not require credentials or a chosen provider. Do not turn a script-writing request into a paid synthesis run.

## Start with the requested outcome

- Preserve intended language, wording, roles, tone, and delivery constraints. When editing, distinguish requested rewrites from changes to performance directions.
- Infer harmless creative details; ask only when a missing choice materially affects the result. If provider or voices are unspecified, finish a provider-neutral draft and identify what audio generation still needs.
- Use one explicit ASCII speaker ID per character, with display names in `cast`. Display names do not select voices or direct actors. Distinct speakers need distinct verified voice mappings if they should sound different.
- Keep directions sparse and playable: intention, emotional change, delivery, and meaningful pauses. Prefer local spans for a phrase; use scene state only when it should affect every following turn. Avoid contradictory or redundant directions.
- Write non-spoken production notes as standalone `//` comments. Parenthetical stage directions, Markdown headings, and inline comments are otherwise spoken text.

Read [references/authoring.md](references/authoring.md) when writing or repairing scripts. Start from [assets/dialogue.tbx](assets/dialogue.tbx) when helpful; [assets/scopes-and-escaping.tbx](assets/scopes-and-escaping.tbx) demonstrates non-obvious rules. These paths are relative to this skill, not the user's project.

## Find the available CLI

Use an existing trusted `timbrex` executable, or a known checkout's `./timbrex`. Inspect `timbrex version`, `timbrex --help`, and relevant command help before relying on flags. A source checkout's entry point is its root: `make build` or `go run .`, not `cmd/timbrex`. Read its `AGENTS.md` before building or editing there.

If the executable is missing, continue writing the script. Explain that validation was not run and offer repository build instructions; do not silently install tools globally, download an arbitrary binary, or claim a check passed. Build locally only when the task permits it and the required Go toolchain is available.

Examples use `timbrex` on PATH. Substitute the verified executable path, quote file paths, and run in the intended working directory.

## Write and check

1. Create UTF-8 `.tbx` files. Prefer `timbrex: 1` and `strict_tags: true` for new maintained scripts. Use explicit `[act "direction"]`, not invented tag names. Preserve intentional existing non-strict behavior unless asked to normalize it.
2. Check cast IDs, narration after paragraph boundaries, span closure, scene inheritance, resets, and literal punctuation. An ordinary blank line ends a turn.
3. Run the actual offline check:

   ```bash
   timbrex check dialogue.tbx --strict-tags
   ```

   Resolve errors, review warnings/information, and rerun after edits. An `ok` alongside warnings is not warning-free. For existing scripts intentionally using implicit acting, use their intended strict-mode setting rather than silently changing semantics.
4. Review spoken text separately from syntax. Avoid accidental narration, spoken production notes, excessive segmentation, and promising precise runtime from text alone.

`check` validates language, placement, vocabulary, and pause arguments. It does not select/preflight a provider, validate every numeric parameter, resolve voice IDs, or test sound quality.

## Configure and generate requested audio

Read [references/running.md](references/running.md) for provider selection, voice mapping, configuration, synthesis, or troubleshooting. Read only the selected provider's detailed documentation when needed. Check that its adapter and model support every audible control.

- Use voice/model IDs from user configuration, version-matched Timbrex documentation, or current official vendor documentation. Never invent IDs from character descriptions. A supported ID is not proof of this account's access.
- Keep request-scoped pronunciation dictionaries, feature flags, and custom deployment routing in the supported provider configuration, not invented DSL tags. Review [references/running.md](references/running.md) and the exact schema in the matching checkout.
- Keep API keys out of scripts, generated examples, chat, logs, and Git. Use existing environment/secret-manager setup. Do not print the environment or read secret values merely to check setup.
- Real-provider synthesis sends the script to that service and may cost money. Run it only within requested generation scope and approved spending constraints. If only writing/checking was requested, stop there. Do not use `synth` as a dry run: no public provider-only preflight or `--dry-run` exists.
- For a requested offline output smoke test, force `--provider stub`; never rely on configured defaults. Stub produces a tone, not speech, and ignores performance and pauses.
- Use a new destination unless replacement is intended. Its parent directory must exist. Default `.wav` needs no FFmpeg; `.mp3`, `.flac`, `.opus`, and `.aac` use a trusted local FFmpeg installation. Check the installed version and [running reference](references/running.md) before choosing another format; never auto-install tools. Start serial; increase `--concurrency` only when appropriate for the user's account and task.
- After success, verify the output exists and is nonempty. Listen if playback is available and the task needs auditory review; otherwise say it was generated but not auditioned. Re-running an uncertain or failed paid request can incur charges again.

## Report precisely

Deliver the script/file, meaningful creative choices, and exact validation performed. Separate:

- Language check: passed, failed, or not run.
- Provider compatibility: reviewed against documentation, locally rejected, or not established.
- Audio: not requested, stub tone, real synthesis completed, or failed; state whether it was listened to.

When blocked, preserve the usable script and identify the smallest missing input/setup. Offer an actionable adaptation for unsupported requirements instead of silently dropping them.

## Compatibility

Targets DSL `timbrex: 1` and the implementation from the same repository revision as this skill. The CLI reports `0.1.0`; that number alone does not establish matching behavior. Prefer installed-version help and documentation from the matching checkout or revision, and report uncertainty.
