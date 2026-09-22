package protocol

import (
	"strings"
	"testing"
	"time"
)

func TestCookieRoundTrip(t *testing.T) {
	expires := time.Date(2030, time.January, 2, 3, 4, 5, 0, time.UTC)
	original := Cookie{
		Name: "session", Value: "abc123", Path: "/", Domain: "example.com",
		Expires: expires, MaxAge: 3600, Secure: true, HttpOnly: true, SameSite: SameSiteStrictMode,
	}
	wire := original.String()
	parsed, err := ParseSetCookie(wire)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Name != original.Name || parsed.Value != original.Value || parsed.Path != original.Path ||
		parsed.Domain != original.Domain || parsed.MaxAge != original.MaxAge || !parsed.Secure || !parsed.HttpOnly ||
		parsed.SameSite != SameSiteStrictMode || !parsed.Expires.Equal(expires) {
		t.Fatalf("parsed cookie = %#v; wire = %q", parsed, wire)
	}
}

func TestParseCookieHeader(t *testing.T) {
	cookies, err := ParseCookieHeader("session=abc; theme=dark")
	if err != nil {
		t.Fatal(err)
	}
	if len(cookies) != 2 || cookies[1].Name != "theme" || cookies[1].Value != "dark" {
		t.Fatalf("cookies = %#v", cookies)
	}
}

func TestCookieRejectsHeaderInjection(t *testing.T) {
	if got := (Cookie{Name: "session", Value: "ok", Path: "/\r\nInjected: yes"}).String(); got != "" {
		t.Fatalf("Cookie.String() = %q", got)
	}
	if _, err := ParseCookieHeader("bad=value\r\nInjected=yes"); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("ParseCookieHeader() error = %v", err)
	}
}
