package diff3_test

import (
	"path/filepath"
	"strings"
	"testing"

	diff3 "github.com/CivNode/diff3-go"
)

// TestMerge_CharAware_Fixtures tests fixtures in testdata/fixtures/char-aware-*/
// using CharacterAware mode. These cases merge cleanly in CharacterAware mode
// but would conflict in LineAware mode.
func TestMerge_CharAware_Fixtures(t *testing.T) {
	root := filepath.Join("testdata", "fixtures")
	entries := []string{"char-aware-clean", "char-aware-conflict"}

	for _, name := range entries {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(root, name)
			ancestor := readFixture(t, dir, "ancestor.txt")
			a := readFixture(t, dir, "a.txt")
			b := readFixture(t, dir, "b.txt")
			expected := readFixture(t, dir, "expected.txt")
			wantConflict := strings.TrimSpace(readFixture(t, dir, "expected_has_conflicts.txt")) == "true"

			result, hadConflicts, err := diff3.Merge(ancestor, a, b, diff3.Options{Mode: diff3.CharacterAware})
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

// TestMerge_CharAware_ConflictsInLineMode verifies that the char-aware-clean fixture
// produces a conflict in default LineAware mode.
func TestMerge_CharAware_ConflictsInLineMode(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "char-aware-clean")
	ancestor := readFixture(t, dir, "ancestor.txt")
	a := readFixture(t, dir, "a.txt")
	b := readFixture(t, dir, "b.txt")

	_, hadConflicts, err := diff3.Merge(ancestor, a, b, diff3.Options{})
	if err != nil {
		t.Fatalf("Merge error: %v", err)
	}
	if !hadConflicts {
		t.Error("expected conflict in LineAware mode for char-aware-clean fixture")
	}
}
