# Design: Reimplement parse_zone as a Go program (goparsezone)

## Context

The Python DNS zone file parser at `/Users/shuque/git/parse_zone/parse_zone.py` (~580 lines) is a single-file tool that parses RFC 1035 master zone files with filtering, statistics, and formatted output. The goal is to reimplement it in Go at `/Users/shuque/git/goparsezone/` as both a **reusable library package** and a **CLI tool**, matching all features while using idiomatic Go patterns. Zero external dependencies (stdlib only), matching the Python's approach.

## Project Layout

```
goparsezone/
  LICENSE                              (exists)
  go.mod                               (new - module github.com/shuque/goparsezone)
  README.md                            (new)
  Design.md                            (this file)
  zonefile/                            (new - library package)
    types.go                           (Record, FilterConfig, ParseResult, constants)
    ttl.go                             (ParseTTL)
    comment.go                         (findComment - unexported helper)
    filter.go                          (IncludeRecord, labelCount)
    parse.go                           (Parse, ParseFile - core state machine)
    output.go                          (PrintRecords, PrintStatistics)
    parse_test.go                      (integration tests with zone files)
    ttl_test.go                        (table-driven TTL parsing tests)
    filter_test.go                     (filter logic tests)
    testdata/
      example.zone                     (copied from parse_zone)
      example2.zone                    (copied from parse_zone)
  cmd/
    goparsezone/
      main.go                          (CLI: flag parsing, wires library calls)
```

## Key Design Decisions

### 1. Library vs CLI separation
- `zonefile` package is the reusable library with all parsing/filtering/output logic
- `cmd/goparsezone/main.go` is a thin CLI that maps flags to library calls
- All output functions take `io.Writer` for testability

### 2. Core types (`zonefile/types.go`)

```go
type Record struct {
    Name  string
    TTL   *int     // nil = not specified (matches Python's None)
    Class string
    Type  string
    Data  string
    Line  int
}

type FilterConfig struct {
    NoDNSSEC      bool
    RRTypes       string   // comma-separated
    IncludeName   string
    IncludeData   string
    ExcludeName   string
    ExcludeData   string
    Wildcard      bool
    Delegations   bool
    TTLMin        *int     // nil = not set
    TTLMax        *int
    ClassFilter   string
    Regex         bool
    MinLabelCount *int
    MaxLabelCount *int
}

type ParseResult struct {
    Records      []Record
    SkippedLines int
    ZoneOrigin   string
}
```

### 3. Parser API (`zonefile/parse.go`)

```go
func Parse(r io.Reader, filters *FilterConfig, warnWriter io.Writer) (*ParseResult, error)
func ParseFile(filepath string, filters *FilterConfig, warnWriter io.Writer) (*ParseResult, error)
```

- `io.Reader` based core - works with files, stdin, strings, buffers
- `warnWriter` for warnings (CLI passes os.Stderr, tests pass io.Discard or bytes.Buffer)
- `ParseFile` is a thin convenience wrapper
- Set `bufio.Scanner` buffer to 1MB to handle large TXT records

### 4. Parser state machine (matching Python exactly)

State variables: `currentOrigin`, `currentTTL`, `zoneOrigin`, `firstSOAFound`, `previousName`, `inParens`, `accumulatedLine`, `accumulatedLineNum`, `accumulatedStartsWithSpace`

Line processing pipeline:
1. Strip comments via `findComment()` (respects double-quoted strings)
2. Skip blank lines (count as skipped unless inside parens)
3. Multi-line accumulation (parentheses)
4. Directive handling: `$ORIGIN`, `$TTL`, `$INCLUDE` (warn+skip), other `$` (skip)
5. Record parsing: owner name (explicit or inherited), TTL/class in either order, type, data
6. Name expansion: `@` -> origin, relative names get `.origin` appended
7. Zone origin detection from first SOA if no `$ORIGIN`
8. Filter application, add to results or increment skipped count

### 5. Filter logic (`zonefile/filter.go`)

```go
func IncludeRecord(record *Record, filters *FilterConfig, zoneOrigin string, warnWriter io.Writer) bool
```

All filters ANDed. Matches Python behavior exactly:
- NoDNSSEC, RRTypes, IncludeName/Data, ExcludeName/Data (substring or regex), Wildcard, Delegations, TTL range, ClassFilter, LabelCount range

### 6. Output (`zonefile/output.go`)

```go
func PrintRecords(w io.Writer, records []Record)
func PrintStatistics(w io.Writer, result *ParseResult)
```

Matching Python's column format for records and statistics layout (RR counts, RRsets, wildcards, delegations, per-type breakdown with percentages).

### 7. CLI (`cmd/goparsezone/main.go`)

Flag names match the Python: `--printrecords`, `--stats`, `--no-dnssec`, `--rrtypes`, `--includename`, `--includedata`, `--excludename`, `--excludedata`, `--regex`, `--wildcard`, `--delegations`, `--ttl-min`, `--ttl-max`, `--class`, `--minlabelcount`, `--maxlabelcount`, `--version`

Uses `flag.Visit` after `flag.Parse()` to detect which int flags were explicitly set (to distinguish "not set" from zero value), populating `*int` pointer fields accordingly.

Positional arg: `flag.Arg(0)` for zone file path; if empty, reads stdin.

## Implementation Order

1. `go.mod` - initialize module
2. `zonefile/types.go` - types and constants
3. `zonefile/ttl.go` + `zonefile/ttl_test.go` - self-contained, testable immediately
4. `zonefile/comment.go` - small helper
5. `zonefile/filter.go` + `zonefile/filter_test.go` - depends on types
6. Copy test zone files to `zonefile/testdata/`
7. `zonefile/parse.go` + `zonefile/parse_test.go` - core parser, depends on all above
8. `zonefile/output.go` - depends on types
9. `cmd/goparsezone/main.go` - CLI wiring
10. `README.md`

## Verification

1. `go build ./...` - compiles cleanly
2. `go test ./...` - all tests pass
3. `go vet ./...` - no issues
4. Manual comparison against Python output:
   - `./goparsezone --printrecords --stats zonefile/testdata/example.zone`
   - `./goparsezone --printrecords --stats zonefile/testdata/example2.zone`
   - `cat zonefile/testdata/example.zone | ./goparsezone --printrecords --stats`
   - Compare output with: `python3 /Users/shuque/git/parse_zone/parse_zone.py --printrecords --stats` on same files
   - Test filters: `--no-dnssec`, `--rrtypes A,AAAA`, `--wildcard`, `--delegations`, `--ttl-min`, `--ttl-max`
