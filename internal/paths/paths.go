// Package paths resolves the on-disk locations claudectx owns.
//
// Layout (see architecture.md):
//
//	~/.config/claudectx/
//	  state.json
//	  shared/{agents,skills,commands}/
//	  profiles/<name>/   == CLAUDE_CONFIG_DIR for that profile
package paths

import (
	"os"
	"path/filepath"
)

const appDir = "claudectx"

// SharedSubdirs are the resource directories shared across all profiles.
var SharedSubdirs = []string{"agents", "skills", "commands"}

// Root returns ~/.config/claudectx, honoring XDG_CONFIG_HOME.
func Root() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, appDir), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", appDir), nil
}

// StateFile returns the path to state.json.
func StateFile() (string, error) {
	r, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(r, "state.json"), nil
}

// SharedDir returns ~/.config/claudectx/shared.
func SharedDir() (string, error) {
	r, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(r, "shared"), nil
}

// ProfilesDir returns ~/.config/claudectx/profiles.
func ProfilesDir() (string, error) {
	r, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(r, "profiles"), nil
}

// ProfileDir returns the config dir for a named profile (its CLAUDE_CONFIG_DIR).
func ProfileDir(name string) (string, error) {
	p, err := ProfilesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(p, name), nil
}
