# claudectx

**Switch between multiple Claude Code accounts, single command, and no re-logins.**

> macOS today; Linux is planned (see [architecture.md](architecture.md) →
> "Platform support").

A **profile** is its own `CLAUDE_CONFIG_DIR`. Claude Code keeps each profile's
credentials, history, and MCP config fully isolated (it even derives a distinct
Keychain slot per config dir), so accounts never collide — you can even run two at
once in different terminals. `claudectx use <name>` just launches `claude` with the
right `CLAUDE_CONFIG_DIR` set — no credential surgery.

**Agents, skills, and commands are shared** across all profiles via a common
`shared/` layer (symlinked into each profile), so you define them once.

> See [`architecture.md`](architecture.md) for the full design and rationale.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/RaphaelNeumann/claudectx/main/install.sh | sh
```

Installs the latest release to `~/.local/bin`. Override the location with
`BINDIR=/usr/local/bin`, or pin a version with `VERSION=v0.1.0`. macOS only for now
(Apple Silicon or Intel).

### From source

```sh
go install github.com/raphaelneumann/claudectx@latest
# or, in a clone:
go build -o claudectx .
```

## Usage

```sh
claudectx add personal        # create a profile
claudectx use personal        # launch claude in it (log in the first time)
claudectx add work
claudectx use work

claudectx                     # interactive picker
claudectx list                # list profiles (marks which are logged in)
claudectx current             # which profile this shell's CLAUDE_CONFIG_DIR points at
claudectx rename old new
claudectx remove work

claudectx use work -- -p "summarize the diff"   # forward args to claude

claudectx shared list         # what's in the shared agents/skills/commands
```

Because profiles are per-process, you can run **two accounts at once** in two
terminals.

### Shell shim (optional)

To set `CLAUDE_CONFIG_DIR` for the current shell (so a plain `claude` inherits it):

```sh
eval "$(claudectx shell-init)"   # in ~/.zshrc
claudectx-use work
```

## Layout

```
~/.config/claudectx/
  state.json
  shared/{agents,skills,commands}/   # shared by all profiles
  profiles/<name>/                   # == CLAUDE_CONFIG_DIR for <name>
    agents -> ../../shared/agents
    skills -> ../../shared/skills
    commands -> ../../shared/commands
    .claude.json, history, ...        # isolated, owned by Claude Code
```

## Status

Scaffolded, building, and the core model is **validated live** (2026-06-18):
per-profile credential isolation, the exact per-dir Keychain slot name, and
directory-level symlinks for the shared layer all confirmed. Remaining before v1:
`rename` credential behavior, a day-2 token-rotation persistence check, and release
packaging (see `architecture.md`).
