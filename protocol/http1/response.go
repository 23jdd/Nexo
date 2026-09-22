package http1

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/23jdd/Nexo/protocol"
)

type Response struct {
	Protocol   string
	StatusCode int
	Header     protocol.Header
	Trailer    protocol.Header
	Body       io.ReadCloser
}

// HTTP/1.1 200 OK\r\n
func ReadResponse(reader io.Reader) (*Response, error) {
	br, ok := reader.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(reader)
	}
	line, err := readLine(br)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid status line: %s", line)
	}
	code, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid status code %q", parts[1])
	}
	resp := &Response{Protocol: parts[0], StatusCode: code, Header: make(protocol.Header)}
	for {
		line, err = readLine(br)
		if err != nil {
			return nil, err
		}
		if line == "" {
			break
		}
		field := strings.SplitN(line, ":", 2)
		if len(field) != 2 {
			return nil, fmt.Errorf("invalid header line: %s", line)
		}
		resp.Header.Add(strings.TrimSpace(field[0]), strings.TrimSpace(field[1]))
	}
	chunked, err := isChunked(resp.Header)
	if err != nil {
		return nil, err
	}
	length, hasLength, err := contentLength(resp.Header)
	if err != nil {
		return nil, err
	}
	if chunked && hasLength {
		return nil, fmt.Errorf("both Transfer-Encoding and Content-Length are set")
	}
	resp.Trailer = make(protocol.Header)
	if chunked {
		resp.Body = newChunkedReader(br, resp.Trailer)
	} else {
		resp.Body = io.NopCloser(io.LimitReader(br, length))
	}
	return resp, nil
}

func WriteResponse(w io.Writer, response *Response) error {
	if response == nil {
		return fmt.Errorf("nil response")
	}
	var body []byte
	var err error
	if response.Body != nil {
		body, err = io.ReadAll(response.Body)
		if err != nil {
			return err
		}
	}
	if response.Header == nil {
		response.Header = make(protocol.Header)
	}
	chunked, err := isChunked(response.Header)
	if err != nil {
		return err
	}
	if chunked {
		response.Header.Del("Content-Length")
		announceTrailers(response.Header, response.Trailer)
	} else if response.Header.Get("Content-Length") == "" {
		response.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	}
	proto := response.Protocol
	if proto == "" {
		proto = "HTTP/1.1"
	}
	bw := bufio.NewWriter(w)
	if _, err = fmt.Fprintf(bw, "%s %d %s\r\n", proto, response.StatusCode, protocol.StatusText(response.StatusCode)); err != nil {
		return err
	}
	for key, values := range response.Header {
		for _, value := range values {
			if _, err = fmt.Fprintf(bw, "%s: %s\r\n", key, value); err != nil {
				return err
			}
		}
	}
	if _, err = io.WriteString(bw, "\r\n"); err == nil {
		if chunked {
			err = writeChunked(bw, body, response.Trailer)
		} else if len(body) > 0 {
			_, err = bw.Write(body)
		}
	}
	if err != nil {
		return err
	}
	return bw.Flush()
}
