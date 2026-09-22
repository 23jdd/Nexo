package protocol

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type SameSite int

const (
	SameSiteDefaultMode SameSite = iota
	SameSiteLaxMode
	SameSiteStrictMode
	SameSiteNoneMode
)

type Cookie struct {
	Name     string
	Value    string
	Path     string
	Domain   string
	Expires  time.Time
	MaxAge   int
	Secure   bool
	HttpOnly bool
	SameSite SameSite
}

func (c Cookie) String() string {
	if !validCookieName(c.Name) || !validCookieValue(c.Value) || !validCookieAttribute(c.Path) || !validCookieAttribute(c.Domain) {
		return ""
	}
	var b strings.Builder
	b.WriteString(c.Name)
	b.WriteByte('=')
	b.WriteString(c.Value)
	if c.Path != "" {
		b.WriteString("; Path=")
		b.WriteString(c.Path)
	}
	if c.Domain != "" {
		b.WriteString("; Domain=")
		b.WriteString(c.Domain)
	}
	if !c.Expires.IsZero() {
		b.WriteString("; Expires=")
		b.WriteString(c.Expires.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	}
	if c.MaxAge != 0 {
		b.WriteString("; Max-Age=")
		b.WriteString(strconv.Itoa(c.MaxAge))
	}
	if c.HttpOnly {
		b.WriteString("; HttpOnly")
	}
	if c.Secure {
		b.WriteString("; Secure")
	}
	switch c.SameSite {
	case SameSiteLaxMode:
		b.WriteString("; SameSite=Lax")
	case SameSiteStrictMode:
		b.WriteString("; SameSite=Strict")
	case SameSiteNoneMode:
		b.WriteString("; SameSite=None")
	}
	return b.String()
}

func ParseCookieHeader(line string) ([]Cookie, error) {
	var cookies []Cookie
	for _, part := range strings.Split(line, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, value, ok := strings.Cut(part, "=")
		name, value = strings.TrimSpace(name), strings.TrimSpace(value)
		if !ok || !validCookieName(name) || !validCookieValue(value) {
			return nil, fmt.Errorf("invalid Cookie pair %q", part)
		}
		cookies = append(cookies, Cookie{Name: name, Value: value})
	}
	return cookies, nil
}

func ParseSetCookie(line string) (*Cookie, error) {
	parts := strings.Split(line, ";")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty Set-Cookie")
	}
	name, value, ok := strings.Cut(strings.TrimSpace(parts[0]), "=")
	if !ok || !validCookieName(name) || !validCookieValue(value) {
		return nil, fmt.Errorf("invalid Set-Cookie %q", line)
	}
	cookie := &Cookie{Name: name, Value: value}
	for _, raw := range parts[1:] {
		attribute, attributeValue, hasValue := strings.Cut(strings.TrimSpace(raw), "=")
		switch strings.ToLower(attribute) {
		case "path":
			if hasValue {
				cookie.Path = attributeValue
			}
		case "domain":
			if hasValue {
				cookie.Domain = attributeValue
			}
		case "expires":
			if hasValue {
				expires, err := parseCookieTime(attributeValue)
				if err != nil {
					return nil, err
				}
				cookie.Expires = expires
			}
		case "max-age":
			if hasValue {
				maxAge, err := strconv.Atoi(attributeValue)
				if err != nil {
					return nil, fmt.Errorf("invalid Max-Age %q", attributeValue)
				}
				cookie.MaxAge = maxAge
			}
		case "secure":
			cookie.Secure = true
		case "httponly":
			cookie.HttpOnly = true
		case "samesite":
			switch strings.ToLower(attributeValue) {
			case "lax":
				cookie.SameSite = SameSiteLaxMode
			case "strict":
				cookie.SameSite = SameSiteStrictMode
			case "none":
				cookie.SameSite = SameSiteNoneMode
			}
		}
	}
	return cookie, nil
}

func parseCookieTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC1123, time.RFC1123Z, time.RFC850, time.ANSIC} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid cookie Expires %q", value)
}

func validCookieName(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char <= 0x20 || char >= 0x7f || strings.ContainsRune("()<>@,;:\\\"/[]?={}", char) {
			return false
		}
	}
	return true
}

func validCookieValue(value string) bool {
	for _, char := range value {
		if char < 0x21 || char > 0x7e || char == ';' || char == ',' || char == '"' || char == '\\' {
			return false
		}
	}
	return true
}

func validCookieAttribute(value string) bool {
	for _, char := range value {
		if char < 0x20 || char >= 0x7f || char == ';' {
			return false
		}
	}
	return true
}
