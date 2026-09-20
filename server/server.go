package server

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/23jdd/Nexo/protocol/http1"
)

type HttpServer struct {
	lis net.Listener
}

func New() *HttpServer {
	return &HttpServer{}
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
		go handler(con)
	}
}

// TODO
func handler(con net.Conn) {
	defer con.Close()
	for {
		request, err := http1.ReadRequest(con)
		if err != nil {
			log.Println(err)
			break
		}
		fmt.Println(request.String())
		err = http1.WriteResponse(con)
	}
}
