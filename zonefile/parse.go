package zonefile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Parse reads DNS records from r and returns the parse result.
// The filters parameter may be nil for no filtering.
// Warnings are written to warnWriter; pass nil to discard.
func Parse(r io.Reader, filters *FilterConfig, warnWriter io.Writer) (*ParseResult, error) {
	if filters == nil {
		filters = &FilterConfig{}
	}
	if warnWriter == nil {
		warnWriter = io.Discard
	}

	result := &ParseResult{}

	var (
		currentOrigin              string
		currentTTL                 *int
		zoneOrigin                 string
		firstSOAFound              bool
		previousName               string
		inParens                   bool
		accumulatedLine            string
		accumulatedLineNum         int
		accumulatedStartsWithSpace bool
	)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	lineNum := 0

	for scanner.Scan() {
		rawLine := scanner.Text()
		lineNum++

		// Strip comments outside of quoted strings
		commentPos := findComment(rawLine)
		if commentPos >= 0 {
			rawLine = rawLine[:commentPos]
		}

		stripped := strings.TrimSpace(rawLine)
		if stripped == "" {
			if !inParens {
				result.SkippedLines++
			}
			continue
		}

		// Handle multi-line records (parenthesized groups)
		var line string
		var startsWithSpace bool

		if inParens {
			accumulatedLine += " " + strings.ReplaceAll(strings.ReplaceAll(stripped, "(", ""), ")", "")
			if strings.Contains(stripped, ")") {
				inParens = false
				line = strings.TrimSpace(accumulatedLine)
				lineNum = accumulatedLineNum
				startsWithSpace = accumulatedStartsWithSpace
			} else {
				continue
			}
		} else if strings.Contains(stripped, "(") {
			accumulatedStartsWithSpace = len(rawLine) > 0 && (rawLine[0] == ' ' || rawLine[0] == '\t')
			accumulatedLine = strings.ReplaceAll(strings.ReplaceAll(stripped, "(", ""), ")", "")
			accumulatedLineNum = lineNum
			if strings.Contains(stripped, ")") {
				// Opening and closing parens on same line
				line = strings.TrimSpace(accumulatedLine)
				startsWithSpace = accumulatedStartsWithSpace
			} else {
				inParens = true
				continue
			}
		} else {
			line = stripped
			startsWithSpace = len(rawLine) > 0 && (rawLine[0] == ' ' || rawLine[0] == '\t')
		}

		// Process directives
		if strings.HasPrefix(line, "$ORIGIN") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentOrigin = parts[1]
				zoneOrigin = currentOrigin
			}
			continue
		}

		if strings.HasPrefix(line, "$TTL") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if ttl, ok := ParseTTL(parts[1]); ok {
					currentTTL = &ttl
				} else {
					fmt.Fprintf(warnWriter, "Warning: Invalid $TTL value on line %d: %s\n", lineNum, parts[1])
				}
			}
			continue
		}

		if strings.HasPrefix(line, "$INCLUDE") {
			fmt.Fprintf(warnWriter, "Warning: $INCLUDE directive on line %d not supported, skipping.\n", lineNum)
			result.SkippedLines++
			continue
		}

		if strings.HasPrefix(line, "$") {
			result.SkippedLines++
			continue
		}

		// Parse record
		parts := strings.Fields(line)
		if len(parts) < 2 {
			fmt.Fprintf(warnWriter, "Warning: Skipping malformed line %d: %s\n", lineNum, line)
			result.SkippedLines++
			continue
		}

		idx := 0

		// Determine owner name
		var name string
		if startsWithSpace {
			name = previousName
			if name == "" {
				fmt.Fprintf(warnWriter, "Warning: No previous owner name for line %d: %s\n", lineNum, line)
				result.SkippedLines++
				continue
			}
		} else {
			name = parts[0]
			idx = 1
		}

		// Parse optional TTL and class in either order (RFC 1035)
		var ttl *int
		var recordClass string
		for i := 0; i < 2; i++ {
			if idx >= len(parts) {
				break
			}
			token := parts[idx]
			if parsed, ok := ParseTTL(token); ok && ttl == nil {
				ttl = &parsed
				idx++
			} else if dnsClasses[strings.ToUpper(token)] && recordClass == "" {
				recordClass = strings.ToUpper(token)
				idx++
			} else {
				break
			}
		}

		if ttl == nil {
			ttl = currentTTL
		}
		if recordClass == "" {
			recordClass = "IN"
		}

		// Next field must be the record type
		if idx >= len(parts) {
			fmt.Fprintf(warnWriter, "Warning: Skipping incomplete line %d: %s\n", lineNum, line)
			result.SkippedLines++
			continue
		}

		recordType := strings.ToUpper(parts[idx])
		idx++

		// Remaining fields are the record data
		data := strings.Join(parts[idx:], " ")

		// Expand relative names
		if name == "@" && currentOrigin != "" {
			name = currentOrigin
		} else if !strings.HasSuffix(name, ".") && currentOrigin != "" {
			name = name + "." + currentOrigin
		}

		previousName = name

		// Determine zone origin from first SOA record if $ORIGIN not present
		if !firstSOAFound && recordType == "SOA" {
			if zoneOrigin == "" {
				zoneOrigin = name
			}
			firstSOAFound = true
		}

		record := Record{
			Name:  name,
			TTL:   ttl,
			Class: recordClass,
			Type:  recordType,
			Data:  data,
			Line:  lineNum,
		}

		if IncludeRecord(&record, filters, zoneOrigin, warnWriter) {
			result.Records = append(result.Records, record)
		} else {
			result.SkippedLines++
		}
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("reading input: %w", err)
	}

	result.ZoneOrigin = zoneOrigin
	return result, nil
}

// ParseFile opens the named file and parses it as a DNS zone file.
func ParseFile(filepath string, filters *FilterConfig, warnWriter io.Writer) (*ParseResult, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f, filters, warnWriter)
}
