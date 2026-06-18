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

	"github.com/rneumann/claudectx/internal/paths"
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
// an intentional per-profile override and left untouched.
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
	return nil
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
	return EnsureSymlinks(name)
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
