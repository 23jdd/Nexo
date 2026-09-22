package protocol

import (
	"net/textproto"
	"strings"
)

type Header map[string][]string

func (h Header) Get(key string) string {
	values, ok := h.values(key)
	if !ok || len(values) == 0 {
		return ""
	} else {
		return values[0]
	}
}

func (h Header) Set(key, value string) {
	h.Del(key)
	h[textproto.CanonicalMIMEHeaderKey(key)] = []string{value}
}

func (h Header) Add(key, value string) {
	if existing, ok := h.key(key); ok {
		key = existing
	} else {
		key = textproto.CanonicalMIMEHeaderKey(key)
	}
	h[key] = append(h[key], value)
}

func (h Header) Del(key string) {
	for existing := range h {
		if strings.EqualFold(existing, key) {
			delete(h, existing)
		}
	}
}

func (h Header) Values(key string) []string {
	values, _ := h.values(key)
	return append([]string(nil), values...)
}

func (h Header) values(key string) ([]string, bool) {
	if values, ok := h[textproto.CanonicalMIMEHeaderKey(key)]; ok {
		return values, true
	}
	if existing, ok := h.key(key); ok {
		return h[existing], true
	}
	return nil, false
}

func (h Header) key(key string) (string, bool) {
	for existing := range h {
		if strings.EqualFold(existing, key) {
			return existing, true
		}
	}
	return "", false
}
