package diff3_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	diff3 "github.com/CivNode/diff3-go"
)

// TestMerge_Fixtures discovers every directory under testdata/fixtures/ and runs
// a golden-file merge test: ancestor.txt, a.txt, b.txt -> expected.txt + expected_has_conflicts.txt.
func TestMerge_Fixtures(t *testing.T) {
	root := filepath.Join("testdata", "fixtures")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("cannot read fixtures dir %q: %v", root, err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip character-aware fixtures here; they are tested separately.
		if strings.HasPrefix(name, "char-aware-") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(root, name)
			ancestor := readFixture(t, dir, "ancestor.txt")
			a := readFixture(t, dir, "a.txt")
			b := readFixture(t, dir, "b.txt")
			expected := readFixture(t, dir, "expected.txt")
			wantConflict := strings.TrimSpace(readFixture(t, dir, "expected_has_conflicts.txt")) == "true"

			result, hadConflicts, err := diff3.Merge(ancestor, a, b, diff3.Options{})
			if err != nil {
				t.Fatalf("Merge error: %v", err)
			}
			if hadConflicts != wantConflict {
				t.Errorf("hadConflicts=%v, want %v", hadConflicts, wantConflict)
			}
			if result != expected {
				t.Errorf("result mismatch\ngot:\n%q\nwant:\n%q", result, expected)
			}
		})
	}
}

func readFixture(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("cannot read fixture %q: %v", filepath.Join(dir, name), err)
	}
	return string(data)
}
