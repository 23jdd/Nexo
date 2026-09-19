package server

import (
	"log"
	"net"
	"strconv"
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

}
