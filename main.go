package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// DNSRecord represents a DNS record
type DNSRecord struct {
	Name     string
	TTL      uint32
	Class    string
	Type     string
	RData    string
	Origin   string
	Comments []string
}

// Zone represents a DNS zone with its records
type Zone struct {
	Origin     string
	DefaultTTL uint32
	Records    []DNSRecord
}

// ParseZone parses a DNS master file format zone
func ParseZone(filename string) (*Zone, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	zone := &Zone{
		Records: make([]DNSRecord, 0),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var currentName string
	var currentTTL uint32
	var currentClass string
	var multiLineRecord string
	var inMultiLine bool
	var multiLineRecordType string

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		// Handle $ORIGIN directive
		if strings.HasPrefix(line, "$ORIGIN") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				zone.Origin = parts[1]
			}
			continue
		}

		// Handle $TTL directive
		if strings.HasPrefix(line, "$TTL") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if ttl, err := parseTTL(parts[1]); err == nil {
					zone.DefaultTTL = ttl
				}
			}
			continue
		}

		// Handle multi-line records
		if inMultiLine {
			// Remove trailing comments from this line
			if idx := strings.Index(line, ";"); idx != -1 {
				line = strings.TrimSpace(line[:idx])
			}
			multiLineRecord += " " + line
			if strings.Contains(line, ")") {
				// End of multi-line record
				// Clean up the RDATA: remove parentheses and normalize whitespace
				cleanedRData := strings.ReplaceAll(multiLineRecord, "(", "")
				cleanedRData = strings.ReplaceAll(cleanedRData, ")", "")
				// Normalize whitespace (replace multiple spaces with single space)
				re := regexp.MustCompile(`\s+`)
				cleanedRData = re.ReplaceAllString(cleanedRData, " ")
				cleanedRData = strings.TrimSpace(cleanedRData)

				// Remove the name, class, and type from the start of the RDATA
				fields := strings.Fields(cleanedRData)
				toRemove := 1 // name
				if len(fields) > toRemove && isClass(fields[toRemove]) {
					toRemove++ // class
				}
				if len(fields) > toRemove {
					toRemove++ // type
				}
				if len(fields) > toRemove {
					cleanedRData = strings.Join(fields[toRemove:], " ")
				}

				// Create the record with cleaned RDATA
				record := &DNSRecord{
					Name:   currentName,
					TTL:    currentTTL,
					Class:  currentClass,
					Origin: zone.Origin,
					RData:  cleanedRData,
					Type:   multiLineRecordType,
				}

				zone.Records = append(zone.Records, *record)
				multiLineRecord = ""
				inMultiLine = false
			}
			continue
		}

		// Check if this line starts a multi-line record
		if strings.Contains(line, "(") {
			// Parse the first line to get name, TTL, class, and type
			fields := strings.Fields(line)
			fieldIndex := 0

			// First field is the name
			if fields[fieldIndex] != "" {
				name := fields[fieldIndex]
				if name == "@" && zone.Origin != "" {
					name = zone.Origin
				} else if !strings.HasSuffix(name, ".") && zone.Origin != "" {
					name = name + "." + zone.Origin
				}
				currentName = name
			}
			fieldIndex++

			// Check if second field is a TTL
			if fieldIndex < len(fields) {
				if ttlValue, err := strconv.ParseUint(fields[fieldIndex], 10, 32); err == nil {
					currentTTL = uint32(ttlValue)
					fieldIndex++
				}
			}

			// Check if next field is a class
			if fieldIndex < len(fields) {
				if isClass(fields[fieldIndex]) {
					currentClass = fields[fieldIndex]
					fieldIndex++
				}
			}

			// Next field should be the record type
			if fieldIndex < len(fields) {
				recordType := strings.ToUpper(fields[fieldIndex])
				// Store the record type for later use
				multiLineRecord = line
				inMultiLine = true
				multiLineRecordType = recordType
				continue
			}
		}

		// Parse single-line DNS record
		record, err := parseRecord(line, currentName, currentTTL, currentClass, zone.Origin)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		// Update current values for next record
		if record.Name != "" {
			currentName = record.Name
		}
		if record.TTL != 0 {
			currentTTL = record.TTL
		}
		if record.Class != "" {
			currentClass = record.Class
		}

		zone.Records = append(zone.Records, *record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return zone, nil
}

// parseRecord parses a single DNS record line
func parseRecord(line, currentName string, currentTTL uint32, currentClass, origin string) (*DNSRecord, error) {
	// Split line into fields, preserving quoted strings
	fields := splitFields(line)
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid record format")
	}

	record := &DNSRecord{
		Origin: origin,
	}

	// Determine field positions based on content
	var name, ttl, class, recordType, rdata string
	fieldIndex := 0

	// First field is always the name (or empty for continuation)
	if fields[fieldIndex] != "" {
		name = fields[fieldIndex]
		if name == "@" && origin != "" {
			name = origin
		} else if !strings.HasSuffix(name, ".") && origin != "" {
			name = name + "." + origin
		}
	} else {
		name = currentName
	}
	fieldIndex++

	// Check if second field is a TTL (numeric)
	if fieldIndex < len(fields) {
		if _, err := strconv.ParseUint(fields[fieldIndex], 10, 32); err == nil {
			ttl = fields[fieldIndex]
			fieldIndex++
		}
	}

	// Check if next field is a class
	if fieldIndex < len(fields) {
		if isClass(fields[fieldIndex]) {
			class = fields[fieldIndex]
			fieldIndex++
		}
	}

	// Next field should be the record type
	if fieldIndex < len(fields) {
		recordType = strings.ToUpper(fields[fieldIndex])
		fieldIndex++
	} else {
		return nil, fmt.Errorf("missing record type")
	}

	// Remaining fields form the RDATA
	if fieldIndex < len(fields) {
		rdata = strings.Join(fields[fieldIndex:], " ")
	}

	// Set record fields
	record.Name = name
	if ttl != "" {
		if ttlValue, err := strconv.ParseUint(ttl, 10, 32); err == nil {
			record.TTL = uint32(ttlValue)
		}
	} else if currentTTL != 0 {
		record.TTL = currentTTL
	}
	if class != "" {
		record.Class = class
	} else if currentClass != "" {
		record.Class = currentClass
	} else {
		record.Class = "IN" // Default class
	}
	record.Type = recordType
	record.RData = rdata

	return record, nil
}

// splitFields splits a line into fields, handling quoted strings
func splitFields(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false
	escapeNext := false

	for _, char := range line {
		if escapeNext {
			current.WriteRune(char)
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			continue
		}

		if char == '"' {
			inQuotes = !inQuotes
			continue
		}

		if char == ' ' || char == '\t' {
			if !inQuotes {
				if current.Len() > 0 {
					fields = append(fields, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(char)
			}
		} else {
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}

// parseTTL parses TTL values including time units
func parseTTL(ttlStr string) (uint32, error) {
	ttlStr = strings.ToUpper(ttlStr)

	// Handle time units
	multiplier := uint32(1)
	if strings.HasSuffix(ttlStr, "S") {
		multiplier = 1
		ttlStr = strings.TrimSuffix(ttlStr, "S")
	} else if strings.HasSuffix(ttlStr, "M") {
		multiplier = 60
		ttlStr = strings.TrimSuffix(ttlStr, "M")
	} else if strings.HasSuffix(ttlStr, "H") {
		multiplier = 3600
		ttlStr = strings.TrimSuffix(ttlStr, "H")
	} else if strings.HasSuffix(ttlStr, "D") {
		multiplier = 86400
		ttlStr = strings.TrimSuffix(ttlStr, "D")
	} else if strings.HasSuffix(ttlStr, "W") {
		multiplier = 604800
		ttlStr = strings.TrimSuffix(ttlStr, "W")
	}

	value, err := strconv.ParseUint(ttlStr, 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32(value) * multiplier, nil
}

// isClass checks if a string is a valid DNS class
func isClass(s string) bool {
	classes := []string{"IN", "CH", "HS", "NONE", "ANY"}
	for _, class := range classes {
		if strings.ToUpper(s) == class {
			return true
		}
	}
	return false
}

// PrintZone prints the parsed zone in a readable format
func (z *Zone) PrintZone() {
	fmt.Printf("Zone: %s\n", z.Origin)
	if z.DefaultTTL != 0 {
		fmt.Printf("Default TTL: %d\n", z.DefaultTTL)
	}
	fmt.Printf("Records: %d\n\n", len(z.Records))

	for i, record := range z.Records {
		fmt.Printf("Record %d:\n", i+1)
		fmt.Printf("  Name: %s\n", record.Name)
		fmt.Printf("  TTL: %d\n", record.TTL)
		fmt.Printf("  Class: %s\n", record.Class)
		fmt.Printf("  Type: %s\n", record.Type)
		fmt.Printf("  RData: %s\n", record.RData)
		fmt.Println()
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <zone-file>\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: File '%s' does not exist\n", filename)
		os.Exit(1)
	}

	zone, err := ParseZone(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing zone file: %v\n", err)
		os.Exit(1)
	}

	zone.PrintZone()
}
