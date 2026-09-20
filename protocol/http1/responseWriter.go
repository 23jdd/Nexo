package http1

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"github.com/23jdd/Nexo/protocol"
)

type ResponseWriter struct {
	bw         *bufio.Writer
	response   *Response
	pushHeader bool
}

func NewResponseWriter(con net.Conn, response *Response) *ResponseWriter {
	bw := bufio.NewWriter(con)
	return &ResponseWriter{
		bw:       bw,
		response: response,
	}
}

// Header Can not set on Writer After
func (rw *ResponseWriter) Header() protocol.Header {
	return rw.response.Header
}

// StatusCode Can not set on Writer After
func (rw *ResponseWriter) StatusCode(status int) {
	rw.response.StatusCode = status
}

func (rw *ResponseWriter) Write(data []byte) error {
	if !rw.pushHeader {
		err := rw.writeHeader()
		if err != nil {
			return err
		}
	}
	_, err := rw.bw.Write(data)
	return err
}

// HTTP/1.1 200 OK\r\n
func (rw *ResponseWriter) writeHeader() error {
	var builder strings.Builder
	code := rw.response.StatusCode
	builder.WriteString(fmt.Sprintf("%s %d %s\r\n", rw.response.Protocol, code, protocol.StatusText(code)))
	for k, v := range rw.response.Header {
		builder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v[0]))
	}
	builder.Write([]byte{'\r', '\n'})
	content := builder.String()
	_, err := rw.bw.WriteString(content)
	rw.pushHeader = true
	return err
}
func (rw *ResponseWriter) Flush() error {
	return rw.bw.Flush()
}
