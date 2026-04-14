package zonefile

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestParseExample(t *testing.T) {
	result, err := ParseFile("testdata/example.zone", nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if result.ZoneOrigin != "example.com." {
		t.Errorf("ZoneOrigin = %q, want %q", result.ZoneOrigin, "example.com.")
	}
	// example.zone has: 1 SOA, 2 NS, 4 A, 2 CNAME, 2 MX, 2 TXT, 2 AAAA, 2 SRV = 17 records
	if len(result.Records) != 17 {
		t.Errorf("got %d records, want 17", len(result.Records))
	}

	// Verify SOA record
	soa := result.Records[0]
	if soa.Type != "SOA" {
		t.Errorf("first record type = %q, want SOA", soa.Type)
	}
	if soa.Name != "example.com." {
		t.Errorf("SOA name = %q, want example.com.", soa.Name)
	}
	if soa.TTL == nil || *soa.TTL != 86400 {
		t.Errorf("SOA TTL = %v, want 86400", soa.TTL)
	}
}

func TestParseExample2(t *testing.T) {
	var warnings bytes.Buffer
	result, err := ParseFile("testdata/example2.zone", nil, &warnings)
	if err != nil {
		t.Fatal(err)
	}
	// Zone origin gets overwritten by the second $ORIGIN directive
	// (matches Python behavior)
	if result.ZoneOrigin != "internal.example2.com." {
		t.Errorf("ZoneOrigin = %q, want %q", result.ZoneOrigin, "internal.example2.com.")
	}

	// Verify $INCLUDE warning was produced
	if !strings.Contains(warnings.String(), "$INCLUDE") {
		t.Error("expected $INCLUDE warning")
	}

	// Find specific records to verify parsing
	var foundSOA, foundCH, foundWild, found3com, foundInternal bool
	for _, r := range result.Records {
		switch {
		case r.Type == "SOA":
			foundSOA = true
			// SOA has explicit TTL of 2w = 1209600
			if r.TTL == nil || *r.TTL != 1209600 {
				t.Errorf("SOA TTL = %v, want 1209600", r.TTL)
			}
		case r.Type == "TLSA" && r.Class == "CH":
			foundCH = true
		case strings.HasPrefix(r.Name, "*.wild"):
			foundWild = true
		case r.Name == "3com.example2.com.":
			found3com = true
		case r.Name == "gateway.internal.example2.com.":
			foundInternal = true
		}
	}

	if !foundSOA {
		t.Error("SOA record not found")
	}
	if !foundCH {
		t.Error("CH class TLSA record not found")
	}
	if !foundWild {
		t.Error("wildcard record not found")
	}
	if !found3com {
		t.Error("3com record not found")
	}
	if !foundInternal {
		t.Error("internal.example2.com. record not found after $ORIGIN change")
	}
}

func TestParseFromReader(t *testing.T) {
	input := `$ORIGIN test.com.
$TTL 300
@ IN SOA ns1 admin 1 3600 900 604800 86400
@ IN A 1.2.3.4
www IN A 5.6.7.8
`
	result, err := Parse(strings.NewReader(input), nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if result.ZoneOrigin != "test.com." {
		t.Errorf("ZoneOrigin = %q, want test.com.", result.ZoneOrigin)
	}
	if len(result.Records) != 3 {
		t.Errorf("got %d records, want 3", len(result.Records))
	}

	// Check www record expanded
	www := result.Records[2]
	if www.Name != "www.test.com." {
		t.Errorf("www name = %q, want www.test.com.", www.Name)
	}
	if www.TTL == nil || *www.TTL != 300 {
		t.Errorf("www TTL = %v, want 300", www.TTL)
	}
}

func TestParseMultiLine(t *testing.T) {
	input := `$ORIGIN test.com.
$TTL 300
@ IN SOA ns1.test.com. admin.test.com. (
    2024010101  ; Serial
    3600        ; Refresh
    1800        ; Retry
    1209600     ; Expire
    86400       ; Minimum
)
@ IN A 1.2.3.4
`
	result, err := Parse(strings.NewReader(input), nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 2 {
		t.Errorf("got %d records, want 2", len(result.Records))
	}
	soa := result.Records[0]
	if soa.Type != "SOA" {
		t.Errorf("first record type = %q, want SOA", soa.Type)
	}
	// SOA data should contain all fields on one logical line
	if !strings.Contains(soa.Data, "2024010101") {
		t.Error("SOA data missing serial number")
	}
}

func TestParseContinuationLine(t *testing.T) {
	input := `$ORIGIN test.com.
$TTL 300
host1 IN A 1.2.3.4
      IN AAAA 2001:db8::1
`
	result, err := Parse(strings.NewReader(input), nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 2 {
		t.Errorf("got %d records, want 2", len(result.Records))
	}
	if result.Records[1].Name != "host1.test.com." {
		t.Errorf("continuation name = %q, want host1.test.com.", result.Records[1].Name)
	}
	if result.Records[1].Type != "AAAA" {
		t.Errorf("continuation type = %q, want AAAA", result.Records[1].Type)
	}
}

func TestParseWithFilters(t *testing.T) {
	result, err := ParseFile("testdata/example.zone", &FilterConfig{RRTypes: "A"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range result.Records {
		if r.Type != "A" {
			t.Errorf("expected only A records, got %s", r.Type)
		}
	}
	if len(result.Records) != 4 {
		t.Errorf("got %d A records, want 4", len(result.Records))
	}
}

func TestParseTTLSuffixes(t *testing.T) {
	input := `$ORIGIN test.com.
$TTL 1h
host1 30m IN A 1.2.3.4
host2 IN A 5.6.7.8
host3 1d IN A 9.10.11.12
`
	result, err := Parse(strings.NewReader(input), nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 3 {
		t.Fatalf("got %d records, want 3", len(result.Records))
	}
	// host1: explicit 30m = 1800
	if *result.Records[0].TTL != 1800 {
		t.Errorf("host1 TTL = %d, want 1800", *result.Records[0].TTL)
	}
	// host2: inherits $TTL 1h = 3600
	if *result.Records[1].TTL != 3600 {
		t.Errorf("host2 TTL = %d, want 3600", *result.Records[1].TTL)
	}
	// host3: explicit 1d = 86400
	if *result.Records[2].TTL != 86400 {
		t.Errorf("host3 TTL = %d, want 86400", *result.Records[2].TTL)
	}
}

func TestParseQuotedSemicolon(t *testing.T) {
	input := `$ORIGIN test.com.
$TTL 300
_dmarc IN TXT "v=DMARC1; p=reject; rua=mailto:d@test.com"
`
	result, err := Parse(strings.NewReader(input), nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("got %d records, want 1", len(result.Records))
	}
	// The semicolons inside quotes should be preserved
	if !strings.Contains(result.Records[0].Data, "p=reject") {
		t.Error("semicolons inside quotes were incorrectly treated as comments")
	}
}
