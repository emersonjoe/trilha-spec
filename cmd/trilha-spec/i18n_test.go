package main

import (
	"regexp"
	"testing"
)

func TestDetectLang(t *testing.T) {
	for in, want := range map[string]string{"": "en", "en": "en", "pt": "pt", "pt-BR": "pt", "PT_br": "pt", "fr": "en", " pt ": "pt"} {
		if got := detectLang(in); got != want {
			t.Errorf("detectLang(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTranslateFallsBack(t *testing.T) {
	old := lang
	defer func() { lang = old }()
	lang = "pt"
	if got := T("error:"); got != "erro:" {
		t.Fatalf("pt: %q", got)
	}
	if got := T("a message without translation"); got != "a message without translation" {
		t.Fatalf("missing key must fall back to English, got %q", got)
	}
	lang = "xx"
	if got := T("error:"); got != "error:" {
		t.Fatalf("unknown language must be English, got %q", got)
	}
}

// Every translation keeps the format verbs of its key: a %s that goes
// missing in a translation is a runtime "%!(EXTRA" nobody sees in tests.
func TestCataloguesKeepVerbs(t *testing.T) {
	verbs := regexp.MustCompile(`%[-0-9]*[sdqv]`)
	for name, c := range catalogues {
		for key, val := range c {
			if a, b := verbs.FindAllString(key, -1), verbs.FindAllString(val, -1); len(a) != len(b) {
				t.Errorf("%s: %q has verbs %v, translation has %v", name, key, a, b)
			}
		}
	}
}
