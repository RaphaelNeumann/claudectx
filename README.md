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

## Setup

Install the shell integration once — it makes `claudectx <name>` switch *this
terminal* and a plain `claude` follow the chosen profile:

```sh
claudectx shell-init --install   # appends to ~/.zshrc or ~/.bashrc (idempotent)
```

Then open a new shell (or `source` the rc file).

## Usage

```sh
claudectx add personal        # create a profile
claudectx add work

claudectx work                # switch THIS terminal to work
claude                        # ...runs in work
claudectx                     # picker: switch this terminal interactively
claudectx default             # revert this terminal to the default profile
claudectx current             # show the default + this terminal's profile

claudectx set default work    # change the DEFAULT (new terminals / plain claude)

claudectx use work            # switch and launch claude immediately
claudectx use work -- -p "summarize the diff"   # ...forwarding args to claude

claudectx list                # list profiles (marks which are logged in)
claudectx rename old new
claudectx remove work
claudectx shared list         # what's in the shared agents/skills/commands
```

### Two scopes: this terminal vs. the default

- **`claudectx <name>`** (or bare `claudectx` for a picker) switches the profile for
  **the current terminal only** — like `nvm use`. `claudectx default` reverts it.
- **`claudectx set default <name>`** changes the **default** profile that new
  terminals (and a plain `claude` with no override) use — like `nvm alias default`.
- A plain **`claude`** runs in this terminal's profile if set, otherwise the default.
  An explicit `CLAUDE_CONFIG_DIR` in the shell always wins.

Because each terminal has its own profile, you can run **two accounts at once** in
two terminals. Terminal switching requires the shell integration above (a subprocess
can't change its parent shell's environment); without it, `claudectx <name>` prints
how to enable it, and `claudectx use <name>` / `set default` still work.

### Shell completion

```sh
claudectx completion install         # install for your current shell ($SHELL)
claudectx completion zsh --install   # ...or target a specific shell
claudectx completion zsh             # or just print the script to stdout
```

Supports bash, zsh, and fish (`--dir` overrides the install location). The command
prints any follow-up step needed to activate it.

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
