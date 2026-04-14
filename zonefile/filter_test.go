package zonefile

import (
	"io"
	"testing"
)

func intPtr(n int) *int { return &n }

func TestIncludeRecord_NoDNSSEC(t *testing.T) {
	filters := &FilterConfig{NoDNSSEC: true}
	rec := &Record{Name: "example.com.", Type: "RRSIG", Class: "IN"}
	if IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("RRSIG should be excluded with NoDNSSEC")
	}
	rec.Type = "A"
	if !IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("A record should be included with NoDNSSEC")
	}
}

func TestIncludeRecord_RRTypes(t *testing.T) {
	filters := &FilterConfig{RRTypes: "A,AAAA"}
	rec := &Record{Name: "example.com.", Type: "A", Class: "IN"}
	if !IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("A should be included")
	}
	rec.Type = "MX"
	if IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("MX should be excluded")
	}
}

func TestIncludeRecord_IncludeName(t *testing.T) {
	filters := &FilterConfig{IncludeName: "www"}
	rec := &Record{Name: "www.example.com.", Type: "A", Class: "IN"}
	if !IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("www.example.com. should match includename 'www'")
	}
	rec.Name = "mail.example.com."
	if IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("mail.example.com. should not match includename 'www'")
	}
}

func TestIncludeRecord_ExcludeName(t *testing.T) {
	filters := &FilterConfig{ExcludeName: "mail"}
	rec := &Record{Name: "mail.example.com.", Type: "A", Class: "IN"}
	if IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("mail.example.com. should be excluded")
	}
	rec.Name = "www.example.com."
	if !IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("www.example.com. should be included")
	}
}

func TestIncludeRecord_IncludeData(t *testing.T) {
	filters := &FilterConfig{IncludeData: "192.168"}
	rec := &Record{Name: "test.", Type: "A", Class: "IN", Data: "192.168.1.1"}
	if !IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("192.168.1.1 should match includedata '192.168'")
	}
	rec.Data = "10.0.0.1"
	if IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("10.0.0.1 should not match includedata '192.168'")
	}
}

func TestIncludeRecord_Wildcard(t *testing.T) {
	filters := &FilterConfig{Wildcard: true}
	rec := &Record{Name: "*.example.com.", Type: "A", Class: "IN"}
	if !IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("*.example.com. should match wildcard filter")
	}
	rec.Name = "www.example.com."
	if IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("www.example.com. should not match wildcard filter")
	}
}

func TestIncludeRecord_Delegations(t *testing.T) {
	filters := &FilterConfig{Delegations: true}
	rec := &Record{Name: "child.example.com.", Type: "NS", Class: "IN"}
	if !IncludeRecord(rec, filters, "example.com.", io.Discard) {
		t.Error("child NS should be included as delegation")
	}
	rec.Name = "example.com."
	if IncludeRecord(rec, filters, "example.com.", io.Discard) {
		t.Error("apex NS should be excluded from delegations")
	}
	rec.Name = "child.example.com."
	rec.Type = "A"
	if IncludeRecord(rec, filters, "example.com.", io.Discard) {
		t.Error("A record should be excluded by delegations filter")
	}
}

func TestIncludeRecord_TTLRange(t *testing.T) {
	filters := &FilterConfig{TTLMin: intPtr(300), TTLMax: intPtr(3600)}
	rec := &Record{Name: "test.", Type: "A", Class: "IN", TTL: intPtr(600)}
	if !IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("TTL 600 should be in range [300, 3600]")
	}
	rec.TTL = intPtr(100)
	if IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("TTL 100 should be below min 300")
	}
	rec.TTL = intPtr(7200)
	if IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("TTL 7200 should be above max 3600")
	}
	rec.TTL = nil
	if IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("nil TTL should be excluded when TTL range is set")
	}
}

func TestIncludeRecord_ClassFilter(t *testing.T) {
	filters := &FilterConfig{ClassFilter: "CH"}
	rec := &Record{Name: "test.", Type: "A", Class: "IN"}
	if IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("IN should not match class filter CH")
	}
	rec.Class = "CH"
	if !IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("CH should match class filter CH")
	}
}

func TestIncludeRecord_LabelCount(t *testing.T) {
	filters := &FilterConfig{MinLabelCount: intPtr(3), MaxLabelCount: intPtr(4)}
	// "example.com." has 3 labels (example, com, root)
	rec := &Record{Name: "example.com.", Type: "A", Class: "IN"}
	if !IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("example.com. (3 labels) should pass [3,4]")
	}
	// "www.example.com." has 4 labels
	rec.Name = "www.example.com."
	if !IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("www.example.com. (4 labels) should pass [3,4]")
	}
	// "a.b.example.com." has 5 labels
	rec.Name = "a.b.example.com."
	if IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("a.b.example.com. (5 labels) should fail max 4")
	}
}

func TestIncludeRecord_Regex(t *testing.T) {
	filters := &FilterConfig{IncludeName: "^www\\.", Regex: true}
	rec := &Record{Name: "www.example.com.", Type: "A", Class: "IN"}
	if !IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("www.example.com. should match regex ^www\\.")
	}
	rec.Name = "mail.example.com."
	if IncludeRecord(rec, filters, "", io.Discard) {
		t.Error("mail.example.com. should not match regex ^www\\.")
	}
}

func TestLabelCount(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{".", 1},
		{"com.", 2},
		{"example.com.", 3},
		{"www.example.com.", 4},
		{"a.b.c.example.com.", 6},
	}
	for _, tt := range tests {
		got := labelCount(tt.name)
		if got != tt.want {
			t.Errorf("labelCount(%q) = %d, want %d", tt.name, got, tt.want)
		}
	}
}
