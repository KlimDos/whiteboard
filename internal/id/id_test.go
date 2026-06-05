package id

import (
	"regexp"
	"testing"
)

func TestGenerate(t *testing.T) {
	got, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 8 {
		t.Fatalf("want len 8, got %d: %q", len(got), got)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9]{8}$`).MatchString(got) {
		t.Fatalf("invalid id: %q", got)
	}
}

func TestValid(t *testing.T) {
	if !Valid("a3Kf9xQp") {
		t.Fatal("expected valid")
	}
	if Valid("bad/id!") {
		t.Fatal("expected invalid")
	}
}
