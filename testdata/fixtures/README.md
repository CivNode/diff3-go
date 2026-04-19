# Test fixtures

Each subdirectory is a golden-file test case loaded by `TestMerge_Fixtures` in
`merge_test.go`. Every directory must contain:

- `ancestor.txt` - the common ancestor
- `a.txt` - version A
- `b.txt` - version B
- `expected.txt` - the expected merge output
- `expected_has_conflicts.txt` - `true` or `false`

Character-aware fixtures (prefix `char-aware-`) are skipped by
`TestMerge_Fixtures` and tested separately in `charaware_test.go`.

## CRLF fixtures

The `crlf-mix` fixture contains real CRLF bytes (`\r\n`) in `b.txt` and
`expected.txt`. A `.gitattributes` rule at the repo root (`testdata/fixtures/**
-text`) prevents git from normalizing these to LF on checkout. If you add a
fixture that relies on specific byte sequences (CRLF, NUL, non-UTF-8), the same
`-text` rule covers it automatically.

The programmatic CRLF test in `crlf_test.go` covers the same scenario in code
for environments where git attributes cannot be guaranteed (e.g., older CI
images that ignore `.gitattributes`).
