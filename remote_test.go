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
	s := string(b)
	for _, field := range []string{"expiry", "path", "domain", "secure", "httpOnly"} {
		if strings.Contains(s, field) {
			t.Fatalf("zero optional field %q should be omitted, got %s", field, s)
		}
	}
	c.Expiry = 1
	c.Path = "/"
	c.Secure = true
	b, err = json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	s = string(b)
	if !strings.Contains(s, `"expiry":1`) || !strings.Contains(s, `"path":"/"`) || !strings.Contains(s, `"secure":true`) {
		t.Fatalf("non-zero optional fields should be present, got %s", s)
	}
}

func TestProcessKeyStringW3CIncludesValueList(t *testing.T) {
	wd := &remoteWD{w3cCompatible: true}
	got := wd.processKeyString("ab")
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", got)
	}
	if m["text"] != "ab" {
		t.Fatalf("text = %v, want ab", m["text"])
	}
	val, ok := m["value"].([]string)
	if !ok {
		t.Fatalf("value type %T, want []string", m["value"])
	}
	if len(val) != 2 || val[0] != "a" || val[1] != "b" {
		t.Fatalf("value = %#v, want [a b]", val)
	}
}

func TestProcessKeyStringLegacy(t *testing.T) {
	wd := &remoteWD{w3cCompatible: false}
	got := wd.processKeyString("ab")
	m, ok := got.(map[string][]string)
	if !ok {
		t.Fatalf("expected map[string][]string, got %T", got)
	}
	if len(m["value"]) != 2 || m["value"][0] != "a" || m["value"][1] != "b" {
		t.Fatalf("value = %#v", m["value"])
	}
}

func TestCookieSameSiteTagRoundTrip(t *testing.T) {
	// Ensure the internal wire cookie struct tag is valid JSON.
	raw := []byte(`{"name":"n","value":"v","sameSite":"Lax"}`)
	var c cookie
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if c.SameSite != "Lax" {
		t.Fatalf("SameSite = %q, want Lax", c.SameSite)
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
