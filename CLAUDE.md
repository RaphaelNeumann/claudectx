# claudectx — Project Context

## What this is

A CLI tool to switch between multiple Claude Code accounts/subscriptions on macOS
without re-authenticating. One command swaps the active account in ~300ms.

## Current status (architecture revised 2026-06-17)

**Architecture was redesigned on 2026-06-17.** The original single-Keychain-slot
"park/restore + patch `~/.claude.json`" model was dropped after discovering that
`CLAUDE_CONFIG_DIR` gives each profile its own isolated config tree **and** its own
auto-namespaced Keychain slot. A profile is now just its own `CLAUDE_CONFIG_DIR`,
and claudectx is a launcher — it never touches credentials.

Read `architecture.md` (authoritative, fully revised). The "Critical technical
decisions" section below is **superseded** and kept only as historical rationale;
see `architecture.md` Appendix A.

Chosen goal: **isolated accounts/history, shared agents/skills/commands.**
**Next step: scaffold the Go project around the env-var launcher model.**

---

## Key files

| Path | Purpose |
|------|---------|
| `claudectx/architecture.md` | Authoritative technical spec — read this first |
| `claudectx-poc/main.go` | Go PoC that proved credential read/write works |
| `claudectx-poc/cctx.sh` | Shell PoC used to validate real account switching |
| `~/.claudectx-poc/` | Snapshots from the PoC test (personal + work) — safe to delete |
| `~/claudectx-poc-backup.json` | Raw credential backup from PoC — safe to delete |

---

## Critical technical decisions — REVISED MODEL (2026-06-17)

The current design is the `CLAUDE_CONFIG_DIR`-per-profile model. Key points:

1. **A profile = its own `CLAUDE_CONFIG_DIR`.** CC reads/writes its whole config
   tree (incl. `.claude.json`, history, MCP) from there. Proven empirically.
2. **Credentials isolate for free.** CC derives the Keychain slot name from the
   config dir: `Claude Code-credentials-<sha256(dir)[:8]>`. Each profile gets its
   own slot; claudectx never touches credentials.
3. **`use` is a launcher**, not a mutator: `exec env CLAUDE_CONFIG_DIR=… claude`.
   No global state changes, so simultaneous multi-account is possible.
4. **Shared layer = static directory symlinks.** `profiles/<name>/{agents,skills,
   commands}` symlink to a single `shared/` copy, set at profile creation. No
   per-switch mutation → no deletion risk.

See `architecture.md` for the authoritative spec and Appendix A for why the
original design (below) was abandoned.

---

## Original critical technical decisions — SUPERSEDED (kept for history)

> ⚠️ The four decisions below describe the dropped single-slot design. They are
> **no longer how claudectx works.** Do not implement against them.

### 1. Use `/usr/bin/security` CLI, never `go-keyring`

`go-keyring` encodes stored values as `go-keyring-base64:<b64>`. Claude Code reads
the Keychain slot raw and expects plain JSON — the encoded form causes a parse error
and logs the user out. **This was discovered by breaking and repairing the live slot
during testing on 2026-06-16.** Recovery was done via `security add-generic-password -U`.

### 2. Only two fields are patched in `~/.claude.json`

`oauthAccount` and `userID`. Everything else (caches, project history, tips) is
account-agnostic and must be preserved. Swap the whole file = destroy user state.

### 3. Re-capture before switch (the critical invariant)

Claude Code silently rotates tokens every ~8h, rewriting the live Keychain slot.
Before switching away from a context, always re-capture the live slot back into
that context's parked Keychain entry (`claudectx:<name>`). Skipping this will leave
a dead `refreshToken` in the parked snapshot → forced re-login on next switch.

### 4. Single Keychain slot, one active account at a time

`Claude Code-credentials` is a single slot. The tool parks inactive accounts under
`claudectx:<name>` and copies to/from `Claude Code-credentials` on each switch.
Simultaneous multi-account is not possible with Claude Code's design.

---

## Validated behavior (live test with two real accounts)

Tested on 2026-06-16 using:
- **personal**: raphael@rneumann.me — `sub=max`, `tier=default_claude_max_5x`
- **work**: raphael.neumann@spok.com — `sub=team`, `tier=default_raven`, org=Spok

Results:
- ✅ personal → work, no re-login required
- ✅ work → personal, no re-login required
- ✅ Token fingerprints restored byte-identical after each switch
- ✅ Zero Keychain prompts throughout (unsigned binary)
- ✅ Identity (`oauthAccount`, org, role, subscriptionType) swaps correctly

---

## Chosen stack

- **Go 1.21+**, `CGO_ENABLED=0`, single static binary
- `spf13/cobra` — commands
- `charmbracelet/huh` — interactive account picker
- `charmbracelet/lipgloss` — output styling
- `os/exec` + `syscall.Exec` — launch `claude` with `CLAUDE_CONFIG_DIR` set
- stdlib `encoding/json` — `state.json` only (no credential/identity files)
- **No Keychain code** — Claude Code owns credentials per profile dir
- `goreleaser` — release + brew tap

---

## Planned commands

```
claudectx †                     picker → switch THIS terminal's profile
claudectx <name> †              switch this terminal to <name>
claudectx default †             revert this terminal to the default profile
claudectx pick | switch †       same as bare claudectx
claudectx set default <name>    change the DEFAULT profile (new terminals)
claudectx use <name> [args…]    switch AND exec claude (forwards args after --)
claudectx add <name>            create profile dir + shared-layer symlinks
claudectx remove <name>         delete a profile dir (never touches shared/)
claudectx list                  list profiles, mark which have a credential slot
claudectx current               print the default + this terminal's profile
claudectx rename <old> <new>    rename a profile (warns: slot is dir-derived → re-login)
claudectx envs [name]           show the profile's env vars (loaded into claude)
claudectx envs [name] --edit    edit the profile's env file in $EDITOR
claudectx shared <cmd>          manage shared agents/skills/commands
claudectx shell-init [--install]  print/install the shell integration
```

† = terminal-scoped; runs through the shell-init `claudectx` function (a binary
can't export to its parent shell). **Two scopes** (like nvm use vs nvm alias
default): the **terminal profile** is `CLAUDE_CONFIG_DIR` exported by the function;
the **default profile** is `state.json` lastUsed (set via `set default`). `claude`
uses the terminal override if set, else the default. (No `refresh` command — token
rotation is handled by Claude Code inside each profile's own slot.)

---

## Storage layout (target — REVISED)

```
~/.config/claudectx/
  state.json                 { "lastUsed": "<name>", "updatedAt": "<RFC3339>" }  # picker default only
  shared/                    # one real copy, shared by ALL profiles
    agents/  skills/  commands/
  profiles/
    <name>/                  # == CLAUDE_CONFIG_DIR for this profile
      agents   -> ../../shared/agents     # static dir symlinks (set at `add`)
      skills   -> ../../shared/skills
      commands -> ../../shared/commands
      claudectx.env                       # optional: per-profile env for `claude` (claudectx-owned)
      .claude.json  backups/  history.jsonl  projects/ …   # CC-owned, isolated

Keychain (managed entirely by Claude Code, not claudectx):
  "Claude Code-credentials-<sha256(profileDir)[:8]>"   # per-profile slot, auto-namespaced
```

No `claudectx:<name>` parked entries, no shared live slot, no `identity.json`.
See `architecture.md` for the full layout.

---

## Open questions before v1 ship

Authoritative list lives in `architecture.md` ("Open questions"). Summary:

1. **Switch-back / rotation persistence** — confirm leaving and returning to a
   profile (and crossing an ~8h token refresh) never forces a re-login. Expected to
   hold since each profile owns its slot; not yet observed over time.
2. **Rename / credential portability** — decided: warn-and-relogin (the slot is
   `sha256(dir)`-derived, so a rename orphans the credential).
3. **`$CLAUDE_CONFIG_DIR/.claude.json` contents** — ✅ **resolved (2026-06-19):**
   keep the whole tree isolated; `.claude.json` holds only per-profile data
   (identity, machine/install state, caches, onboarding, path-keyed `projects`).
   `settings.json` stays per-profile (NOT shared) — `model` is profile-specific and
   CC rewrites the file at runtime via `/model` /`/config`, so a symlink would leak
   changes across profiles. Power-user escape hatch: a hand-made `settings.json`
   symlink survives `EnsureSymlinks` (real entries = intentional overrides). See
   `architecture.md` Open question 3.

(The pre-redesign questions — refresh-token rotation under the single shared slot,
Keychain ACL after signing, `~/.claude.json` write race — are obsolete: there is no
shared slot and claudectx no longer writes `~/.claude.json`.)

---

## Progress / what to do next

**Done (2026-06-17 → 18):**
- Go project scaffolded: `cmd/` (use, add, remove, list, current, rename, shared,
  shell-init, hidden _profile-dir) + `internal/{paths,store,profile,launch}`.
  Builds clean, `go vet` clean, `internal/profile` has passing tests.
- **Core model validated live** (see `architecture.md` "Resolved validations"):
  credential isolation confirmed (fresh login required, separate Keychain slot),
  the `SlotName()` hash matched the real slot byte-for-byte, and a shared probe
  agent appeared in a profile session's `/agents` (dir-level symlinks work).
- A real `personal` profile exists and is logged in.

- **`rename` decision made** (Open question 2): warn-and-relogin, to keep the
  "Claude Code owns all credentials" invariant. Implemented + documented.
- **Release tooling + tests added:** `.goreleaser.yaml` (darwin amd64/arm64, brew
  tap `RaphaelNeumann/homebrew-tap`), `Makefile`, version injection via
  `-ldflags -X .../cmd.version`, and testscript CLI integration tests in
  `cmd/testdata/script/` (basics + launch). All green; gofmt clean.

**CI/release (added 2026-06-18):** `.github/workflows/ci.yml` (PRs + branch pushes:
gofmt/vet/test/build) and `release.yml` (push to main: test → derive semver from
Conventional Commits via `mathieudutour/github-tag-action` → tag → GoReleaser
release). `.goreleaser.yaml` uses `homebrew_casks` (skipped unless
`HOMEBREW_TAP_GITHUB_TOKEN` secret is set). Validated locally: `goreleaser check` +
full `release --snapshot` produce the darwin amd64/arm64 archives + checksums. First
release fires on the first `feat:`/`fix:` commit to main (→ v0.1.0 for `feat:`).

**Homebrew cask — DONE (2026-06-19):** `RaphaelNeumann/homebrew-tap` exists with
`Casks/claudectx.rb` published, the `HOMEBREW_TAP_GITHUB_TOKEN` secret is set, and
releases are flowing (v0.1.2 shipped 2026-06-19). Commits `cdea032`/`3249f96`/`04ed995`
iterated the cask (valid env template → quarantine strip → single fqn install cmd).

**Per-profile env — DONE (2026-06-19):** profiles can carry a `claudectx.env`
(`KEY=VALUE` dotenv) that the launcher loads into the **claude process only** (not the
shell) on both paths — `use` (`launch.Exec`) and the `claude` wrapper via hidden
`_exec-claude` (`launch.ExecDir`). Managed with `claudectx envs [name]` / `--edit`.
Motivating case: a work profile on Google Vertex (`CLAUDE_CODE_USE_VERTEX`, …);
generalizes to Bedrock/proxy. Provider-side credential isolation (gcloud/ADC) is out of
scope. Tests: `internal/profile/env_test.go` + `cmd/testdata/script/{launch,envs}.txtar`.

**Open question 3 — RESOLVED (2026-06-19):** whole config tree stays isolated;
`settings.json` stays per-profile (not shared). Audit + rationale in `architecture.md`
Open question 3. Document-only; no code change.

**Next:**
1. Day-2 persistence test (Open question 1): switch away/back + cross an ~8h token
   refresh without a forced re-login. (Only confirmable over time — the last
   genuinely-open item before v1.)

**Future (deferred):** Linux & Windows support — v1 is macOS-only. Plan + binary
findings captured in `architecture.md` ("Platform support"). Linux is cheap (CC
stores `.credentials.json` inside the config dir; `syscall.Exec` + symlinks already
work); Windows needs a spawn-based launcher, junctions/copy for the shared layer,
and a live test of credential isolation.
