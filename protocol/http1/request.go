package http1

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strconv"
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

func ReadRequest(reader io.Reader) (*Request, error) {
	br, ok := reader.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(reader)
	}
	req := &Request{}
	req.Header = make(protocol.Header)
	requestLine, err := readLine(br)
	if err != nil {
		return nil, err
	}
	splits := strings.SplitN(string(requestLine), " ", 3)
	if len(splits) != 3 {
		return nil, fmt.Errorf("invalid request line: %s", requestLine)
	}
	method, path, prt := splits[0], splits[1], splits[2]
	req.Method = method
	req.URL, err = url.Parse(path)
	if err != nil {
		return nil, err
	}
	req.Protocol = prt
	for {
		line, err := readLine(br)
		if err != nil {
			return nil, err
		}
		if len(line) == 0 {
			break
		}
		headers := strings.SplitN(line, ":", 2)
		if len(headers) != 2 {
			return nil, fmt.Errorf("invalid header line: %s:%d", line, len(line))
		}
		key, value := strings.TrimSpace(headers[0]), strings.TrimSpace(headers[1])
		if key == "" {
			return nil, fmt.Errorf("invalid empty header name")
		}
		req.Header.Add(key, value)
	}
	length := int64(0)
	if value := req.Header.Get("Content-Length"); value != "" {
		length, err = strconv.ParseInt(value, 10, 64)
		if err != nil || length < 0 {
			return nil, fmt.Errorf("invalid Content-Length %q", value)
		}
	}
	req.Body = io.NopCloser(io.LimitReader(br, length))
	return req, nil
}

// WriteRequest serializes an HTTP/1.x request, including a Content-Length when
// a body is present and no framing header was supplied by the caller.
func WriteRequest(w io.Writer, req *Request) error {
	if req == nil || req.URL == nil {
		return fmt.Errorf("nil request or URL")
	}
	proto := req.Protocol
	if proto == "" {
		proto = "HTTP/1.1"
	}
	target := req.URL.RequestURI()
	if target == "" {
		target = "/"
	}
	var body []byte
	var err error
	if req.Body != nil {
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return err
		}
	}
	if req.Header == nil {
		req.Header = make(protocol.Header)
	}
	if req.URL.Host != "" && req.Header.Get("Host") == "" {
		req.Header.Set("Host", req.URL.Host)
	}
	if len(body) > 0 && req.Header.Get("Content-Length") == "" {
		req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	}
	bw := bufio.NewWriter(w)
	if _, err = fmt.Fprintf(bw, "%s %s %s\r\n", req.Method, target, proto); err != nil {
		return err
	}
	for key, values := range req.Header {
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
