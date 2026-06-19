# claudectx — Technical Architecture

> **Revised 2026-06-17.** This supersedes the original single-Keychain-slot design.
> A profile is now its own `CLAUDE_CONFIG_DIR`. See **Appendix A** for why the old
> "park/restore one slot + patch `~/.claude.json`" approach was dropped.

---

## What claudectx is

A CLI that manages multiple **isolated** Claude Code profiles (one per account /
subscription) on macOS and launches `claude` into the chosen one. Each profile keeps
its own credentials, history, MCP config, and tips; **agents, skills, and commands
are shared** across all profiles via a common layer.

Switching is launching `claude` with a per-profile `CLAUDE_CONFIG_DIR` — no
credential surgery, no global-state mutation.

---

## Core mechanism — `CLAUDE_CONFIG_DIR` per profile

Claude Code reads/writes its **entire** config tree from `$CLAUDE_CONFIG_DIR` when
that env var is set, instead of the default `~/.claude/` + `~/.claude.json`.

Proven on 2026-06-17 (CC v2.1.179): with `CLAUDE_CONFIG_DIR=/tmp/probe`, Claude Code
created `/tmp/probe/.claude.json` and `/tmp/probe/backups/`, and `claude mcp list`
saw none of the global config — full isolation.

### Credentials isolate automatically

The Keychain **service name is derived from the config dir.** From the decompiled
`Hv()` resolver:

| `CLAUDE_CONFIG_DIR` | Keychain service used |
|---------------------|-----------------------|
| unset (default)     | `Claude Code-credentials` |
| set to `<dir>`      | `Claude Code-credentials-<first 8 hex of sha256(dir)>` |

So **each profile gets its own credential slot for free.** claudectx never reads,
writes, parks, or copies credentials. Claude Code's own login flow writes the slot;
Claude Code's own token refresh rotates it in place. There is no shared slot to
contend over, therefore no re-capture invariant and no rotation-staleness risk.

> ✅ Confirmed live on 2026-06-18: logging a real account into a profile dir created
> exactly the predicted slot `Claude Code-credentials-<sha256(absDir)[:8]>` (the
> session required a fresh login; the default slot was untouched). `acct` = the
> macOS login name. See **Resolved validations**.

### What this buys us

- **Isolated** per profile: credentials, `~/.claude.json` state, project history,
  MCP servers, tips/onboarding, caches.
- **Shared** across profiles: agents, skills, commands (via the shared layer below).
- **Simultaneous** accounts: two profiles can run in two terminals at once.
- **No destructive failure modes**: claudectx never mutates a global file mid-flight
  and never deletes user content on a switch (the shared layer is static symlinks).

---

## Language & build constraints

- **Go 1.21+**, `CGO_ENABLED=0`, single static binary.
- No CGO; must cross-compile via `GOOS`/`GOARCH` without a C toolchain.
- Module path: `github.com/raphaelneumann/claudectx`

---

## Dependency decisions

| Concern | Package | Decision |
|---------|---------|----------|
| Commands | `spf13/cobra` | Subcommand routing |
| Interactive picker | `charmbracelet/huh` | Lightweight select; no full TUI |
| Output styling | `charmbracelet/lipgloss` | `list`/`current` views only |
| Config/state | stdlib `encoding/json` | Small; no viper |
| Process launch | `os/exec` + `syscall.Exec` | exec `claude` with env set |
| Testing | stdlib `testing` + `rogpeppe/go-internal/testscript` | Script-based CLI tests |
| Release | `goreleaser` | Cross-platform binaries + brew tap |

**No Keychain library and no `/usr/bin/security` calls.** claudectx does not touch
credentials at all — Claude Code owns them entirely. (This also sidesteps the
go-keyring base64 corruption hazard from the old design, which no longer applies.)

---

## Storage layout

```
~/.config/claudectx/
  state.json                       0644   # last-used profile (for the no-arg picker default)
  shared/                          0700   # one real copy, shared by ALL profiles
    agents/      <name>.md
    skills/      <name>/SKILL.md
    commands/    <name>.md
  profiles/
    <name>/                        0700   # == CLAUDE_CONFIG_DIR for this profile
      agents   -> ../../shared/agents      # symlink (dir), set at profile creation
      skills   -> ../../shared/skills      # symlink (dir)
      commands -> ../../shared/commands    # symlink (dir)
      .claude.json                         # CC-owned, isolated (created on first run)
      backups/  history.jsonl  projects/ … # CC-owned, isolated
```

claudectx owns `state.json`, `shared/`, and the **symlinks** inside each profile dir.
Everything else under `profiles/<name>/` is owned and written by Claude Code; claudectx
never edits those files.

No credential material on disk — credentials live only in each profile's Keychain
slot, managed by Claude Code.

### state.json schema

```json
{
  "lastUsed":  "<name or empty string>",
  "updatedAt": "<RFC3339>"
}
```

`lastUsed` is a convenience default for the interactive picker only. It is **not**
authoritative "active" state — multiple profiles can be active simultaneously in
different terminals, so there is no single global "active" profile.

---

## The shared layer (agents / skills / commands)

These are discrete files Claude Code reads but never rewrites, so they can be shared
safely by pointing each profile's `agents/`, `skills/`, `commands/` entries at the
single `shared/` copy via **directory symlinks** created when the profile is made.

- One edit in `shared/` is visible to every profile immediately. No sync, no
  per-switch materialization, no manifest, no deletion risk.
- Claude Code follows these symlinks — confirmed live 2026-06-18: a probe agent in
  `shared/agents/` appeared in a profile session's `/agents` through the symlinked
  `agents/` directory (see **Resolved validations**).
- Per-profile overrides (a profile that wants its *own* agent) are an explicit future
  extension: replace that profile's `agents` symlink with a real dir, or layer a
  per-profile dir ahead of the shared one. Out of scope for v1.

---

## `use` — switch algorithm (authoritative)

`use` does **not** mutate any global state. It resolves the profile and execs Claude
Code with the right environment.

```
1. resolve <name> → profiles/<name>/ must exist (else error: run `claudectx add`)
2. ensure shared-layer symlinks exist in profiles/<name>/ (self-heal if missing)
3. write state.json: lastUsed = <name>, updatedAt = now   (best-effort; non-fatal)
4. exec:  env CLAUDE_CONFIG_DIR=<abs path to profiles/<name>>  claude  [args…]
          via syscall.Exec so claudectx replaces itself with the claude process
```

Any extra args after `use <name>` are forwarded to `claude` verbatim
(`claudectx use work -- -p "summarize"` → `claude -p "summarize"`).

There is no partial-failure window: steps 1–3 are local and cheap, and step 4 is a
single `exec`. If login is required (fresh profile or expired refresh token), Claude
Code itself handles the OAuth flow inside that profile's own slot.

### Two scopes: terminal profile vs. default profile

There are two notions of "current", mirroring `nvm use` vs. `nvm alias default`:

- **Terminal profile** — `CLAUDE_CONFIG_DIR` exported in the shell. Set by
  `claudectx <name>` / the picker; reverted by `claudectx default`. Affects only the
  current terminal. **Requires the shell integration** (a binary cannot export to its
  parent shell).
- **Default profile** — `state.json` lastUsed. Set by `claudectx set default <name>`.
  Used by new terminals and any `claude` with no terminal override.

What a plain `claude` uses (resolved by the `claude` wrapper):

```
1. CLAUDE_CONFIG_DIR set in the shell  → use it (terminal override / explicit)
2. else  `claudectx _current-dir`      → the default profile (state.json lastUsed)
3. else  plain claude                   → default ~/.claude
```

`state.json` is a convenience default, not authoritative "active" state — multiple
profiles run at once across terminals, each via its own `CLAUDE_CONFIG_DIR`.

### Shell integration

`claudectx shell-init` (sourced from `~/.zshrc`/`~/.bashrc`) prints two POSIX
functions:

- a **`claude` wrapper** — runs the real `claude` with `CLAUDE_CONFIG_DIR` resolved
  as above (explicit value always wins).
- a **`claudectx` wrapper** — intercepts the terminal-switching forms and exports
  `CLAUDE_CONFIG_DIR`, delegating everything else to the binary:
  - `claudectx` / `pick` / `switch` → `_pick-dir` (picker, UI on stderr) → export
  - `claudectx <name>` → `_profile-dir <name>` → export
  - `claudectx default` / `reset` → unset (follow the default)
  - `add|remove|list|use|set|current|rename|shared|shell-init|completion|…` → binary

Function names contain no hyphens (valid in POSIX sh as well as bash/zsh). Without
the integration, the binary's `claudectx <name>` / picker print how to enable it;
`use` and `set default` work regardless.

`claudectx shell-init --install` appends the `eval` line to the shell rc file
(`~/.zshrc`/`~/.bashrc` from `$SHELL`, or `--rc <file>`), idempotently. fish is
unsupported (different function syntax).

---

## Planned commands

(Forms marked † are terminal-scoped and run through the shell-init `claudectx`
function.)

```
claudectx †                    picker → switch THIS terminal's profile
claudectx <name> †             switch this terminal to <name>
claudectx default †            revert this terminal to the default profile
claudectx pick | switch †      same as bare claudectx
claudectx set default <name>   change the DEFAULT profile (new terminals)
claudectx use <name> [args…]   switch AND exec claude (forwards args after --)
claudectx add <name>           create profiles/<name>/ + shared-layer symlinks
claudectx remove <name>        delete a profile dir (prompts; never touches shared/)
claudectx list                 list profiles, mark which have a credential slot
claudectx current              print the default + this terminal's profile
claudectx rename <old> <new>   rename a profile dir + its Keychain slot follows*
claudectx shared <cmd>         manage shared agents/skills/commands
claudectx shell-init [--install]  print/install the shell integration
```

\* **Rename caveat:** the Keychain slot name is `sha256(dir)`-derived, so renaming a
profile dir changes its slot and would orphan the old credential → forces re-login in
the renamed profile. `rename` must warn about this (or copy the slot, if we add the
one credential operation we'd otherwise avoid). Tracked in **Open question 2**.

---

## `add` — create a profile

```
1. mkdir -p profiles/<name>/ (0700)
2. ensure shared/{agents,skills,commands}/ exist (0700)
3. create relative symlinks:
     profiles/<name>/agents   -> ../../shared/agents
     profiles/<name>/skills   -> ../../shared/skills
     profiles/<name>/commands -> ../../shared/commands
4. print next step: `claudectx use <name>` then log in (CC drives OAuth)
```

No credential or `.claude.json` work — Claude Code creates and owns those on first run.

---

## Error states

| State | Detection | Recovery |
|-------|-----------|---------|
| `use <name>` but profile dir missing | stat `profiles/<name>` | error → `claudectx add <name>` |
| Shared-layer symlink missing/broken | readlink check in `use` | self-heal: recreate symlink, continue |
| `shared/` accidentally deleted | stat in `use`/`add` | recreate empty `shared/` dirs, warn |
| Profile never logged in | CC reports unauthenticated at launch | normal — CC drives the OAuth flow |
| Renamed profile forces re-login | slot name is dir-derived | warn on `rename` (see Open question 2) |
| `claude` not on PATH | `exec.LookPath("claude")` fails | error with install hint |

---

## Resolved validations (2026-06-18)

Confirmed live on CC v2.1.179 by logging a real account into a `personal` profile
while the default-dir session stayed active:

- ✅ **Credential isolation.** The profile session required a fresh login and wrote a
  new Keychain slot; the default `Claude Code-credentials` was untouched. Isolation
  is real, not just structural.
- ✅ **Slot-name formula.** The created slot was `Claude Code-credentials-2decd513`,
  matching `sha256(absProfileDir)[:8]` byte-for-byte (predicted == observed). The
  `SlotName()` implementation and the `list` "logged in" marker are correct.
- ✅ **Directory-level shared layer.** A probe agent dropped in `shared/agents/`
  appeared in the profile session's `/agents` list through the symlinked `agents/`
  *directory* — confirming CC honors dir-level symlinks for the shared layer.

## Open questions before v1 ship

1. **Switch-back / rotation persistence.** Confirm that leaving and returning to a
   profile — and crossing an ~8h token refresh — does NOT force a re-login. Expected
   to hold (each profile owns its slot; CC rotates in place), but not yet observed
   over time.

2. **Rename / credential portability.** ✅ **Decided: (a) warn + accept re-login.**
   Since the slot is `sha256(dir)`-derived, renaming a profile dir orphans its
   credential. Option (b) — copying the slot via `security` — was rejected: it would
   be the *only* place claudectx touches credentials, breaking the "Claude Code owns
   all credentials" invariant for a rare operation. `rename` warns; the user logs in
   once more in the renamed profile.

3. **`~/.claude.json` location & shared bits.** Confirm exactly what lives in
   `$CLAUDE_CONFIG_DIR/.claude.json` vs. elsewhere, and whether anything users expect
   to be shared (e.g. global settings.json) needs its own symlink in the shared layer.

---

## Platform support (current: macOS only; Linux & Windows = future)

v1 targets **macOS only**. Linux and Windows are deferred but feasible — the core
`CLAUDE_CONFIG_DIR` isolation is platform-agnostic; only credential detection, the
launcher, and the shared-layer symlinks have platform-specific concerns. Findings
below are from the CC v2.1.179 binary (same JS bundle ships on all platforms).

| Concern | macOS | Linux | Windows |
|---------|-------|-------|---------|
| Config-tree isolation (`CLAUDE_CONFIG_DIR`) | ✅ confirmed | ✅ same mechanism | ✅ same mechanism |
| Credential isolation | ✅ per-dir Keychain slot | ✅ `.credentials.json` lives **inside** the config dir → free, simpler than macOS | ⚠️ binary shows `DPAPI` / `Credential Manager`; unconfirmed whether per-dir |
| Launcher (`use`) | `syscall.Exec` | `syscall.Exec` works | ❌ `syscall.Exec` is a runtime stub (`EWINDOWS`) → needs spawn-and-wait |
| Shared-layer symlinks | ✅ | ✅ native | ⚠️ symlinks need admin/Dev-Mode → use junctions (`mklink /J`) or copy |
| `HasCredential` marker | `/usr/bin/security` | check `<profileDir>/.credentials.json` | check the Windows cred store / file |

Key discovery: CC has a file credential store at `Xn()/.credentials.json`, where
`Xn()` follows `CLAUDE_CONFIG_DIR` (unless `CLAUDE_SECURESTORAGE_CONFIG_DIR` is set).
No libsecret/gnome-keyring/secret-tool references exist in the binary, so Linux
almost certainly uses the plaintext `.credentials.json` file in the config dir.

### Linux plan (cheap — mostly done already)
- `syscall.Exec` and native symlinks already work; the binary cross-compiles today.
- Make `HasCredential` platform-aware (build-tagged `credential_darwin.go` /
  `credential_other.go`): on Linux, test for `<profileDir>/.credentials.json`.
- Add `linux/amd64` + `linux/arm64` to `.goreleaser.yaml`.
- Doc note: on Linux the credential is **plaintext on disk** (CC's own design —
  no guaranteed keyring), unlike the macOS Keychain.

### Windows plan (more work + unknowns)
- Launcher: `launch_windows.go` that spawns `claude`, inherits stdio, waits, and
  propagates the exit code (no `exec`).
- Shared layer: directory junctions (no privilege) or copy-on-create instead of
  symlinks.
- Config paths: use `os.UserConfigDir()` semantics; confirm CC's Windows config dir.
- **Open unknown:** confirm with a live test whether the Windows credential store
  (DPAPI / Credential Manager) isolates per `CLAUDE_CONFIG_DIR`. If it uses one
  global, fixed-named entry, isolation breaks and we fall back to
  `CLAUDE_SECURESTORAGE_CONFIG_DIR` per profile.

---

## Appendix A — why the original single-slot design was dropped

The first architecture assumed Claude Code had exactly **one** global credential slot
(`Claude Code-credentials`) and one global `~/.claude.json`, so switching required:

- parking inactive accounts under `claudectx:<name>` Keychain entries,
- copying to/from the single live slot on every switch,
- a "re-capture before switch" invariant to survive ~8h token rotation,
- surgically patching `oauthAccount`/`userID` into the shared `~/.claude.json`,
- guarding against `go-keyring` base64 corruption of the raw slot.

The 2026-06-17 discovery that **`CLAUDE_CONFIG_DIR` namespaces both the config tree
and the Keychain slot** removes the shared-resource contention those mechanisms
existed to manage. With one config dir per profile, Claude Code itself keeps each
account's credentials and state separate, and claudectx never needs to touch
credentials. The old approach is retained here only as rationale; it is not the
implementation.
