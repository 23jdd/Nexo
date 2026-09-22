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
