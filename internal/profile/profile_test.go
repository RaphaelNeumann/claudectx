package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raphaelneumann/claudectx/internal/paths"
)

// sandbox points the config root at a temp dir for the duration of a test.
func sandbox(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return filepath.Join(dir, "claudectx")
}

func TestValidateName(t *testing.T) {
	good := []string{"personal", "work", "a", "a.b_c-1"}
	bad := []string{"", ".", "..", "../evil", "a/b", `a\b`, "-leading", ".dot"}
	for _, n := range good {
		if err := ValidateName(n); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", n, err)
		}
	}
	for _, n := range bad {
		if err := ValidateName(n); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", n)
		}
	}
}

func TestAddCreatesProfileWithSharedSymlinks(t *testing.T) {
	root := sandbox(t)

	if err := Add("work"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !Exists("work") {
		t.Fatal("Exists(work) = false after Add")
	}

	for _, sub := range paths.SharedSubdirs {
		link := filepath.Join(root, "profiles", "work", sub)
		fi, err := os.Lstat(link)
		if err != nil {
			t.Fatalf("Lstat(%s): %v", link, err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s is not a symlink", sub)
		}
		got, _ := os.Readlink(link)
		want := filepath.Join("..", "..", "shared", sub)
		if got != want {
			t.Errorf("%s links to %q, want %q", sub, got, want)
		}
	}
}

func TestSharedLayerIsShared(t *testing.T) {
	root := sandbox(t)
	if err := Add("a"); err != nil {
		t.Fatal(err)
	}
	if err := Add("b"); err != nil {
		t.Fatal(err)
	}
	// Write through the shared dir; both profiles must see it.
	if err := os.WriteFile(filepath.Join(root, "shared", "agents", "x.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"a", "b"} {
		seen := filepath.Join(root, "profiles", p, "agents", "x.md")
		if _, err := os.Stat(seen); err != nil {
			t.Errorf("profile %q does not see shared agent: %v", p, err)
		}
	}
}

func TestAddDuplicateFails(t *testing.T) {
	sandbox(t)
	if err := Add("dup"); err != nil {
		t.Fatal(err)
	}
	if err := Add("dup"); err == nil {
		t.Fatal("Add(dup) twice = nil, want error")
	}
}

func TestRenamePreservesSymlinksAndRemove(t *testing.T) {
	root := sandbox(t)
	if err := Add("old"); err != nil {
		t.Fatal(err)
	}
	if err := Rename("old", "new"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if Exists("old") || !Exists("new") {
		t.Fatal("rename did not move the profile")
	}
	// Relative symlink must still resolve after the move.
	link := filepath.Join(root, "profiles", "new", "agents")
	if got, _ := os.Readlink(link); got != filepath.Join("..", "..", "shared", "agents") {
		t.Errorf("symlink target after rename = %q", got)
	}

	if err := Remove("new"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if Exists("new") {
		t.Fatal("profile still exists after Remove")
	}
	// Removing a profile must not delete the shared copy.
	if _, err := os.Stat(filepath.Join(root, "shared", "agents")); err != nil {
		t.Errorf("shared dir was damaged by Remove: %v", err)
	}
}

func TestEnsureSymlinksSelfHeals(t *testing.T) {
	root := sandbox(t)
	if err := Add("h"); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "profiles", "h", "agents")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := EnsureSymlinks("h"); err != nil {
		t.Fatalf("EnsureSymlinks: %v", err)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Errorf("symlink not recreated: %v", err)
	}
}

func TestAddSeedsClaudeMD(t *testing.T) {
	root := sandbox(t)

	if err := Add("demo"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	mdPath := filepath.Join(root, "profiles", "demo", "CLAUDE.md")
	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "demo") {
		t.Error("CLAUDE.md does not contain the profile name")
	}
	if !strings.Contains(content, "settings.json") {
		t.Error("CLAUDE.md does not contain the settings warning")
	}
}

func TestSeedClaudeMDDoesNotOverwrite(t *testing.T) {
	root := sandbox(t)

	if err := Add("keep"); err != nil {
		t.Fatal(err)
	}
	mdPath := filepath.Join(root, "profiles", "keep", "CLAUDE.md")
	custom := []byte("my custom instructions")
	if err := os.WriteFile(mdPath, custom, 0o644); err != nil {
		t.Fatal(err)
	}
	// EnsureSymlinks also calls seedClaudeMD — must not overwrite.
	if err := EnsureSymlinks("keep"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(custom) {
		t.Error("seedClaudeMD overwrote existing CLAUDE.md")
	}
}

func TestListSorted(t *testing.T) {
	sandbox(t)
	for _, n := range []string{"charlie", "alpha", "bravo"} {
		if err := Add(n); err != nil {
			t.Fatal(err)
		}
	}
	got, err := List()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "bravo", "charlie"}
	if len(got) != len(want) {
		t.Fatalf("List len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i] {
			t.Errorf("List[%d] = %q, want %q", i, got[i].Name, want[i])
		}
	}
}
