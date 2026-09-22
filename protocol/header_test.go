package protocol

import "testing"

func TestHeaderOperationsAreCaseInsensitive(t *testing.T) {
	h := make(Header)
	h.Add("content-type", "text/plain")
	h.Add("Content-Type", "application/json")
	if got := h.Get("CONTENT-TYPE"); got != "text/plain" {
		t.Fatalf("Get() = %q", got)
	}
	if got := len(h.Values("content-type")); got != 2 {
		t.Fatalf("Values() length = %d", got)
	}
	h.Set("CONTENT-TYPE", "text/html")
	if got := h.Values("Content-Type"); len(got) != 1 || got[0] != "text/html" {
		t.Fatalf("Values() = %#v", got)
	}
	h.Del("content-TYPE")
	if got := h.Get("Content-Type"); got != "" {
		t.Fatalf("Get() after Del = %q", got)
	}
}
