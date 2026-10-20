# Review rounds

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

Stopped after rounds 3 and 4: two consecutive clean rounds, different angles, no P0 or P1.
