// Package myers implements the Myers line-level diff algorithm.
//
// The algorithm computes the shortest edit script (SES) between two slices of
// strings and returns a sequence of hunks classifying each region as Equal,
// Insert, or Delete. The implementation follows the original O(ND) paper by
// Eugene W. Myers (1986) with a compact per-step trace to keep memory at O(D^2)
// rather than O(N*D).
package myers

// Kind identifies the type of a diff hunk.
type Kind int

const (
	// Equal means the lines appear in both a and b.
	Equal Kind = iota
	// Insert means the lines appear only in b (added relative to a).
	Insert
	// Delete means the lines appear only in a (removed relative to b).
	Delete
)

// Hunk is a contiguous region produced by Diff.
type Hunk struct {
	Kind  Kind
	Lines []string
}

// Diff computes the shortest edit script between a and b using the Myers O(ND)
// algorithm and returns the result as a slice of Hunks. The union of Equal and
// Delete lines reconstructs a; the union of Equal and Insert lines reconstructs b.
func Diff(a, b []string) []Hunk {
	n := len(a)
	m := len(b)
	if n == 0 && m == 0 {
		return nil
	}

	script := editScript(a, b, n, m)

	var hunks []Hunk
	ia, ib := 0, 0

	emit := func(kind Kind, lines []string) {
		if len(hunks) > 0 && hunks[len(hunks)-1].Kind == kind {
			hunks[len(hunks)-1].Lines = append(hunks[len(hunks)-1].Lines, lines...)
			return
		}
		cp := make([]string, len(lines))
		copy(cp, lines)
		hunks = append(hunks, Hunk{Kind: kind, Lines: cp})
	}

	for _, op := range script {
		switch op {
		case 'e':
			emit(Equal, a[ia:ia+1])
			ia++
			ib++
		case 'd':
			emit(Delete, a[ia:ia+1])
			ia++
		case 'i':
			emit(Insert, b[ib:ib+1])
			ib++
		}
	}
	return hunks
}

// editScript runs the Myers algorithm and returns the op-code sequence.
func editScript(a, b []string, n, m int) []byte {
	max := n + m
	offset := max + 1
	v := make([]int, 2*max+3)

	// trace[d] = copy of v's active diagonals AFTER step d's forward pass.
	// trace[d] stores 2*d+1 values for diagonals k=-d..d.
	trace := make([][]int, 0, max+1)

	foundD := -1

	for d := 0; d <= max; d++ {
		// Forward pass for edit distance d.
		done := false
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
				x = v[offset+k+1]
			} else {
				x = v[offset+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[offset+k] = x
			if x >= n && y >= m {
				done = true
			}
		}
		// Snapshot the active diagonals after this pass.
		snap := make([]int, 2*d+1)
		for k := -d; k <= d; k++ {
			snap[k+d] = v[offset+k]
		}
		trace = append(trace, snap)

		if done {
			foundD = d
			break
		}
	}

	// Invariant: for any finite n+m the Myers forward pass reaches done=true
	// by d=n+m at the latest (pure delete-then-insert path). The loop above
	// runs d=0..max where max=n+m, so foundD is always set when we reach here.

	// getV reads V[k] from the snapshot at edit distance d.
	// d is always in [0, foundD] because the backtrack loop runs d=foundD..1
	// (condition d>0), so d-1 reaches 0 but never goes negative.
	getV := func(d, k int) int {
		idx := k + d
		if idx < 0 || idx >= len(trace[d]) {
			return -1
		}
		return trace[d][idx]
	}

	// Backtrack from (n,m) to (0,0).
	ops := make([]byte, 0, n+m)
	x, y := n, m

	for d := foundD; d > 0; d-- {
		k := x - y

		// Determine the previous diagonal using trace[d-1].
		var prevK int
		vm1 := getV(d-1, k-1)
		vp1 := getV(d-1, k+1)
		if k == -d || (k != d && vm1 < vp1) {
			prevK = k + 1 // came via insert (moved down)
		} else {
			prevK = k - 1 // came via delete (moved right)
		}

		prevX := getV(d-1, prevK)
		prevY := prevX - prevK

		// The edit step ends at (afterX, afterY); then a snake takes us to (x,y).
		var afterX int
		if prevK < k {
			// delete: moved right from (prevX,prevY) to (prevX+1,prevY).
			afterX = prevX + 1
		} else {
			// insert: moved down from (prevX,prevY) to (prevX,prevY+1).
			afterX = prevX
		}

		// Snake from (afterX,afterY) to (x,y) — all equal.
		for i := 0; i < x-afterX; i++ {
			ops = append(ops, 'e')
		}

		// Emit the edit.
		if prevK < k {
			ops = append(ops, 'd')
		} else {
			ops = append(ops, 'i')
		}

		x = prevX
		y = prevY
	}

	// Initial snake: (0,0) to (x,y) — all equal (from d=0 snake).
	for i := 0; i < x; i++ {
		ops = append(ops, 'e')
	}

	// Ops were built end-to-start; reverse.
	for i, j := 0, len(ops)-1; i < j; i, j = i+1, j-1 {
		ops[i], ops[j] = ops[j], ops[i]
	}
	return ops
}
