package diff3_test

import (
	"testing"
	"testing/quick"

	diff3 "github.com/CivNode/diff3-go"
)

// TestMergeIdempotent verifies that Merge(c, a, a) == a with no conflicts.
// When both sides make the same change, the result is always that change.
func TestMergeIdempotent(t *testing.T) {
	err := quick.Check(func(a, c string) bool {
		result, hadConflicts, mergeErr := diff3.Merge(c, a, a, diff3.Options{})
		if mergeErr != nil {
			return false
		}
		return !hadConflicts && result == a
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

// TestMergeSymmetric verifies that Merge(c, a, b) and Merge(c, b, a) have the
// same hadConflicts value. The text content may differ (marker order is swapped)
// but conflict presence must be symmetric.
func TestMergeSymmetric(t *testing.T) {
	err := quick.Check(func(a, b, c string) bool {
		_, hadAB, errAB := diff3.Merge(c, a, b, diff3.Options{})
		_, hadBA, errBA := diff3.Merge(c, b, a, diff3.Options{})
		return errAB == nil && errBA == nil && hadAB == hadBA
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

// TestMergeTrivialCases verifies the three trivial merge scenarios.
func TestMergeTrivialCases(t *testing.T) {
	t.Run("Merge_CCC_equals_C", func(t *testing.T) {
		err := quick.Check(func(c string) bool {
			result, hadConflicts, mergeErr := diff3.Merge(c, c, c, diff3.Options{})
			return mergeErr == nil && !hadConflicts && result == c
		}, nil)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Merge_CAC_equals_A", func(t *testing.T) {
		// Merge(c, a, c) == a: B unchanged means take A.
		err := quick.Check(func(a, c string) bool {
			result, hadConflicts, mergeErr := diff3.Merge(c, a, c, diff3.Options{})
			return mergeErr == nil && !hadConflicts && result == a
		}, nil)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Merge_CCB_equals_B", func(t *testing.T) {
		// Merge(c, c, b) == b: A unchanged means take B.
		err := quick.Check(func(b, c string) bool {
			result, hadConflicts, mergeErr := diff3.Merge(c, c, b, diff3.Options{})
			return mergeErr == nil && !hadConflicts && result == b
		}, nil)
		if err != nil {
			t.Error(err)
		}
	})
}

// TestMergeCustomMarkers verifies that custom conflict markers are used.
func TestMergeCustomMarkers(t *testing.T) {
	opts := diff3.Options{
		MarkerLeft:     "<<< OURS",
		MarkerAncestor: "|||||||",
		MarkerRight:    ">>> THEIRS",
	}
	ancestor := "line1\noriginal\nline3\n"
	a := "line1\nchange_a\nline3\n"
	b := "line1\nchange_b\nline3\n"

	result, hadConflicts, err := diff3.Merge(ancestor, a, b, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hadConflicts {
		t.Fatal("expected conflict")
	}

	want := "line1\n<<< OURS\nchange_a\n|||||||\nchange_b\n>>> THEIRS\nline3\n"
	if result != want {
		t.Errorf("got:\n%q\nwant:\n%q", result, want)
	}
}
