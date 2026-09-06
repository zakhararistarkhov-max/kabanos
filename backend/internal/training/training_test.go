package training

import "testing"

func TestValidators(t *testing.T) {
	if !validCategory("strength") || validCategory("nope") {
		t.Fatal("category validation wrong")
	}
	if !validDifficulty("hard") || validDifficulty("extreme") {
		t.Fatal("difficulty validation wrong")
	}
	if !validJoint("low") || validJoint("zero") {
		t.Fatal("joint validation wrong")
	}
}

func TestNormalizeTags(t *testing.T) {
	got := normalizeTags([]string{" Dumbbells ", "dumbbells", "", "BARBELL", "барбелл"}, 12)
	// lower-cased, trimmed, de-duplicated → 3 unique entries.
	if len(got) != 3 {
		t.Fatalf("expected 3 tags, got %d: %v", len(got), got)
	}
	if got[0] != "dumbbells" {
		t.Fatalf("expected first tag 'dumbbells', got %q", got[0])
	}
}

func TestNormalizeTagsCap(t *testing.T) {
	in := make([]string, 20)
	for i := range in {
		in[i] = string(rune('a' + i))
	}
	if got := normalizeTags(in, 5); len(got) != 5 {
		t.Fatalf("expected cap of 5, got %d", len(got))
	}
}

func TestRound1(t *testing.T) {
	if got := round1(4.05); got != 4.1 && got != 4.0 { // float rounding tolerance
		t.Fatalf("round1(4.05)=%v", got)
	}
	if got := round1(3.333333); got != 3.3 {
		t.Fatalf("round1(3.3333)=%v", got)
	}
}

func TestValidVideoURL(t *testing.T) {
	cases := map[string]bool{
		"":                          true,
		"https://youtu.be/abc":      true,
		"http://example.com/v":      true,
		"ftp://example.com/v":       false,
		"javascript:alert(1)":       false,
		"not a url":                 false,
	}
	for in, want := range cases {
		if got := validVideoURL(in); got != want {
			t.Fatalf("validVideoURL(%q)=%v want %v", in, got, want)
		}
	}
}
