// Package launch resolves a profile to its CLAUDE_CONFIG_DIR and replaces the
// current process with `claude` running in it.
package launch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rneumann/claudectx/internal/paths"
	"github.com/rneumann/claudectx/internal/profile"
)

// Exec replaces the current process with `claude`, with CLAUDE_CONFIG_DIR pointed
// at the named profile. Extra args are forwarded to claude verbatim. On success it
// does not return (the process image is replaced).
func Exec(name string, args []string) error {
	if !profile.Exists(name) {
		return fmt.Errorf("profile %q does not exist (run `claudectx add %s`)", name, name)
	}
	if err := profile.EnsureSymlinks(name); err != nil {
		return fmt.Errorf("ensuring shared layer: %w", err)
	}
	pdir, err := paths.ProfileDir(name)
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(pdir)
	if err != nil {
		return err
	}
	bin, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("`claude` not found on PATH: %w", err)
	}
	env := setEnv(os.Environ(), "CLAUDE_CONFIG_DIR", abs)
	argv := append([]string{"claude"}, args...)
	return syscall.Exec(bin, argv, env)
}

// setEnv returns env with key set to val, replacing any existing entry.
func setEnv(env []string, key, val string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			continue
		}
		out = append(out, e)
	}
	return append(out, prefix+val)
}
