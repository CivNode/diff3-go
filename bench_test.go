package diff3_test

import (
	"fmt"
	"strings"
	"testing"

	diff3 "github.com/CivNode/diff3-go"
)

// generateText creates a deterministic text of roughly lineCount lines.
func generateText(lineCount int, prefix string) string {
	var sb strings.Builder
	for i := 0; i < lineCount; i++ {
		fmt.Fprintf(&sb, "%s line %d: the quick brown fox jumped over the lazy dog\n", prefix, i)
	}
	return sb.String()
}

// generateConflictingText creates a variant with every modEvery-th line modified.
func generateConflictingText(base string, modEvery int, suffix string) string {
	lines := strings.SplitAfter(base, "\n")
	for i := 0; i < len(lines); i += modEvery {
		if lines[i] != "" {
			lines[i] = lines[i][:len(lines[i])-1] + " " + suffix + "\n"
		}
	}
	return strings.Join(lines, "")
}

// BenchmarkMerge10KB benchmarks a ~10 KB three-way merge with sparse clean changes.
func BenchmarkMerge10KB(b *testing.B) {
	ancestor := generateText(130, "anc") // ~10 KB
	a := generateConflictingText(ancestor, 20, "[a]")
	bText := generateConflictingText(ancestor, 25, "[b]")
	b.SetBytes(int64(len(ancestor)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = diff3.Merge(ancestor, a, bText, diff3.Options{})
	}
}

// BenchmarkMerge100KB benchmarks a ~100 KB three-way merge with many conflicts.
func BenchmarkMerge100KB(b *testing.B) {
	ancestor := generateText(1300, "anc") // ~100 KB
	a := generateConflictingText(ancestor, 20, "[a]")
	bText := generateConflictingText(ancestor, 20, "[b]") // same positions -> conflicts
	b.SetBytes(int64(len(ancestor)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = diff3.Merge(ancestor, a, bText, diff3.Options{})
	}
}

// BenchmarkMerge2MB benchmarks a ~2 MB three-way merge targeting <20 ms.
// The input has many overlapping hunks (every 50th line conflicts) — a realistic
// worst case for a large source file where two engineers edit different parts of
// the same function repeatedly.
func BenchmarkMerge2MB(b *testing.B) {
	ancestor := generateText(26000, "anc") // ~1.7 MB
	// Changes every 50th line on each side — realistic many-conflict scenario.
	a := generateConflictingText(ancestor, 50, "[a]")
	bText := generateConflictingText(ancestor, 50, "[b]") // same positions -> conflicts
	b.SetBytes(int64(len(ancestor)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = diff3.Merge(ancestor, a, bText, diff3.Options{})
	}
}
