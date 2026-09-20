package http1

import (
	"io"
	"net"

	"github.com/23jdd/Nexo/protocol"
)

type Response struct {
	Protocol   string
	StatusCode int
	Header     protocol.Header
	Body       io.ReadCloser
}

// HTTP/1.1 200 OK\r\n
func WriteResponse(con net.Conn, response *Response) error {
	_, err := con.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"))
	return err
}
