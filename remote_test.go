package selenium

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCookieExpiryOmitempty(t *testing.T) {
	c := Cookie{Name: "a", Value: "b"}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "expiry") {
		t.Fatalf("zero Expiry should be omitted, got %s", b)
	}
	c.Expiry = 1
	b, err = json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"expiry":1`) {
		t.Fatalf("non-zero Expiry should be present, got %s", b)
	}
}
