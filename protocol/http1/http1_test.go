package http1

import (
	"io"
	"net"
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
