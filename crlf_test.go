package diff3_test

import (
	"testing"

	diff3 "github.com/CivNode/diff3-go"
)

// TestMerge_CRLFMix verifies that CRLF and LF line endings are treated as
// distinct strings. When B changes a line from LF to CRLF and A changes the
// same line's content, the result is a conflict.
//
// This test is defined in code rather than a fixture file because git normalizes
// CRLF to LF on checkout, which would corrupt file-based CRLF fixtures.
func TestMerge_CRLFMix(t *testing.T) {
	// ancestor: three LF lines
	ancestor := "line1\nline2\nline3\n"
	// a: changes line2 to "modified" (LF)
	a := "line1\nmodified\nline3\n"
	// b: converts all lines to CRLF endings
	b := "line1\r\nline2\r\nline3\r\n"

	// Expected: line1 taken from B (CRLF, only B changed it), line2 conflict,
	// line3 taken from B (CRLF, only B changed it).
	want := "line1\r\n<<<<<<< \nmodified\n=======\nline2\r\n>>>>>>>\nline3\r\n"

	result, hadConflicts, err := diff3.Merge(ancestor, a, b, diff3.Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hadConflicts {
		t.Error("expected conflicts for CRLF/LF mismatch on modified line")
	}
	if result != want {
		t.Errorf("result mismatch\ngot:\n%q\nwant:\n%q", result, want)
	}
}
