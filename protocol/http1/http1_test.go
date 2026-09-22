package http1

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/url"
	"strings"
	"testing"

	"github.com/23jdd/Nexo/protocol"
)

func TestReadRequestSeparatesRequestLineAndHeaders(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	go func() {
		_, _ = io.WriteString(clientConn, "GET / HTTP/1.1\r\nHost: localhost:9999\r\n\r\n")
	}()

	request, err := ReadRequest(serverConn)
	if err != nil {
		t.Fatalf("ReadRequest() error = %v", err)
	}
	if request.Protocol != "HTTP/1.1" {
		t.Fatalf("Protocol = %q, want %q", request.Protocol, "HTTP/1.1")
	}
	if host := request.Header.Get("Host"); host != "localhost:9999" {
		t.Fatalf("Host = %q, want %q", host, "localhost:9999")
	}
}

func TestReadRequestUsesContentLengthAndPreservesNextRequest(t *testing.T) {
	wire := "POST /first HTTP/1.1\r\nContent-Length: 5\r\n\r\nhelloGET /next HTTP/1.1\r\n\r\n"
	reader := bufio.NewReader(strings.NewReader(wire))
	first, err := ReadRequest(reader)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(first.Body)
	if err != nil || string(body) != "hello" {
		t.Fatalf("body = %q, err = %v", body, err)
	}
	second, err := ReadRequest(reader)
	if err != nil {
		t.Fatal(err)
	}
	if second.URL.Path != "/next" {
		t.Fatalf("second path = %q", second.URL.Path)
	}
}

func TestRequestAndResponseRoundTrip(t *testing.T) {
	requestWire := bytes.Buffer{}
	req := &Request{Protocol: "HTTP/1.1", Method: "POST", URL: mustURL(t, "http://example.com/items?q=1"), Header: make(protocol.Header), Body: io.NopCloser(strings.NewReader("data"))}
	if err := WriteRequest(&requestWire, req); err != nil {
		t.Fatal(err)
	}
	parsed, err := ReadRequest(bufio.NewReader(&requestWire))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.URL.RequestURI() != "/items?q=1" || parsed.Header.Get("Host") != "example.com" {
		t.Fatalf("parsed request = %#v", parsed)
	}
}

func TestChunkedRequestWithExtensionAndTrailer(t *testing.T) {
	wire := "POST /upload HTTP/1.1\r\nTransfer-Encoding: chunked\r\nTrailer: X-Checksum\r\n\r\n" +
		"5;name=value\r\nhello\r\n6\r\n world\r\n0\r\nX-Checksum: ok\r\n\r\n" +
		"GET /next HTTP/1.1\r\n\r\n"
	reader := bufio.NewReader(strings.NewReader(wire))
	req, err := ReadRequest(reader)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(req.Body)
	if err != nil || string(body) != "hello world" {
		t.Fatalf("body = %q, err = %v", body, err)
	}
	if got := req.Trailer.Get("x-checksum"); got != "ok" {
		t.Fatalf("trailer = %q", got)
	}
	next, err := ReadRequest(reader)
	if err != nil || next.URL.Path != "/next" {
		t.Fatalf("next request = %#v, err = %v", next, err)
	}
}

func TestChunkedResponseRoundTrip(t *testing.T) {
	var wire bytes.Buffer
	resp := &Response{
		Protocol: "HTTP/1.1", StatusCode: protocol.StatusOK,
		Header:  protocol.Header{"Transfer-Encoding": {"chunked"}},
		Trailer: protocol.Header{"X-Checksum": {"ok"}},
		Body:    io.NopCloser(strings.NewReader("hello")),
	}
	if err := WriteResponse(&wire, resp); err != nil {
		t.Fatal(err)
	}
	parsed, err := ReadResponse(bufio.NewReader(&wire))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(parsed.Body)
	if err != nil || string(body) != "hello" {
		t.Fatalf("body = %q, err = %v", body, err)
	}
	if got := parsed.Trailer.Get("X-Checksum"); got != "ok" {
		t.Fatalf("trailer = %q", got)
	}
}

func TestRejectsAmbiguousMessageFraming(t *testing.T) {
	wire := "POST / HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Length: 3\r\n\r\n"
	if _, err := ReadRequest(strings.NewReader(wire)); err == nil {
		t.Fatal("ReadRequest() accepted Transfer-Encoding with Content-Length")
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestResponseWriterWritesValidStatusLine(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	done := make(chan error, 1)
	go func() {
		writer := NewResponseWriter(serverConn, &Response{
			Protocol:   "HTTP/1.1",
			StatusCode: protocol.StatusOK,
			Header:     make(protocol.Header),
		})
		if err := writer.Write([]byte("hello")); err != nil {
			done <- err
			return
		}
		if err := writer.Flush(); err != nil {
			done <- err
			return
		}
		done <- serverConn.Close()
	}()

	response, err := io.ReadAll(clientConn)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("writing response: %v", err)
	}
	if !strings.HasPrefix(string(response), "HTTP/1.1 200 OK\r\n") {
		t.Fatalf("response = %q, want valid status line", response)
	}
}
