package zonefile

import "testing"

func TestParseTTL(t *testing.T) {
	tests := []struct {
		input string
		want  int
		ok    bool
	}{
		{"3600", 3600, true},
		{"86400", 86400, true},
		{"1h", 3600, true},
		{"1h30m", 5400, true},
		{"1d", 86400, true},
		{"2w", 1209600, true},
		{"30m", 1800, true},
		{"1H", 3600, true},   // case insensitive
		{"1D", 86400, true},  // case insensitive
		{"300", 300, true},
		{"1s", 1, true},
		{"1h30", 3630, true}, // trailing bare number = seconds
		{"0", 0, false},      // zero not valid
		{"", 0, false},
		{"abc", 0, false},
		{"1x", 0, false},
		{"hello", 0, false},
		{"-1", 0, false},
	}

	for _, tt := range tests {
		got, ok := ParseTTL(tt.input)
		if ok != tt.ok || got != tt.want {
			t.Errorf("ParseTTL(%q) = (%d, %v), want (%d, %v)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}
