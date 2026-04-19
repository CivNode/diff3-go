package myers_test

import (
	"testing"

	"github.com/CivNode/diff3-go/internal/myers"
)

func TestDiff_Identical(t *testing.T) {
	a := []string{"a", "b", "c"}
	hunks := myers.Diff(a, a)
	for _, h := range hunks {
		if h.Kind != myers.Equal {
			t.Fatalf("expected all Equal hunks for identical input, got %v", h.Kind)
		}
	}
}

func TestDiff_AllInsert(t *testing.T) {
	a := []string{}
	b := []string{"x", "y"}
	hunks := myers.Diff(a, b)
	if len(hunks) != 1 || hunks[0].Kind != myers.Insert || len(hunks[0].Lines) != 2 {
		t.Fatalf("expected single Insert hunk with 2 lines, got %v", hunks)
	}
}

func TestDiff_AllDelete(t *testing.T) {
	a := []string{"x", "y"}
	b := []string{}
	hunks := myers.Diff(a, b)
	if len(hunks) != 1 || hunks[0].Kind != myers.Delete || len(hunks[0].Lines) != 2 {
		t.Fatalf("expected single Delete hunk with 2 lines, got %v", hunks)
	}
}

func TestDiff_SimpleEdit(t *testing.T) {
	// a=["a","b","c"] b=["a","x","c"] → equal(a), delete(b), insert(x), equal(c)
	a := []string{"a", "b", "c"}
	b := []string{"a", "x", "c"}
	hunks := myers.Diff(a, b)

	// Reconstruct b from hunks to verify correctness
	var got []string
	for _, h := range hunks {
		switch h.Kind {
		case myers.Equal, myers.Insert:
			got = append(got, h.Lines...)
		}
	}
	if !strSliceEqual(got, b) {
		t.Fatalf("reconstructed %v, want %v", got, b)
	}
}

func TestDiff_AppendLines(t *testing.T) {
	a := []string{"a", "b"}
	b := []string{"a", "b", "c", "d"}
	hunks := myers.Diff(a, b)
	var got []string
	for _, h := range hunks {
		if h.Kind == myers.Equal || h.Kind == myers.Insert {
			got = append(got, h.Lines...)
		}
	}
	if !strSliceEqual(got, b) {
		t.Fatalf("reconstructed %v, want %v", got, b)
	}
}

func TestDiff_PrependLines(t *testing.T) {
	a := []string{"c", "d"}
	b := []string{"a", "b", "c", "d"}
	hunks := myers.Diff(a, b)
	var got []string
	for _, h := range hunks {
		if h.Kind == myers.Equal || h.Kind == myers.Insert {
			got = append(got, h.Lines...)
		}
	}
	if !strSliceEqual(got, b) {
		t.Fatalf("reconstructed %v, want %v", got, b)
	}
}

func TestDiff_EmptyBoth(t *testing.T) {
	hunks := myers.Diff(nil, nil)
	if len(hunks) != 0 {
		t.Fatalf("expected no hunks for empty inputs, got %v", hunks)
	}
}

// TestDiff_ReconstructA verifies that Equal + Delete lines reconstruct a.
func TestDiff_ReconstructA(t *testing.T) {
	a := []string{"one", "two", "three", "four"}
	b := []string{"one", "THREE", "four", "five"}
	hunks := myers.Diff(a, b)

	var gotA []string
	for _, h := range hunks {
		if h.Kind == myers.Equal || h.Kind == myers.Delete {
			gotA = append(gotA, h.Lines...)
		}
	}
	if !strSliceEqual(gotA, a) {
		t.Fatalf("reconstructed a=%v, want %v", gotA, a)
	}
}

func strSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
