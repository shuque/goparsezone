package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/shuque/goparsezone/zonefile"
)

func main() {
	var (
		printRecords  bool
		stats         bool
		noDNSSEC      bool
		rrTypes       string
		includeName   string
		includeData   string
		excludeName   string
		excludeData   string
		regex         bool
		wildcard      bool
		delegations   bool
		ttlMin        int
		ttlMax        int
		classFilter   string
		minLabelCount int
		maxLabelCount int
		showVersion   bool
	)

	flag.BoolVar(&printRecords, "printrecords", false, "Print the DNS records in the zone")
	flag.BoolVar(&stats, "stats", false, "Print DNS record type statistics")
	flag.BoolVar(&noDNSSEC, "no-dnssec", false, "Exclude DNSSEC-related records")
	flag.StringVar(&rrTypes, "rrtypes", "", "Comma-separated list of record types to include")
	flag.StringVar(&includeName, "includename", "", "Include records with names containing string")
	flag.StringVar(&includeData, "includedata", "", "Include records with data containing string")
	flag.StringVar(&excludeName, "excludename", "", "Exclude records with names containing string")
	flag.StringVar(&excludeData, "excludedata", "", "Exclude records with data containing string")
	flag.BoolVar(&regex, "regex", false, "Treat filter values as regex patterns")
	flag.BoolVar(&wildcard, "wildcard", false, "Only process wildcard DNS records")
	flag.BoolVar(&delegations, "delegations", false, "Only process delegation NS records")
	flag.IntVar(&ttlMin, "ttl-min", 0, "Minimum TTL value (inclusive)")
	flag.IntVar(&ttlMax, "ttl-max", 0, "Maximum TTL value (inclusive)")
	flag.StringVar(&classFilter, "class", "", "Filter records by class")
	flag.IntVar(&minLabelCount, "minlabelcount", 0, "Minimum label count in owner name")
	flag.IntVar(&maxLabelCount, "maxlabelcount", 0, "Maximum label count in owner name")
	flag.BoolVar(&showVersion, "version", false, "Show version")

	flag.Parse()

	if showVersion {
		fmt.Printf("goparsezone %s\n", zonefile.Version)
		os.Exit(0)
	}

	// Detect which int flags were explicitly set
	flagsSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagsSet[f.Name] = true
	})

	// Build filter config
	filters := &zonefile.FilterConfig{
		NoDNSSEC:    noDNSSEC,
		RRTypes:     rrTypes,
		IncludeName: includeName,
		IncludeData: includeData,
		ExcludeName: excludeName,
		ExcludeData: excludeData,
		Wildcard:    wildcard,
		Delegations: delegations,
		ClassFilter: classFilter,
		Regex:       regex,
	}

	if flagsSet["ttl-min"] {
		filters.TTLMin = &ttlMin
	}
	if flagsSet["ttl-max"] {
		filters.TTLMax = &ttlMax
	}
	if flagsSet["minlabelcount"] {
		filters.MinLabelCount = &minLabelCount
	}
	if flagsSet["maxlabelcount"] {
		filters.MaxLabelCount = &maxLabelCount
	}

	// Validate
	if filters.MinLabelCount != nil && filters.MaxLabelCount != nil &&
		*filters.MinLabelCount > *filters.MaxLabelCount {
		fmt.Fprintln(os.Stderr, "Error: --minlabelcount cannot be greater than --maxlabelcount.")
		os.Exit(1)
	}

	// Parse zone file
	var result *zonefile.ParseResult
	var err error

	zoneFile := flag.Arg(0)
	if zoneFile != "" {
		info, statErr := os.Stat(zoneFile)
		if statErr != nil || info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: '%s' is not a file.\n", zoneFile)
			os.Exit(1)
		}
		result, err = zonefile.ParseFile(zoneFile, filters, os.Stderr)
	} else {
		result, err = zonefile.Parse(os.Stdin, filters, os.Stderr)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if printRecords {
		zonefile.PrintRecords(os.Stdout, result.Records)
	}

	if stats {
		if printRecords {
			fmt.Println()
		}
		zonefile.PrintStatistics(os.Stdout, result)
	}
}
