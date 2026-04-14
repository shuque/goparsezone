package zonefile

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// IncludeRecord determines whether a record passes all active filters.
// Returns true if the record should be included.
func IncludeRecord(record *Record, filters *FilterConfig, zoneOrigin string, warnWriter io.Writer) bool {
	if filters == nil {
		return true
	}

	if filters.NoDNSSEC && dnssecTypes[record.Type] {
		return false
	}

	if filters.RRTypes != "" {
		allowed := make(map[string]bool)
		for _, rt := range strings.Split(filters.RRTypes, ",") {
			allowed[strings.ToUpper(strings.TrimSpace(rt))] = true
		}
		if !allowed[record.Type] {
			return false
		}
	}

	if filters.IncludeName != "" {
		if filters.Regex {
			matched, err := regexp.MatchString("(?i)"+filters.IncludeName, record.Name)
			if err != nil {
				if warnWriter != nil {
					fmt.Fprintf(warnWriter, "Warning: Invalid regex pattern '%s': %v\n", filters.IncludeName, err)
				}
				return false
			}
			if !matched {
				return false
			}
		} else {
			if !strings.Contains(strings.ToLower(record.Name), strings.ToLower(filters.IncludeName)) {
				return false
			}
		}
	}

	if filters.ExcludeName != "" {
		if filters.Regex {
			matched, err := regexp.MatchString("(?i)"+filters.ExcludeName, record.Name)
			if err != nil {
				if warnWriter != nil {
					fmt.Fprintf(warnWriter, "Warning: Invalid regex pattern '%s': %v\n", filters.ExcludeName, err)
				}
				return false
			}
			if matched {
				return false
			}
		} else {
			if strings.Contains(strings.ToLower(record.Name), strings.ToLower(filters.ExcludeName)) {
				return false
			}
		}
	}

	if filters.IncludeData != "" {
		if filters.Regex {
			matched, err := regexp.MatchString("(?i)"+filters.IncludeData, record.Data)
			if err != nil {
				if warnWriter != nil {
					fmt.Fprintf(warnWriter, "Warning: Invalid regex pattern '%s': %v\n", filters.IncludeData, err)
				}
				return false
			}
			if !matched {
				return false
			}
		} else {
			if !strings.Contains(strings.ToLower(record.Data), strings.ToLower(filters.IncludeData)) {
				return false
			}
		}
	}

	if filters.ExcludeData != "" {
		if filters.Regex {
			matched, err := regexp.MatchString("(?i)"+filters.ExcludeData, record.Data)
			if err != nil {
				if warnWriter != nil {
					fmt.Fprintf(warnWriter, "Warning: Invalid regex pattern '%s': %v\n", filters.ExcludeData, err)
				}
				return false
			}
			if matched {
				return false
			}
		} else {
			if strings.Contains(strings.ToLower(record.Data), strings.ToLower(filters.ExcludeData)) {
				return false
			}
		}
	}

	if filters.Wildcard && !strings.HasPrefix(record.Name, "*.") {
		return false
	}

	if filters.Delegations {
		if record.Type != "NS" {
			return false
		}
		if zoneOrigin != "" && record.Name == zoneOrigin {
			return false
		}
	}

	if filters.TTLMin != nil || filters.TTLMax != nil {
		if record.TTL == nil {
			return false
		}
		if filters.TTLMin != nil && *record.TTL < *filters.TTLMin {
			return false
		}
		if filters.TTLMax != nil && *record.TTL > *filters.TTLMax {
			return false
		}
	}

	if filters.ClassFilter != "" && !strings.EqualFold(record.Class, filters.ClassFilter) {
		return false
	}

	if filters.MinLabelCount != nil || filters.MaxLabelCount != nil {
		lc := labelCount(record.Name)
		if filters.MinLabelCount != nil && lc < *filters.MinLabelCount {
			return false
		}
		if filters.MaxLabelCount != nil && lc > *filters.MaxLabelCount {
			return false
		}
	}

	return true
}

// labelCount returns the number of DNS labels in name.
// The root "." has 1 label. "example.com." has 3 labels.
func labelCount(name string) int {
	if name == "." {
		return 1
	}
	trimmed := strings.TrimRight(name, ".")
	return len(strings.Split(trimmed, ".")) + 1
}
