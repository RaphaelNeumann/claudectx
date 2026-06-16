# claudectx — Technical Architecture

---

## Language & build constraints

- **Go 1.21+**, `CGO_ENABLED=0`, single static binary.
- No CGO. The binary must cross-compile with `GOOS`/`GOARCH` without a C toolchain.
- Module path: `github.com/rneumann/claudectx`

---

## Dependency decisions

| Concern | Package | Decision |
|---------|---------|----------|
| Commands | `spf13/cobra` | Standard; subcommand routing |
| Interactive picker | `charmbracelet/huh` | Lightweight select; no full TUI needed |
| Output styling | `charmbracelet/lipgloss` | list/current views only |
| Config/identity | stdlib `encoding/json` | Config is small; no viper |
| Keychain | `os/exec` → `/usr/bin/security` | See credential mechanism below |
| Testing | stdlib `testing` + `rogpeppe/go-internal/testscript` | Script-based CLI integration tests |
| Release | `goreleaser` | Cross-platform binaries + brew tap |

---

## Credential mechanism

**Drive `/usr/bin/security` via `os/exec`. Do not use `go-keyring` or any library
that wraps the raw keychain value.**

`go-keyring` encodes the stored value as `go-keyring-base64:<b64>` before writing.
Claude Code reads the slot raw and expects plain JSON — the encoded form causes a
JSON parse error and logs the user out. Confirmed by live test on 2026-06-16.

### Keychain service names

| Service string | Owner |
|----------------|-------|
| `Claude Code-credentials` | Claude Code (live active credential) |
| `claudectx:<name>` | this tool (parked credential per context) |

The `acct` field is always `os/user.Current().Username` (the macOS login name).

### Keychain calls

```bash
# read
security find-generic-password -s "<service>" -a "<acct>" -w

# write (upsert)
security add-generic-password -U -s "<service>" -a "<acct>" -w "<json>"

# delete
security delete-generic-password -s "<service>" -a "<acct>"
```

Wrap each in a helper that:
- returns `(string, error)` for reads
- returns `error` for writes/deletes
- maps exit code 44 ("item not found") to a typed sentinel `ErrNotFound`

---

## Credential blob schema (Claude Code's format, do not alter)

```json
{
  "claudeAiOauth": {
    "accessToken":      "<string, ~108 chars>",
    "refreshToken":     "<string, ~108 chars>",
    "expiresAt":        "<int64, unix ms>",
    "scopes":           ["<string>", ...],
    "subscriptionType": "<string>",
    "rateLimitTier":    "<string>"
  }
}
```

---

## Storage layout

```
~/.config/claudectx/
  state.json                   0644
  contexts/
    <name>/
      identity.json            0600
```

Directory permissions: `~/.config/claudectx/` and all `contexts/<name>/` → 0700.

No credential material on disk. Credentials live exclusively in Keychain
(`claudectx:<name>` service).

### state.json schema

```json
{
  "active":    "<name or empty string>",
  "updatedAt": "<RFC3339>"
}
```

### identity.json schema

Extracted from `~/.claude.json` at snapshot time. Only these two fields:

```json
{
  "oauthAccount": { ...verbatim from ~/.claude.json... },
  "userID":       "<string>"
}
```

---

## `~/.claude.json` patching

Only `oauthAccount` and `userID` are replaced. The rest of the file is preserved.

Atomic write protocol:
1. `json.Unmarshal` full `~/.claude.json` into `map[string]any`.
2. Replace `oauthAccount` and `userID` keys.
3. `json.Marshal` into a temp file created with `os.CreateTemp` in the same
   directory as `~/.claude.json` (guarantees same filesystem → rename is atomic).
4. `os.Rename(tmp, "~/.claude.json")`.

Never write `~/.claude.json` directly. Never `json.Marshal` with indentation
(Claude Code writes compact JSON; match the format).

---

## `use` — switch algorithm (authoritative)

```
1. resolve <name> → load identity.json, verify claudectx:<name> keychain slot exists
2. if <name> == state.active → print "already active", exit 0
3. re-capture current active context:
     a. read "Claude Code-credentials" slot
     b. write to "claudectx:<state.active>" slot
     c. patch identity.json for <state.active> from current ~/.claude.json
   (skip step 3 entirely if state.active is empty)
4. write claudectx:<name> credential → "Claude Code-credentials" slot
5. patch ~/.claude.json with <name>'s identity.json content
6. write state.json: active = <name>
7. print confirmation
```

Steps 4–6 run sequentially. If step 4 fails, steps 5–6 are skipped and an error
is returned. The tool never writes a partial state where the keychain and identity
are mismatched.

Step 3 (re-capture) is unconditional and never skippable — this is the mechanism
that survives refresh-token rotation.

---

## Token rotation

Claude Code silently refreshes `accessToken` every ~8h using `refreshToken`,
rewrites the live `Claude Code-credentials` slot with new values, and the
`refreshToken` itself rotates (one-time-use).

The step-3 re-capture in `use` handles this: the parked snapshot always gets the
freshest tokens before being displaced.

If tokens are not re-captured before switching, the parked context's stored
`refreshToken` may be dead, causing a forced re-login on next `use`.

---

## Corruption detection

On any read of `Claude Code-credentials`, check if value starts with
`go-keyring-base64:`. If so:
- refuse the operation
- print an error identifying the cause (go-keyring library wrote to the slot)
- print recovery instruction: `claudectx restore <name>` or manual re-login

---

## Running session detection

Before patching `~/.claude.json` (step 5 of `use`), check:
```go
exec.Command("pgrep", "-x", "claude").Run() // exit 0 = process found
```
If a `claude` process is running, print a warning. Do not block; proceed unless
`--wait` flag is set (which polls until the process exits).

---

## Error states

| State | Detection | Recovery |
|-------|-----------|---------|
| Context missing keychain slot | `ErrNotFound` on read | print `claudectx remove <name>` to clean orphaned identity.json |
| Credential is go-keyring encoded | value prefix check | print repair instructions |
| `~/.claude.json` unreadable/unparseable | `json.Unmarshal` error | abort, do not patch |
| Partial `use` failure (step 4 fails) | `error` from write | print what succeeded, print `claudectx use <previous>` recovery command |
| `state.active` points to non-existent context | missing identity.json | warn, set active to empty, continue with `use` |

---

## Open questions

1. **Refresh token rotation frequency** — does `refreshToken` rotate on every
   `accessToken` refresh, or is it stable? Observed stable across one test session
   (2026-06-16). Confirm over 24h before v1 release. Affects how critical the
   re-capture step is if a user force-switches without it.

2. **Keychain ACL after code signing** — unsigned binary had zero prompts in
   testing. Re-test after signing; a signed binary may trigger a "Do you want to
   allow access?" dialog on first use. If so, evaluate whether a one-time
   `security unlock-keychain` or entitlement resolves it.

3. **`~/.claude.json` write race** — CC can rewrite this file at any time (token
   refresh, settings change). The atomic rename handles most races. Evaluate
   whether a `flock`-style advisory lock is needed for the read-modify-write cycle.
