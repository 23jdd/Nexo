package http1

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"

	"github.com/23jdd/Nexo/protocol"
)

type ResponseWriter struct {
	bw            *bufio.Writer
	response      *Response
	body          bytes.Buffer
	headerWritten bool
	chunked       bool
	finished      bool
}

func NewResponseWriter(con net.Conn, response *Response) *ResponseWriter {
	return &ResponseWriter{
		bw:       bufio.NewWriter(con),
		response: response,
	}
}

func NewResponseWriterFor(w io.Writer, response *Response) *ResponseWriter {
	return &ResponseWriter{bw: bufio.NewWriter(w), response: response}
}

// Header returns the response headers. Changes after Flush have no effect.
func (rw *ResponseWriter) Header() protocol.Header {
	return rw.response.Header
}

// Trailer returns fields written after the final chunk. Transfer-Encoding must
// be set to chunked for trailers to be sent.
func (rw *ResponseWriter) Trailer() protocol.Header {
	if rw.response.Trailer == nil {
		rw.response.Trailer = make(protocol.Header)
	}
	return rw.response.Trailer
}

func (rw *ResponseWriter) SetCookie(cookie protocol.Cookie) error {
	value := cookie.String()
	if value == "" {
		return fmt.Errorf("invalid cookie")
	}
	rw.Header().Add("Set-Cookie", value)
	return nil
}

// StatusCode sets the response status. Changes after Flush have no effect.
func (rw *ResponseWriter) StatusCode(status int) {
	rw.response.StatusCode = status
}

func (rw *ResponseWriter) Write(data []byte) error {
	if rw.finished {
		return io.ErrClosedPipe
	}
	_, err := rw.body.Write(data)
	return err
}

// Flush sends buffered data without ending the response. When no explicit
// Content-Length is present it switches the response to chunked framing.
func (rw *ResponseWriter) Flush() error {
	if rw.finished {
		return io.ErrClosedPipe
	}
	if !rw.headerWritten {
		if rw.response.Header == nil {
			rw.response.Header = make(protocol.Header)
		}
		var err error
		rw.chunked, err = isChunked(rw.response.Header)
		if err != nil {
			return err
		}
		if !rw.chunked && rw.response.Header.Get("Content-Length") == "" {
			rw.response.Header.Set("Transfer-Encoding", "chunked")
			rw.chunked = true
		}
		if rw.chunked {
			rw.response.Header.Del("Content-Length")
			announceTrailers(rw.response.Header, rw.response.Trailer)
		}
		if err := rw.writeHeader(); err != nil {
			return err
		}
		rw.headerWritten = true
	}
	if err := rw.writePending(); err != nil {
		return err
	}
	return rw.bw.Flush()
}

// Finish writes the final response bytes. The server calls it after the
// handler returns; handlers normally use Flush only for incremental delivery.
func (rw *ResponseWriter) Finish() error {
	if rw.finished {
		return nil
	}
	rw.finished = true
	if !rw.headerWritten {
		rw.response.Body = io.NopCloser(bytes.NewReader(rw.body.Bytes()))
		rw.body.Reset()
		if err := WriteResponse(rw.bw, rw.response); err != nil {
			return err
		}
		return rw.bw.Flush()
	}
	if err := rw.writePending(); err != nil {
		return err
	}
	if rw.chunked {
		if err := writeChunked(rw.bw, nil, rw.response.Trailer); err != nil {
			return err
		}
	}
	return rw.bw.Flush()
}

func (rw *ResponseWriter) writePending() error {
	if rw.body.Len() == 0 {
		return nil
	}
	data := append([]byte(nil), rw.body.Bytes()...)
	rw.body.Reset()
	if rw.chunked {
		return writeChunk(rw.bw, data)
	}
	_, err := rw.bw.Write(data)
	return err
}

func (rw *ResponseWriter) writeHeader() error {
	proto := rw.response.Protocol
	if proto == "" {
		proto = "HTTP/1.1"
	}
	if _, err := fmt.Fprintf(rw.bw, "%s %d %s\r\n", proto, rw.response.StatusCode, protocol.StatusText(rw.response.StatusCode)); err != nil {
		return err
	}
	for key, values := range rw.response.Header {
		for _, value := range values {
			if _, err := fmt.Fprintf(rw.bw, "%s: %s\r\n", key, value); err != nil {
				return err
			}
		}
	}
	_, err := io.WriteString(rw.bw, "\r\n")
	return err
}
