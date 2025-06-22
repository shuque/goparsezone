# DNS Zone Parser

A Go program that parses DNS master file format zone files and displays their contents.

Written with the help of Cursor, mainly to evaluate its ability to generate working Go code.


## Features

- Parses standard DNS master file format
- Handles $ORIGIN and $TTL directives
- Supports quoted strings in RDATA
- Handles multi-line records (like SOA)
- Supports TTL values with time units (S, M, H, D, W)
- Proper error handling with line numbers

## Usage

```bash
go run main.go <zone-file>
```

Or build and run:

```bash
go build -o dns-parser main.go
./dns-parser <zone-file>
```

## Example

```bash
go run main.go example.zone
```

This will parse the `example.zone` file and display all DNS records in a structured format.

## Zone File Format

The program expects standard DNS master file format:

```
$ORIGIN example.com.
$TTL 3600

@       IN      SOA     ns1.example.com. admin.example.com. (
                2023120101      ; serial
                3600            ; refresh
                1800            ; retry
                1209600         ; expire
                3600            ; minimum TTL
                )

@       IN      NS      ns1.example.com.
www     IN      A       192.168.1.20
```

## Error Handling

The program provides detailed error messages including:
- File not found errors
- Parsing errors with line numbers
- Invalid record format errors

## Requirements

- Go 1.16 or later 