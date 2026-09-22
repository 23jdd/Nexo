package http1

import (
	"bytes"
	"io"
	"net"

	"github.com/23jdd/Nexo/protocol"
)

type ResponseWriter struct {
	w        io.Writer
	response *Response
	body     bytes.Buffer
	flushed  bool
}

func NewResponseWriter(con net.Conn, response *Response) *ResponseWriter {
	return &ResponseWriter{
		w:        con,
		response: response,
	}
}

func NewResponseWriterFor(w io.Writer, response *Response) *ResponseWriter {
	return &ResponseWriter{w: w, response: response}
}

// Header returns the response headers. Changes after Flush have no effect.
func (rw *ResponseWriter) Header() protocol.Header {
	return rw.response.Header
}

// StatusCode sets the response status. Changes after Flush have no effect.
func (rw *ResponseWriter) StatusCode(status int) {
	rw.response.StatusCode = status
}

func (rw *ResponseWriter) Write(data []byte) error {
	if rw.flushed {
		return io.ErrClosedPipe
	}
	_, err := rw.body.Write(data)
	return err
}

func (rw *ResponseWriter) Flush() error {
	if rw.flushed {
		return nil
	}
	rw.flushed = true
	rw.response.Body = io.NopCloser(bytes.NewReader(rw.body.Bytes()))
	return WriteResponse(rw.w, rw.response)
}
