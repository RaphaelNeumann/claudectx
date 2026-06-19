package profile

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/raphaelneumann/claudectx/internal/paths"
)

// EnvFileName is the per-profile dotenv claudectx loads into the `claude` process
// environment at launch. It is claudectx-owned; Claude Code never reads it. Vars set
// here apply only to the launched claude process, not to the user's shell.
const EnvFileName = "claudectx.env"

var envKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// EnvPath returns the path to a profile's claudectx.env file.
func EnvPath(name string) (string, error) {
	dir, err := paths.ProfileDir(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, EnvFileName), nil
}

// LoadEnv reads <dir>/claudectx.env and returns its entries as "KEY=VALUE" strings
// suitable for appending to an environment. A missing file is not an error (returns
// nil). Blank lines and lines beginning with '#' are ignored; an optional leading
// "export " is stripped; matching surrounding single/double quotes around the value
// are removed. Malformed lines are skipped. dir is the profile's config dir.
func LoadEnv(dir string) ([]string, error) {
	f, err := os.Open(filepath.Join(dir, EnvFileName))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []string
	seen := map[string]int{} // key -> index in out, so later lines override earlier
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := parseEnvLine(sc.Text())
		if !ok {
			continue
		}
		if i, dup := seen[k]; dup {
			out[i] = k + "=" + v
			continue
		}
		seen[k] = len(out)
		out = append(out, k+"="+v)
	}
	return out, sc.Err()
}

func parseEnvLine(line string) (key, val string, ok bool) {
	s := strings.TrimSpace(line)
	if s == "" || strings.HasPrefix(s, "#") {
		return "", "", false
	}
	s = strings.TrimPrefix(s, "export ")
	i := strings.IndexByte(s, '=')
	if i <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(s[:i])
	if !envKeyRe.MatchString(key) {
		return "", "", false
	}
	return key, unquote(strings.TrimSpace(s[i+1:])), true
}

func unquote(s string) string {
	if len(s) >= 2 {
		if c := s[0]; (c == '"' || c == '\'') && s[len(s)-1] == c {
			return s[1 : len(s)-1]
		}
	}
	return s
}
