package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/rneumann/claudectx/cmd"
)

// TestMain registers `claudectx` as a testscript command so .txtar scripts can run
// the real CLI in an isolated subprocess.
func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"claudectx": func() int {
			cmd.Execute() // os.Exit(1) on error; returns here on success
			return 0
		},
	}))
}

func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: filepath.Join("testdata", "script"),
		Setup: func(e *testscript.Env) error {
			// Sandbox all profile state under the script's work dir.
			e.Setenv("XDG_CONFIG_HOME", filepath.Join(e.WorkDir, "xdg"))
			return nil
		},
	})
}
