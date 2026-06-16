# claudectx — Project Context

## What this is

A CLI tool to switch between multiple Claude Code accounts/subscriptions on macOS
without re-authenticating. One command swaps the active account in ~300ms.

## Current status (paused 2026-06-16)

Architecture defined. PoC validated. Not yet scaffolded as a real Go project.
**Next step: scaffold the Go project structure and start implementing.**

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

## Critical technical decisions (proven by testing, not just theory)

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
- `os/exec` → `/usr/bin/security` — Keychain (raw, no library)
- stdlib `encoding/json` — config/identity files
- `goreleaser` — release + brew tap

---

## Planned commands

```
claudectx                     interactive picker
claudectx use <name>          switch to context (core command)
claudectx add <name>          snapshot current session as <name>
claudectx remove <name>       delete a saved context
claudectx list                list contexts, mark active
claudectx current             print active context + account info
claudectx rename <old> <new>  rename a context
claudectx refresh             manually re-capture active context tokens
```

---

## Storage layout (target)

```
~/.config/claudectx/
  state.json          { "active": "<name>", "updatedAt": "<RFC3339>" }
  contexts/
    <name>/
      identity.json   { "oauthAccount": {...}, "userID": "..." }

Keychain:
  "claudectx:<name>"  parked credential per context (raw JSON, no encoding)
  "Claude Code-credentials"  live slot owned by Claude Code
```

---

## Open questions before v1 ship

1. **Refresh token rotation** — does `refreshToken` rotate on every `accessToken`
   refresh? Observed stable in one session; confirm over 24h. Critical for
   understanding the blast radius of a missed re-capture.
2. **Keychain ACL after code signing** — unsigned binary had zero prompts. Re-test
   after signing; may need an entitlement or a one-time `security unlock-keychain`.
3. **`~/.claude.json` write race** — atomic rename handles most cases; evaluate if
   an advisory `flock` is needed for the read-modify-write cycle.

---

## What to do next session

1. `cd /Users/rneumann/projects/claudectx`
2. Run `go mod init github.com/rneumann/claudectx`
3. Scaffold the directory structure:
   ```
   cmd/           cobra commands (use.go, add.go, remove.go, list.go, current.go, rename.go, refresh.go)
   internal/
     keychain/    security CLI wrapper
     store/       state.json + identity.json read/write
     claude/      ~/.claude.json patcher
   main.go
   ```
4. Start with `internal/keychain` (the foundation everything else depends on) and
   its tests using a sandbox Keychain service name.
