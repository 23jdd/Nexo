package server

import (
	"github.com/23jdd/Nexo/protocol"
	"github.com/23jdd/Nexo/protocol/http1"
)

type Handler func(w *http1.ResponseWriter, r *http1.Request)

func (hs *HttpServer) HandleFunc(pattern string, handler Handler) {
	hs.handle("", pattern, handler)
}

func (hs *HttpServer) handle(method, pattern string, handler Handler) {
	if hs.m[pattern] == nil {
		hs.m[pattern] = make(map[string]Handler)
	}
	hs.m[pattern][method] = handler
}

func (hs *HttpServer) GET(pattern string, handler Handler) {
	hs.handle(string(protocol.Get), pattern, handler)
}

func (hs *HttpServer) POST(pattern string, handler Handler) {
	hs.handle(string(protocol.Post), pattern, handler)
}
