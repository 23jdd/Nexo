package http1

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"

	"github.com/23jdd/Nexo/protocol"
)

type Request struct {
	Protocol string
	Method   string
	URL      *url.URL
	Header   protocol.Header
	Body     io.ReadCloser
}

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
	req.Protocol = prt
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
	sawCarriageReturn := false
	var b bytes.Buffer
	for {
		_, err := reader.Read(buff)
		if err != nil {
			return "", err
		}

		if sawCarriageReturn {
			if buff[0] == '\n' {
				break
			}
			b.WriteByte('\r')
			sawCarriageReturn = false
		}

		if buff[0] == '\r' {
			sawCarriageReturn = true
			continue
		}
		b.WriteByte(buff[0])
	}
	return b.String(), nil

}

func (req *Request) Query(key string) string {
	return req.URL.Query().Get(key)
}
func (req *Request) DefaultQuery(key string, def string) string {
	value, ok := req.URL.Query()[key]
	if !ok {
		return def
	} else {
		return value[0]
	}
}
func (req *Request) QueryS(key string) []string {
	return req.URL.Query()[key]
}
