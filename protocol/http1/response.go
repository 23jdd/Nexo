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
	length := int64(0)
	if value := resp.Header.Get("Content-Length"); value != "" {
		length, err = strconv.ParseInt(value, 10, 64)
		if err != nil || length < 0 {
			return nil, fmt.Errorf("invalid Content-Length %q", value)
		}
	}
	resp.Body = io.NopCloser(io.LimitReader(br, length))
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
	if response.Header.Get("Content-Length") == "" {
		response.Header.Set("Content-Length", strconv.Itoa(len(body)))
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
	if _, err = io.WriteString(bw, "\r\n"); err == nil && len(body) > 0 {
		_, err = bw.Write(body)
	}
	if err != nil {
		return err
	}
	return bw.Flush()
}
