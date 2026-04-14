package zonefile

// findComment returns the index of the first semicolon (;) that is not
// inside a double-quoted string, or -1 if no such semicolon exists.
func findComment(line string) int {
	inQuote := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			inQuote = !inQuote
		case ';':
			if !inQuote {
				return i
			}
		}
	}
	return -1
}
