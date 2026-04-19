# diff3-go

Three-way text merge with Git-compatible conflict markers, written in Go.

**Status: alpha.** The API is stable for the documented surface, but may change before v1.0.

## Install

```
go get github.com/CivNode/diff3-go
```

Requires Go 1.22 or later (minimum declared in `go.mod`). No non-stdlib dependencies.

## Quick start

```go
package main

import (
    "fmt"

    diff3 "github.com/CivNode/diff3-go"
)

func main() {
    ancestor := "hello\nworld\n"
    a        := "hello\nGo world\n"
    b        := "hello\nworld\nfrom diff3\n"

    result, hadConflicts, err := diff3.Merge(ancestor, a, b, diff3.Options{})
    if err != nil {
        panic(err)
    }
    fmt.Printf("conflicts: %v\n%s", hadConflicts, result)
    // conflicts: false
    // hello
    // Go world
    // from diff3
}
```

## API

```go
// Merge performs a three-way merge of a and b against their common ancestor.
// Returns the merged text, whether any conflict markers were emitted, and any error.
func Merge(ancestor, a, b string, opts Options) (result string, hadConflicts bool, err error)

type Options struct {
    Mode           Mode   // LineAware (default) or CharacterAware
    MarkerLeft     string // default: "<<<<<<< "
    MarkerAncestor string // default: "======="
    MarkerRight    string // default: ">>>>>>>"
}

type Mode int
const (
    LineAware      Mode = iota // each line is the smallest merge unit
    CharacterAware             // secondary char-level diff resolves sub-line conflicts
)
```

**Clean merge** (only one side changed a region): the changed version is taken, no markers.

**Identical change** (both sides made the same edit): the edit is taken, no markers.

**Conflict** (both sides changed the same region differently): Git-style markers surround
the two versions:

```
<<<<<<< 
version from a
=======
version from b
>>>>>>>
```

### Modes

`LineAware` (default) treats each line as the smallest unit. Two edits to the same line
always produce a conflict marker.

`CharacterAware` runs a secondary character-level diff within conflicting line regions.
If the character-level changes are non-overlapping within the region, they merge cleanly.
Example: A changes the first word of a line, B changes the last word of the same line.
`LineAware` emits a conflict; `CharacterAware` merges cleanly.

### Custom markers

```go
result, _, _ := diff3.Merge(ancestor, a, b, diff3.Options{
    MarkerLeft:     "<<< OURS",
    MarkerAncestor: "|||||||",
    MarkerRight:    ">>> THEIRS",
})
```

## Benchmarks

Measured on an AMD Ryzen 9 3950X (amd64, Go 1.22):

```
BenchmarkMerge10KB-32     84654     69542 ns/op    108.71 MB/s   130392 B/op      369 allocs/op
BenchmarkMerge100KB-32     6870    920890 ns/op     83.50 MB/s  1635010 B/op     2753 allocs/op
BenchmarkMerge2MB-32        158  39127031 ns/op     53.60 MB/s 68615110 B/op    31968 allocs/op
```

The 2 MB benchmark uses a true 2.000 MB file (34 562 lines) with a conflict every 50 lines.
Time complexity is O(N * D) where D is the edit distance; performance degrades with
denser edits.

Run the benchmarks yourself:

```
go test -bench=. -benchmem ./...
```

## Current limitations

- **CRLF line endings are treated as distinct from LF.** A line ending in `\r\n` and the
  same line ending in `\n` are different strings and will conflict. Normalize line endings
  before calling `Merge` if you need platform-transparent merging.
- **No whitespace-ignore mode.** There is no equivalent to `diff -w`. If whitespace
  differences matter, use `LineAware` mode (default).
- **Performance degrades on high-conflict inputs.** When both sides change many lines
  relative to the ancestor, the Myers O(N*D) algorithm becomes slow. For inputs where
  most lines on both sides differ from the ancestor, the 2 MB case above may take
  longer than 20 ms.
- **`CharacterAware` mode is not recursive.** Character-level merge uses the same
  algorithm as line-level; it does not descend further on conflicting characters.

## Contributing

Issues and pull requests welcome. Please include tests for any change. Run
`go test ./...` before submitting.

## License

MIT. See [LICENSE](LICENSE).
