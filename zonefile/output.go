package zonefile

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// PrintRecords writes DNS records in a formatted table to w.
func PrintRecords(w io.Writer, records []Record) {
	if len(records) == 0 {
		fmt.Fprintln(w, "No DNS records found in the zone file.")
		return
	}

	for _, r := range records {
		ttl := "N/A"
		if r.TTL != nil {
			ttl = fmt.Sprintf("%d", *r.TTL)
		}
		cls := r.Class
		if cls == "" {
			cls = "N/A"
		}
		rtype := r.Type
		if rtype == "" {
			rtype = "N/A"
		}
		fmt.Fprintf(w, "%-30s %-8s %-4s %-8s %s\n", r.Name, ttl, cls, rtype, r.Data)
	}
}

// PrintStatistics writes zone statistics to w.
func PrintStatistics(w io.Writer, result *ParseResult) {
	fmt.Fprintln(w, "### DNS Zone Statistics:")

	if result.ZoneOrigin != "" {
		fmt.Fprintf(w, "### Zone: %s\n\n", result.ZoneOrigin)
	}

	if len(result.Records) == 0 {
		fmt.Fprintln(w, "No DNS records found to analyze.")
		if result.SkippedLines > 0 {
			fmt.Fprintf(w, "Lines skipped during parsing: %d\n", result.SkippedLines)
		}
		return
	}

	fmt.Fprintf(w, "%-12s %8d\n", "Records:", len(result.Records))
	if result.SkippedLines > 0 {
		fmt.Fprintf(w, "Lines skipped during parsing: %d\n", result.SkippedLines)
	}

	// Calculate RRsets (records with same name, class, and type)
	type rrsetKey struct {
		name  string
		class string
		rtype string
	}
	rrsets := make(map[rrsetKey]int)
	for _, r := range result.Records {
		key := rrsetKey{r.Name, r.Class, r.Type}
		rrsets[key]++
	}

	fmt.Fprintf(w, "%-12s %8d\n", "RRsets:", len(rrsets))

	// Count distinct names
	distinctNames := make(map[string]bool)
	for _, r := range result.Records {
		distinctNames[r.Name] = true
	}
	fmt.Fprintf(w, "%-12s %8d\n", "Names:", len(distinctNames))

	// Count wildcard records
	wildcardCount := 0
	for _, r := range result.Records {
		if strings.HasPrefix(r.Name, "*.") {
			wildcardCount++
		}
	}
	if wildcardCount > 0 {
		fmt.Fprintf(w, "%-12s %8d\n", "Wildcards:", wildcardCount)
	}

	// Count delegation records (unique names for NS records not at zone origin)
	delegationNames := make(map[string]bool)
	for _, r := range result.Records {
		if r.Type == "NS" && r.Name != result.ZoneOrigin {
			delegationNames[r.Name] = true
		}
	}
	if len(delegationNames) > 0 {
		fmt.Fprintf(w, "%-12s %8d\n", "Delegations:", len(delegationNames))
	}

	// Count record types
	recordTypes := make(map[string]int)
	for _, r := range result.Records {
		recordTypes[r.Type]++
	}

	// Count RRsets by type
	rrsetsByType := make(map[string]int)
	for key := range rrsets {
		rrsetsByType[key.rtype]++
	}

	// Sort type names
	typeNames := make([]string, 0, len(recordTypes))
	for t := range recordTypes {
		typeNames = append(typeNames, t)
	}
	sort.Strings(typeNames)

	fmt.Fprintf(w, "\nRecord type statistics:\n")
	fmt.Fprintf(w, "%-10s %8s %11s %9s %9s\n", "Type", "RR", "RR%", "RRsets", "RRset%")
	fmt.Fprintln(w, strings.Repeat("-", 51))

	totalRecords := len(result.Records)
	totalRRsets := len(rrsets)
	for _, rtype := range typeNames {
		count := recordTypes[rtype]
		pct := float64(count) / float64(totalRecords) * 100
		rrsetCount := rrsetsByType[rtype]
		var rrsetPct float64
		if totalRRsets > 0 {
			rrsetPct = float64(rrsetCount) / float64(totalRRsets) * 100
		}
		fmt.Fprintf(w, "%-10s %8d %10.1f%% %9d %8.1f%%\n", rtype, count, pct, rrsetCount, rrsetPct)
	}
}
