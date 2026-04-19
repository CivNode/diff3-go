// Package myers implements the Myers line-level diff algorithm.
//
// The algorithm computes the shortest edit script (SES) between two slices of
// strings and returns a sequence of hunks classifying each region as Equal,
// Insert, or Delete. The implementation follows the original O(ND) paper by
// Eugene W. Myers (1986).
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

	// editScript returns a sequence of ops: 'e' (equal), 'd' (delete), 'i' (insert).
	// Each op is applied once to move through (a,b) in lock-step.
	script := editScript(a, b)

	var hunks []Hunk
	ia, ib := 0, 0

	emit := func(kind Kind, lines []string) {
		if len(lines) == 0 {
			return
		}
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

// editScript runs Myers and returns a flat op-code slice.
func editScript(a, b []string) []byte {
	n := len(a)
	m := len(b)
	max := n + m + 1

	// v[k+max] = furthest-right x on diagonal k.
	v := make([]int, 2*max+1)

	// trace[d] stores a copy of v after each edit distance d is explored.
	trace := make([][]int, 0, n+m+1)

	for d := 0; d <= n+m; d++ {
		snap := make([]int, len(v))
		copy(snap, v)
		trace = append(trace, snap)

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[max+k-1] < v[max+k+1]) {
				x = v[max+k+1]
			} else {
				x = v[max+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[max+k] = x
			if x >= n && y >= m {
				// Backtrack from (n,m) through the trace to build the script.
				return backtrack(trace, a, b, n, m, max)
			}
		}
	}
	// Unreachable for finite inputs.
	return nil
}

// backtrack reconstructs the edit script by walking backwards through the trace.
func backtrack(trace [][]int, a, b []string, x, y, max int) []byte {
	// We'll build the script in reverse and flip at the end.
	var rev []byte

	for d := len(trace) - 1; d > 0; d-- {
		snap := trace[d]
		k := x - y

		var prevK int
		if k == -d || (k != d && snap[max+k-1] < snap[max+k+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}
		prevX := snap[max+prevK]
		prevY := prevX - prevK

		// Snake: from (prevX,prevY) to wherever the edit step ended, then diagonal.
		// The edit step goes from (prevX,prevY) → one step right or down.
		// Then the snake carries us from that point to (x,y).

		// Emit snake steps (equal) from bottom of snake back to start of snake.
		// The snake start is at (prevX+1, prevY) [delete] or (prevX, prevY+1) [insert].
		var snakeStartX, snakeStartY int
		if prevK == k-1 {
			// We moved right (delete) from (prevX,prevY) to (prevX+1,prevY).
			snakeStartX = prevX + 1
			snakeStartY = prevY
		} else {
			// We moved down (insert) from (prevX,prevY) to (prevX,prevY+1).
			snakeStartX = prevX
			snakeStartY = prevY + 1
		}

		// Emit equal steps from (x,y) back to (snakeStartX,snakeStartY).
		for x > snakeStartX && y > snakeStartY {
			x--
			y--
			rev = append(rev, 'e')
		}

		// Emit the edit step.
		if prevK == k-1 {
			// delete step
			x = prevX
			rev = append(rev, 'd')
		} else {
			// insert step
			y = prevY
			rev = append(rev, 'i')
		}
	}

	// Any remaining movement from (x,y) back to (0,0) is all equal (initial snake).
	for x > 0 && y > 0 {
		x--
		y--
		rev = append(rev, 'e')
	}

	// Reverse the script.
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev
}
