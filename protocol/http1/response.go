package http1

import (
	"io"
	"net"

	"github.com/23jdd/Nexo/protocol"
)

type Response struct {
	StatusCode int
	Header     protocol.Header
	Body       io.ReadCloser
}

// HTTP/1.1 200 OK\r\n
func WriteResponse(con net.Conn) error {
	_, err := con.Write([]byte("HTTP/1.1 200 OK\\r\\n"))
	return err
}
