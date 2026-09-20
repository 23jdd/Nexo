package server

import (
	"errors"
	"io"
	"log"
	"net"
	"strconv"

	"github.com/23jdd/Nexo/protocol"
	"github.com/23jdd/Nexo/protocol/http1"
)

type HttpServer struct {
	lis net.Listener
	m   map[string]Handler
}

func New() *HttpServer {
	return &HttpServer{
		m: make(map[string]Handler),
	}
}
func (hs *HttpServer) Run(address string, port int) error {
	host := net.JoinHostPort(address, strconv.Itoa(port))
	lis, err := net.Listen("tcp", host)
	if err != nil {
		return err
	}
	defer func() {
		err := lis.Close()
		if err != nil {
			log.Println(err)
		}
	}()
	for {
		con, err := lis.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go hs.handler(con)
	}
}

// TODO
func (hs *HttpServer) handler(con net.Conn) {
	defer con.Close()
	request, err := http1.ReadRequest(con)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			log.Println(err)
		}
		return
	}
	//fmt.Println(request.String())
	//err = http1.WriteResponse(con, &http1.Response{})
	url := request.URL.Path
	handler := hs.m[url]
	writer := http1.NewResponseWriter(con, &http1.Response{
		Protocol:   request.Protocol,
		Header:     make(protocol.Header),
		Body:       nil,
		StatusCode: protocol.StatusOK,
	})
	if handler == nil {
		return
	}
	handler(writer, request)
	writer.Flush()
}

// TODO
func match(path string, pattern string) bool {
	return path == pattern
}
