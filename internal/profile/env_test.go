package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnv(t *testing.T) {
	dir := t.TempDir()

	// Missing file is not an error.
	got, err := LoadEnv(dir)
	if err != nil {
		t.Fatalf("LoadEnv(missing): %v", err)
	}
	if got != nil {
		t.Errorf("LoadEnv(missing) = %v, want nil", got)
	}

	content := "" +
		"# a comment\n" +
		"\n" +
		"CLAUDE_CODE_USE_VERTEX=1\n" +
		"export CLOUD_ML_REGION=us-east5\n" +
		"QUOTED=\"has space\"\n" +
		"SINGLE='val#notcomment'\n" +
		"  PADDED  =  trimmed  \n" +
		"=novalue\n" + // malformed: no key
		"bad key=skip\n" + // malformed: invalid key
		"CLAUDE_CODE_USE_VERTEX=2\n" // later line overrides earlier
	if err := os.WriteFile(filepath.Join(dir, EnvFileName), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err = LoadEnv(dir)
	if err != nil {
		t.Fatalf("LoadEnv: %v", err)
	}
	want := []string{
		"CLAUDE_CODE_USE_VERTEX=2", // overridden, keeps original position
		"CLOUD_ML_REGION=us-east5", // export stripped
		"QUOTED=has space",         // double quotes stripped
		"SINGLE=val#notcomment",    // single quotes stripped, inner # kept
		"PADDED=trimmed",           // key/value trimmed
	}
	if len(got) != len(want) {
		t.Fatalf("LoadEnv = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("LoadEnv[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
