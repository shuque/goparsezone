package zonefile

import "strings"

// ttlMultipliers maps BIND-style TTL suffixes to their value in seconds.
var ttlMultipliers = map[byte]int{
	's': 1,
	'm': 60,
	'h': 3600,
	'd': 86400,
	'w': 604800,
}

// ParseTTL parses a TTL string value, handling BIND-style unit suffixes.
// Examples: "3600", "1h", "1h30m", "1d", "2w".
// Returns the TTL in seconds and true if valid, or 0 and false if not.
func ParseTTL(value string) (int, bool) {
	if len(value) == 0 || !isDigit(value[0]) {
		return 0, false
	}

	lower := strings.ToLower(value)

	// Pure numeric case
	allDigits := true
	for i := 0; i < len(lower); i++ {
		if !isDigit(lower[i]) {
			allDigits = false
			break
		}
	}
	if allDigits {
		n := 0
		for i := 0; i < len(lower); i++ {
			n = n*10 + int(lower[i]-'0')
		}
		return n, true
	}

	// Suffixed case
	total := 0
	current := 0
	for i := 0; i < len(lower); i++ {
		ch := lower[i]
		if isDigit(ch) {
			current = current*10 + int(ch-'0')
		} else if mult, ok := ttlMultipliers[ch]; ok {
			total += current * mult
			current = 0
		} else {
			return 0, false
		}
	}
	// Trailing bare number treated as seconds
	total += current

	if total > 0 {
		return total, true
	}
	return 0, false
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
