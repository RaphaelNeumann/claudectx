// Package profile manages profile directories and the shared resource layer.
//
// A profile is a directory under ~/.config/claudectx/profiles/<name> that serves
// as a Claude Code CLAUDE_CONFIG_DIR. claudectx owns the directory and the
// shared-layer symlinks inside it; Claude Code owns everything else (.claude.json,
// history, credentials slot). See architecture.md.
package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/raphaelneumann/claudectx/internal/paths"
)

var nameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ValidateName rejects names that are empty, path-traversing, or otherwise unsafe
// as a single directory component.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name is empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid profile name %q", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("profile name %q may not contain path separators", name)
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("profile name %q must match [A-Za-z0-9._-] and start alphanumeric", name)
	}
	return nil
}

// Exists reports whether a profile directory exists.
func Exists(name string) bool {
	pdir, err := paths.ProfileDir(name)
	if err != nil {
		return false
	}
	fi, err := os.Stat(pdir)
	return err == nil && fi.IsDir()
}

// EnsureShared creates the shared/{agents,skills,commands} directories.
func EnsureShared() error {
	sh, err := paths.SharedDir()
	if err != nil {
		return err
	}
	for _, sub := range paths.SharedSubdirs {
		if err := os.MkdirAll(filepath.Join(sh, sub), 0o700); err != nil {
			return err
		}
	}
	return nil
}

// EnsureSymlinks makes each profiles/<name>/<sub> a symlink to ../../shared/<sub>.
// It self-heals broken or wrong symlinks. A real (non-symlink) entry is treated as
// an intentional per-profile override and left untouched. It also seeds CLAUDE.md
// if missing (backfills profiles created before this feature).
func EnsureSymlinks(name string) error {
	pdir, err := paths.ProfileDir(name)
	if err != nil {
		return err
	}
	if err := EnsureShared(); err != nil {
		return err
	}
	for _, sub := range paths.SharedSubdirs {
		link := filepath.Join(pdir, sub)
		// Relative target from profiles/<name>/ up to shared/<sub>.
		target := filepath.Join("..", "..", "shared", sub)
		fi, err := os.Lstat(link)
		switch {
		case os.IsNotExist(err):
			if err := os.Symlink(target, link); err != nil {
				return err
			}
		case err != nil:
			return err
		case fi.Mode()&os.ModeSymlink != 0:
			if cur, _ := os.Readlink(link); cur != target {
				if err := os.Remove(link); err != nil {
					return err
				}
				if err := os.Symlink(target, link); err != nil {
					return err
				}
			}
		default:
			// real dir/file present — per-profile override, leave as-is
		}
	}
	return seedClaudeMD(name, pdir)
}

// Add creates a new profile directory and its shared-layer symlinks.
func Add(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if Exists(name) {
		return fmt.Errorf("profile %q already exists", name)
	}
	pdir, err := paths.ProfileDir(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(pdir, 0o700); err != nil {
		return err
	}
	if err := EnsureSymlinks(name); err != nil {
		return err
	}
	return seedClaudeMD(name, pdir)
}

// seedClaudeMD writes a CLAUDE.md into the profile directory if one does not
// already exist. The file is loaded by Claude Code as user-level instructions
// for every session under this profile, telling it that settings changes belong
// in the profile's own settings.json (not ~/.claude/settings.json).
func seedClaudeMD(name, pdir string) error {
	p := filepath.Join(pdir, "CLAUDE.md")
	if _, err := os.Stat(p); err == nil {
		return nil // already exists — don't overwrite user customizations
	}
	content := fmt.Sprintf(`# claudectx profile: %s

This session is running under a claudectx profile. The profile's config
directory is $CLAUDE_CONFIG_DIR (this directory).

**Settings changes (model, effort, theme, etc.) MUST be made in this
directory's settings.json, NOT in ~/.claude/settings.json.** Claude Code
always loads ~/.claude/settings.json as a global baseline — any setting
placed there overrides all profiles, defeating per-profile isolation.

Use /config or /model within the session (they write to the correct
$CLAUDE_CONFIG_DIR/settings.json), or edit the file directly.
`, name)
	return os.WriteFile(p, []byte(content), 0o644)
}

// Remove deletes a profile directory. It refuses to remove anything that is not a
// direct child of the profiles directory. The shared/ copy is never touched
// (os.RemoveAll removes the symlinks, not their targets). The Keychain credential
// slot is left in place (claudectx does not manage credentials).
func Remove(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if !Exists(name) {
		return fmt.Errorf("profile %q does not exist", name)
	}
	pdir, err := paths.ProfileDir(name)
	if err != nil {
		return err
	}
	pd, err := paths.ProfilesDir()
	if err != nil {
		return err
	}
	absP, err := filepath.Abs(pdir)
	if err != nil {
		return err
	}
	absRoot, err := filepath.Abs(pd)
	if err != nil {
		return err
	}
	if filepath.Dir(absP) != absRoot {
		return fmt.Errorf("refusing to remove %s: not inside %s", absP, absRoot)
	}
	return os.RemoveAll(pdir)
}

// Rename moves a profile directory. The shared-layer symlinks are relative
// (../../shared/...) so they survive the move. NOTE: the Keychain credential slot
// is derived from the directory path, so a rename orphans the old credential and
// forces a re-login (see architecture.md Open Question 2).
func Rename(oldName, newName string) error {
	if err := ValidateName(oldName); err != nil {
		return err
	}
	if err := ValidateName(newName); err != nil {
		return err
	}
	if !Exists(oldName) {
		return fmt.Errorf("profile %q does not exist", oldName)
	}
	if Exists(newName) {
		return fmt.Errorf("profile %q already exists", newName)
	}
	oldDir, err := paths.ProfileDir(oldName)
	if err != nil {
		return err
	}
	newDir, err := paths.ProfileDir(newName)
	if err != nil {
		return err
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		return err
	}
	return EnsureSymlinks(newName)
}

// SlotName returns the Keychain service name Claude Code uses for a profile.
//
// Slot name is "Claude Code-credentials-" + sha256(absDir)[:8]. Confirmed live
// 2026-06-18: a real login produced exactly the predicted slot (see architecture.md
// Resolved validations). NFC normalization is a no-op for the ASCII profile dirs we
// create. Used for the "logged in" marker in `list`.
func SlotName(name string) (string, error) {
	pdir, err := paths.ProfileDir(name)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(pdir)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(abs))
	return "Claude Code-credentials-" + hex.EncodeToString(sum[:])[:8], nil
}

// HasCredential reports whether a Keychain slot exists for the profile. This is a
// read-only probe; claudectx never writes credentials.
func HasCredential(name string) bool {
	slot, err := SlotName(name)
	if err != nil {
		return false
	}
	return exec.Command("/usr/bin/security", "find-generic-password", "-s", slot).Run() == nil
}

// Info describes a profile for listing.
type Info struct {
	Name          string
	HasCredential bool
}

// List returns all profiles sorted by name.
func List() ([]Info, error) {
	pd, err := paths.ProfilesDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(pd)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Info
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		out = append(out, Info{Name: e.Name(), HasCredential: HasCredential(e.Name())})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
