// Package zonefile parses DNS master zone format files (RFC 1035).
package zonefile

// Version is the version of this library.
const Version = "0.1.0"

// Record represents a single parsed DNS resource record.
type Record struct {
	Name  string // Owner name (fully qualified or relative)
	TTL   *int   // TTL in seconds; nil if not specified and no $TTL default
	Class string // DNS class (IN, CH, etc.)
	Type  string // Record type (A, AAAA, MX, etc.)
	Data  string // Record data as a space-joined string
	Line  int    // Source line number
}

// FilterConfig specifies criteria for including/excluding records.
// All active filters are ANDed together.
type FilterConfig struct {
	NoDNSSEC      bool   // Exclude DNSSEC-related record types
	RRTypes       string // Comma-separated list of types to include
	IncludeName   string // Include records whose name contains this (substring or regex)
	IncludeData   string // Include records whose data contains this (substring or regex)
	ExcludeName   string // Exclude records whose name contains this (substring or regex)
	ExcludeData   string // Exclude records whose data contains this (substring or regex)
	Wildcard      bool   // Include only wildcard records (name starts with "*.")
	Delegations   bool   // Include only delegation NS records (not at zone apex)
	TTLMin        *int   // Minimum TTL value (inclusive)
	TTLMax        *int   // Maximum TTL value (inclusive)
	ClassFilter   string // Include only records of this class
	Regex         bool   // Treat name/data filters as regex patterns
	MinLabelCount *int   // Minimum label count in owner name
	MaxLabelCount *int   // Maximum label count in owner name
}

// ParseResult holds the output of parsing a zone file.
type ParseResult struct {
	Records      []Record // Parsed (and filtered) records
	SkippedLines int      // Number of lines skipped
	ZoneOrigin   string   // Detected zone origin
}

// dnssecTypes lists DNSSEC record types excluded by the NoDNSSEC filter.
var dnssecTypes = map[string]bool{
	"DNSKEY":     true,
	"DS":         true,
	"NSEC3PARAM": true,
	"NSEC3":      true,
	"NSEC":       true,
	"RRSIG":      true,
}

// dnsClasses lists recognized DNS classes.
var dnsClasses = map[string]bool{
	"IN":  true,
	"CH":  true,
	"CS":  true,
	"HS":  true,
	"ANY": true,
}
