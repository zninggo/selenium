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

func TestW3CFindStrategy(t *testing.T) {
	tests := []struct {
		by, value, wantBy, wantValue string
	}{
		{ByID, "login", ByCSSSelector, "#login"},
		{ByName, "q", ByCSSSelector, `[name="q"]`},
		{ByClassName, "btn-primary", ByCSSSelector, `[class~="btn-primary"]`},
		{ByCSSSelector, ".keep", ByCSSSelector, ".keep"},
		{ByXPATH, "//div", ByXPATH, "//div"},
		{ByTagName, "input", ByTagName, "input"},
	}
	for _, tc := range tests {
		gotBy, gotValue := w3cFindStrategy(tc.by, tc.value)
		if gotBy != tc.wantBy || gotValue != tc.wantValue {
			t.Errorf("w3cFindStrategy(%q, %q) = (%q, %q), want (%q, %q)",
				tc.by, tc.value, gotBy, gotValue, tc.wantBy, tc.wantValue)
		}
	}
}
