package http1

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strings"

	"github.com/23jdd/Nexo/protocol"
)

type Request struct {
	Method string
	URL    *url.URL
	Header protocol.Header
	Body   io.ReadCloser
}

// GET /users/10 HTTP/1.1\r\n
func ReadRequest(conn net.Conn) (*Request, error) {
	req := &Request{}
	req.Header = make(protocol.Header)
	statusLine, err := readLine(conn)
	if err != nil {
		return nil, err
	}
	splits := strings.SplitN(string(statusLine), " ", 3)
	if len(splits) != 3 {
		return nil, fmt.Errorf("invalid status line: %s", statusLine)
	}
	method, path, prt := splits[0], splits[1], splits[2]
	req.Method = method
	req.URL, err = url.Parse(path)
	if err != nil {
		return nil, err
	}
	log.Println(prt)
	for {
		line, err := readLine(conn)
		if err != nil {
			return nil, err
		}
		if len(line) == 0 {
			break
		}
		headers := strings.SplitN(line, ": ", 2)
		if len(headers) != 2 {
			return nil, fmt.Errorf("invalid header line: %s:%d", line, len(line))
		}
		key, value := headers[0], headers[1]
		req.Header.Set(key, value)
	}
	req.Body = conn
	return req, nil
}

func readLine(reader io.Reader) (string, error) {
	buff := make([]byte, 1)
	s1 := false
	var b bytes.Buffer
	for {
		_, err := reader.Read(buff)
		if err != nil {
			return "", err
		}
		if buff[0] == '\n' {
			s1 = true
			continue
		}
		if s1 && buff[0] == '\r' {
			break
		}
		b.WriteByte(buff[0])
	}
	return b.String(), nil

}
func (r Request) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Method:%s\n", r.Method))
	builder.WriteString(fmt.Sprintf("URL:%s\n", r.URL.String()))
	for k, v := range r.Header {
		builder.WriteString(fmt.Sprintf("header %s:%s\n", k, v))
	}
	return builder.String()
}
