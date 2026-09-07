package config

import "testing"

func TestWebPushSubscriber(t *testing.T) {
	// webpush-go prepends "mailto:" to any non-https subject, so the value we
	// hand it must be a bare address (or an https URL) — never already
	// "mailto:"-prefixed, or Apple rejects the JWT as BadJwtToken.
	cases := []struct {
		in   string
		want string
	}{
		{"mailto:you@example.com", "you@example.com"},
		{"MAILTO:you@example.com", "you@example.com"},
		{"  mailto:you@example.com  ", "you@example.com"},
		{"you@example.com", "you@example.com"},
		{"https://example.com/contact", "https://example.com/contact"},
	}
	for _, c := range cases {
		got := VAPID{Subject: c.in}.WebPushSubscriber()
		if got != c.want {
			t.Errorf("WebPushSubscriber(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
