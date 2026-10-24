# Use Timbrex with an agent

[简体中文](agent-skill.zh-cn.md) · [Tutorial](tutorial.md)

The [Timbrex skill](../skills/timbrex/SKILL.md) teaches an agent one workflow: turn a brief into a script, revise dialogue and performance, check the language, choose compatible provider settings, generate requested audio, and troubleshoot failures. It is a portable instruction package, not a separate TTS service or an automatic installer.

## 1. Prepare the CLI

Writing a script needs no provider account. Checking it needs the Timbrex CLI. Use a trusted installed executable, or clone and build with Git, the Go version in [go.mod](../go.mod), and `make`:

```bash
git clone https://github.com/zyy17/timbrex.git
cd timbrex
make build
./timbrex version
```

If you already have a checkout, skip the first two commands and run the build from its root. Tell your agent where the executable or checkout lives if it is not on PATH. The skill does not automatically install Timbrex, Go, or provider credentials. An agent may build a trusted checkout when authorized. Without a CLI, it can still draft a script, but should say validation was not run.

## 2. Install and load the skill

Copy the entire `skills/timbrex` directory, including `references` and `assets`, into the target project's skill directory. Do not copy only `SKILL.md`.

- Codex CLI/IDE: `.agents/skills/timbrex/`.
- Claude Code: `.claude/skills/timbrex/`.
- Cursor: `.cursor/skills/timbrex/`, or reuse the Codex `.agents/skills/timbrex/` copy.

From the target project's root, replace the source path and choose one destination:

```bash
skill_src=/absolute/path/to/timbrex/skills/timbrex
skill_parent=.agents/skills
mkdir -p "$skill_parent"
test ! -e "$skill_parent/timbrex" && cp -R "$skill_src" "$skill_parent/timbrex"
```

Change `skill_parent` to `.claude/skills` or `.cursor/skills` as appropriate. If the destination exists, review and update that copy instead of overwriting it with this new-install command. The repository's distribution directory `skills/timbrex` is not itself a host's automatic discovery directory.

Open the target project in your agent. In agent chat, not the shell:

- Codex: `$timbrex` followed by your task, or choose it through `/skills`.
- Claude Code: `/timbrex` followed by your task.
- Cursor: type `/`, select `timbrex`, and describe your task.

If it is not listed, check the path and reopen/restart the agent. These are local project installs, not hosted-service or plugin installs. The content is portable; discovery, available tools, and permissions depend on the host. Installation references: [Codex](https://learn.chatgpt.com/docs/build-skills), [Claude Code](https://code.claude.com/docs/en/skills), [Cursor](https://cursor.com/docs/skills).

## 3. Ask for your first script

After invoking the skill, try:

```text
Write reunion.tbx: a short Chinese scene with a narrator, Lin, and Chen
meeting on an empty train platform. Make it restrained and warm, with a
few pauses and one whispered line. Use explicit cast IDs and strict tags.
Use the Timbrex CLI at /absolute/path/to/timbrex/timbrex to check it.
Only write and check the script; do not generate audio.
```

Replace the executable path with your actual path. Expect a `.tbx` file and an honest validation result, not invented voice IDs or a claim that the dialogue has been heard. Ask for an estimated length if helpful; exact duration depends on synthesis.

The bundled [dialogue example](../skills/timbrex/assets/dialogue.tbx) is also a starting point. The [scope and escaping example](../skills/timbrex/assets/scopes-and-escaping.tbx) shows nested controls, reset, literal brackets, and comments.

## 4. Check and smoke-test locally

To try the bundled example from the repository root:

```bash
./timbrex check skills/timbrex/assets/dialogue.tbx --strict-tags
mkdir -p out
./timbrex synth skills/timbrex/assets/dialogue.tbx --provider stub -o out/dialogue-tone.wav
```

The first command prints `ok` without contacting a speech service. The `synth` command writes a test tone, not dialogue. Stub does not reproduce voices, pauses, or directions. Use a fresh output path if that file already exists.

Passing `check` establishes language validity, not provider support. It does not validate every control's range, account access, or voice ID. There is no `--dry-run` or standalone provider-preflight command.

## 5. Generate speech when ready

Choose a [provider](providers.md), verify its current model availability and [supported controls](capabilities.md), and configure credentials privately through your environment or secret manager. Do not put keys in the script or chat. Timbrex does not automatically load `.env` files.

For multiple voices, ask the agent to prepare a working copy:

```text
Copy skills/timbrex/assets/dialogue.tbx to a new dialogue.tbx.
Use OpenAI with gpt-4o-mini-tts. Map narrator and lin to coral, and chen
to onyx under voices.openai. Check the script, but do not synthesize yet.
```

Those are baseline-supported IDs; verify current model availability and your account's voice access. Names and `cast: auto` do not choose distinct voices; `--voice` is only a fallback and does not override cast mappings.

The following example sends the script to OpenAI and may incur charges. Run it only after configuring access, checking the [model lifecycle notice](providers.md#openai), reviewing the script, and deciding to generate speech:

```bash
./timbrex synth dialogue.tbx --provider openai --model gpt-4o-mini-tts --voice coral -o out/dialogue.wav
```

The prepared copy gives Lin and Chen different voices; the untouched bundled example has no mappings and would use the same fallback for everyone. For another provider, use its model/voice IDs and compatible controls; Azure chooses its model through the voice and rejects `--model`.

You can instead ask the agent to do the authorized generation, naming provider, voices, output file, and any spending constraints. It should not make a paid request merely to test compatibility. After generation, check the actual audio; a valid audio file does not prove every word or direction sounds right. For optional MP3, FLAC, Opus, or AAC export and its local FFmpeg requirement, see [audio output](audio-output.md).

## 6. Revise and troubleshoot

Useful follow-up requests:

```text
Keep every spoken word, but make Chen less nervous. Avoid changing Lin's voice.
Explain why this reset restores the scene's calm mood instead of neutral.
Adapt this script to Gemini without silently dropping unsupported controls.
Diagnose this error without sending another paid request.
```

The skill keeps language checking, adapter compatibility, and audible quality separate. It should flag unsupported requests such as background music, overlapping speech, phoneme wrappers, or voice cloning, then offer practical alternatives. See the [tutorial](tutorial.md), [DSL specification](spec.md), and [provider guide](providers.md) for details.
