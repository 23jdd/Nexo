package server

import (
	"github.com/23jdd/Nexo/protocol/http1"
)

type Handler func(w http1.ResponseWriter, r *http1.Request)

func (hs*HttpServer)HandleFunc (pattern string, handler Handler){

}
func (hs*HttpServer)ReadRequest(){

}