// Package diff3 implements three-way text merge with Git-compatible conflict markers.
//
// The primary entry point is Merge. It performs a line-level (or character-level)
// three-way merge of two texts against their common ancestor using the Myers diff
// algorithm. Clean regions are taken without markers; conflicting regions are
// surrounded by Git-style <<<<<<< / ======= / >>>>>>> markers.
//
// The package has no dependencies outside the Go standard library.
package diff3

import (
	"strings"

	"github.com/CivNode/diff3-go/internal/myers"
)

// Mode selects the granularity used when classifying conflict regions.
type Mode int

const (
	// LineAware (default) treats each newline-terminated line as the smallest unit.
	// Two modifications to the same line always conflict.
	LineAware Mode = iota

	// CharacterAware performs a secondary character-level diff within "both modified"
	// line regions. If the character-level changes are non-overlapping, the region
	// merges cleanly even though the lines differ.
	CharacterAware
)

// Options configures the behaviour of Merge.
type Options struct {
	// Mode selects line-aware (default) or character-aware merging.
	Mode Mode

	// MarkerLeft overrides the opening conflict marker (default "<<<<<<< ").
	MarkerLeft string

	// MarkerAncestor overrides the ancestor divider (default "=======").
	MarkerAncestor string

	// MarkerRight overrides the closing conflict marker (default ">>>>>>>").
	MarkerRight string
}

// Merge performs a three-way merge of a and b against their common ancestor.
//
// The algorithm:
//  1. Diff ancestor->a and ancestor->b independently (Myers line-level diff).
//  2. Walk the ancestor position-by-position, recording what each side produces.
//     A "replacement" (delete+insert pair in the diff) is kept together as a unit.
//  3. For each position: if only one side changed, take that side; if both changed
//     identically, take either; if both changed differently, emit conflict markers.
//
// Returns the merged text, whether any unresolvable conflicts were found, and any
// internal error (currently always nil, reserved for future use).
func Merge(ancestor, a, b string, opts Options) (result string, hadConflicts bool, err error) {
	ml := opts.MarkerLeft
	if ml == "" {
		ml = "<<<<<<< "
	}
	ma := opts.MarkerAncestor
	if ma == "" {
		ma = "======="
	}
	mr := opts.MarkerRight
	if mr == "" {
		mr = ">>>>>>>"
	}

	ancLines := splitLines(ancestor)
	aLines := splitLines(a)
	bLines := splitLines(b)

	hunksA := myers.Diff(ancLines, aLines)
	hunksB := myers.Diff(ancLines, bLines)

	regions := mergeRegions(ancLines, hunksA, hunksB)

	var sb strings.Builder
	for _, r := range regions {
		switch r.kind {
		case rkEqual, rkTakeA:
			for _, ln := range r.aLines {
				sb.WriteString(ln)
			}
		case rkTakeB:
			for _, ln := range r.bLines {
				sb.WriteString(ln)
			}
		case rkConflict:
			if opts.Mode == CharacterAware {
				merged, clean := charMerge(r.aLines, r.bLines, r.ancLines)
				if clean {
					sb.WriteString(merged)
					continue
				}
			}
			hadConflicts = true
			sb.WriteString(ml)
			sb.WriteByte('\n')
			for _, ln := range r.aLines {
				sb.WriteString(ln)
			}
			sb.WriteString(ma)
			sb.WriteByte('\n')
			for _, ln := range r.bLines {
				sb.WriteString(ln)
			}
			sb.WriteString(mr)
			sb.WriteByte('\n')
		}
	}
	return sb.String(), hadConflicts, nil
}

// regionKind classifies a merged region.
type regionKind int

const (
	rkEqual    regionKind = iota // both sides identical to ancestor
	rkTakeA                      // only A changed (take A's output)
	rkTakeB                      // only B changed (take B's output)
	rkConflict                   // both changed, differently
)

// mergeRegion is one aligned block from the three-way merge.
type mergeRegion struct {
	kind     regionKind
	ancLines []string // ancestor lines for this region
	aLines   []string // what A produces (nil = deleted)
	bLines   []string // what B produces (nil = deleted)
}

// posState describes what one side does to ancestor line i.
type posState struct {
	// state: 0=unchanged, 1=deleted, 2=replaced
	state   int
	replace []string // valid when state==2
	// preIns: pure insertions before ancestor line i (no deletion)
	preIns []string
}

// buildPosData converts a diff hunk list into per-ancestor-position state.
// pos has ancN+1 slots; slot ancN is for trailing inserts.
func buildPosData(hunks []myers.Hunk, ancN int) []posState {
	pos := make([]posState, ancN+1)
	ancIdx := 0
	for hi := 0; hi < len(hunks); hi++ {
		h := hunks[hi]
		switch h.Kind {
		case myers.Equal:
			ancIdx += len(h.Lines)

		case myers.Delete:
			ancStart := ancIdx
			ancIdx += len(h.Lines)
			// Peek at next hunk: if it's an Insert, this is a replacement.
			var insertLines []string
			if hi+1 < len(hunks) && hunks[hi+1].Kind == myers.Insert {
				hi++
				insertLines = hunks[hi].Lines
			}
			nDel := ancIdx - ancStart
			nIns := len(insertLines)
			if nIns > 0 {
				if nIns == nDel {
					// Equal-length replacement: distribute 1:1 across ancestor positions.
					// This preserves per-line alignment for cleaner conflict detection.
					for i := ancStart; i < ancIdx && i < ancN; i++ {
						pos[i].state = 2
						pos[i].replace = insertLines[i-ancStart : i-ancStart+1]
					}
				} else {
					// Different-length replacement: mark ancStart as "replaced" with all
					// inserted lines; mark the rest of the deleted positions as "deleted".
					if ancStart < ancN {
						pos[ancStart].state = 2
						pos[ancStart].replace = insertLines
					}
					for i := ancStart + 1; i < ancIdx && i < ancN; i++ {
						pos[i].state = 1
					}
				}
			} else {
				// Pure deletion.
				for i := ancStart; i < ancIdx && i < ancN; i++ {
					pos[i].state = 1
				}
			}

		case myers.Insert:
			// Pure insert before ancIdx.
			slot := ancIdx
			if slot > ancN {
				slot = ancN
			}
			pos[slot].preIns = append(pos[slot].preIns, h.Lines...)
		}
	}
	return pos
}

// mergeRegions produces the sequence of merged regions.
func mergeRegions(ancLines []string, hunksA, hunksB []myers.Hunk) []mergeRegion {
	ancN := len(ancLines)
	posA := buildPosData(hunksA, ancN)
	posB := buildPosData(hunksB, ancN)

	var out []mergeRegion

	add := func(kind regionKind, anc, aL, bL []string) {
		if len(out) > 0 && out[len(out)-1].kind == kind {
			last := &out[len(out)-1]
			last.ancLines = append(last.ancLines, anc...)
			last.aLines = append(last.aLines, aL...)
			last.bLines = append(last.bLines, bL...)
			return
		}
		out = append(out, mergeRegion{
			kind:     kind,
			ancLines: append([]string(nil), anc...),
			aLines:   append([]string(nil), aL...),
			bLines:   append([]string(nil), bL...),
		})
	}

	for i := 0; i <= ancN; i++ {
		pA := posA[i]
		pB := posB[i]

		// Handle pure insertions before position i.
		insA := pA.preIns
		insB := pB.preIns
		hasInsA := len(insA) > 0
		hasInsB := len(insB) > 0
		if hasInsA || hasInsB {
			switch {
			case strSlicesEqual(insA, insB):
				add(rkTakeA, nil, insA, insA)
			case hasInsA && !hasInsB:
				add(rkTakeA, nil, insA, nil)
			case !hasInsA && hasInsB:
				add(rkTakeB, nil, nil, insB)
			default:
				add(rkConflict, nil, insA, insB)
			}
		}

		if i == ancN {
			break
		}

		ancLine := ancLines[i]

		// Compute each side's output for ancestor line i.
		var aOut, bOut []string
		switch pA.state {
		case 0:
			aOut = []string{ancLine}
		case 1:
			aOut = nil
		case 2:
			aOut = pA.replace
		}
		switch pB.state {
		case 0:
			bOut = []string{ancLine}
		case 1:
			bOut = nil
		case 2:
			bOut = pB.replace
		}

		aChanged := pA.state != 0
		bChanged := pB.state != 0

		switch {
		case !aChanged && !bChanged:
			add(rkEqual, []string{ancLine}, aOut, bOut)
		case aChanged && !bChanged:
			add(rkTakeA, []string{ancLine}, aOut, bOut)
		case !aChanged && bChanged:
			add(rkTakeB, []string{ancLine}, aOut, bOut)
		default: // both changed
			if strSlicesEqual(aOut, bOut) {
				add(rkTakeA, []string{ancLine}, aOut, bOut)
			} else {
				add(rkConflict, []string{ancLine}, aOut, bOut)
			}
		}
	}

	return out
}

// splitLines splits text into a slice of lines. Each line retains its terminating
// newline character. A final fragment without a trailing newline is returned as-is.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func strSlicesEqual(a, b []string) bool {
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

// charMerge attempts to resolve a conflict at character granularity.
// It returns the merged text and true if the merge was clean.
// aLines and bLines are the conflicting line sets; ancLines are the ancestor lines.
func charMerge(aLines, bLines, ancLines []string) (string, bool) {
	ancText := strings.Join(ancLines, "")
	aText := strings.Join(aLines, "")
	bText := strings.Join(bLines, "")

	ancChars := splitChars(ancText)
	aChars := splitChars(aText)
	bChars := splitChars(bText)

	hunksA := myers.Diff(ancChars, aChars)
	hunksB := myers.Diff(ancChars, bChars)

	regions := mergeRegions(ancChars, hunksA, hunksB)

	var sb strings.Builder
	for _, r := range regions {
		switch r.kind {
		case rkEqual, rkTakeA:
			for _, ch := range r.aLines {
				sb.WriteString(ch)
			}
		case rkTakeB:
			for _, ch := range r.bLines {
				sb.WriteString(ch)
			}
		case rkConflict:
			return "", false
		}
	}
	return sb.String(), true
}

// splitChars splits a string into a slice of individual UTF-8 characters (runes).
func splitChars(s string) []string {
	runes := []rune(s)
	out := make([]string, len(runes))
	for i, r := range runes {
		out[i] = string(r)
	}
	return out
}
