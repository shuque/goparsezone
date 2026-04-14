# goparsezone

A Go library and command-line tool for parsing DNS master zone files
(RFC 1035 format).

This is a Go reimplementation of [parse_zone](https://github.com/shuque/parse_zone),
a Python DNS zone file parser.

## Features

- Parses RFC 1035 master zone file format
- Handles `$ORIGIN`, `$TTL`, and `$INCLUDE` directives
- Supports multi-line records (parenthesized continuation)
- BIND-style TTL suffixes (s, m, h, d, w) including compound values like `1h30m`
- Record filtering by type, name, data, class, TTL range, label count, wildcards, and delegations
- Substring and regex matching for name/data filters
- Zone statistics with RRset counting and per-type breakdowns
- Reads from file or stdin
- Zero external dependencies (Go stdlib only)

## Installation

```bash
go install github.com/shuque/goparsezone/cmd/goparsezone@latest
```

This installs the `goparsezone` binary into `$GOBIN` if set, otherwise
`$GOPATH/bin`, otherwise `$HOME/go/bin`. You can check the target
directory with:

```bash
go env GOBIN
go env GOPATH
```

Ensure the install directory is in your `$PATH` so the binary is
runnable without a full path:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

To build the binary in the current working directory for testing:

```bash
go build ./cmd/goparsezone
```

## Usage

```bash
# Parse a zone file and print records
goparsezone --printrecords example.zone

# Print statistics
goparsezone --stats example.zone

# Both records and statistics
goparsezone --printrecords --stats example.zone

# Read from stdin
cat example.zone | goparsezone --printrecords --stats

# Filter by record type
goparsezone --printrecords --rrtypes A,AAAA example.zone

# Exclude DNSSEC records
goparsezone --printrecords --stats --no-dnssec signed.zone

# Show only wildcard records
goparsezone --printrecords --wildcard example.zone

# Show only delegation NS records
goparsezone --printrecords --delegations example.zone

# Filter by TTL range
goparsezone --printrecords --ttl-min 300 --ttl-max 3600 example.zone

# Filter by name (substring match)
goparsezone --printrecords --includename www example.zone

# Filter by name (regex match)
goparsezone --printrecords --regex --includename "^www\." example.zone
```

## Library Usage

The `zonefile` package can be imported for use in other Go programs:

```go
import "github.com/shuque/goparsezone/zonefile"

// Parse from a file
result, err := zonefile.ParseFile("example.zone", nil, os.Stderr)

// Parse from an io.Reader
result, err := zonefile.Parse(reader, nil, os.Stderr)

// Parse with filters
filters := &zonefile.FilterConfig{
    RRTypes:  "A,AAAA",
    NoDNSSEC: true,
}
result, err := zonefile.ParseFile("example.zone", filters, os.Stderr)

// Access results
for _, record := range result.Records {
    fmt.Printf("%s %s %s\n", record.Name, record.Type, record.Data)
}
```

## CLI Options

```
Usage: goparsezone [options] [zonefile]

Options:
  --printrecords       Print the DNS records in the zone
  --stats              Print DNS record type statistics
  --no-dnssec          Exclude DNSSEC-related records
  --rrtypes TYPES      Comma-separated list of record types to include
  --includename NAME   Include records with names containing string
  --includedata DATA   Include records with data containing string
  --excludename NAME   Exclude records with names containing string
  --excludedata DATA   Exclude records with data containing string
  --regex              Treat filter values as regex patterns
  --wildcard           Only process wildcard DNS records
  --delegations        Only process delegation NS records
  --ttl-min TTL        Minimum TTL value (inclusive)
  --ttl-max TTL        Maximum TTL value (inclusive)
  --class CLASS        Filter records by class
  --minlabelcount N    Minimum label count in owner name
  --maxlabelcount N    Maximum label count in owner name
  --version            Show version
```

## License

MIT License. See [LICENSE](LICENSE) for details.

## Author

Shumon Huque
