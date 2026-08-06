package main

import "testing"

func TestDefaultURL(t *testing.T) {
	if got := resolveURL(""); got != "http://localhost:8080" {
		t.Fatalf("resolveURL(\"\") = %q", got)
	}
}

func TestExplicitURL(t *testing.T) {
	if got := resolveURL("https://finance.hermestech.uk"); got != "https://finance.hermestech.uk" {
		t.Fatalf("resolveURL(explicit) = %q", got)
	}
}
