package client

import (
	"io"
	"net"
	"testing"

	"github.com/23jdd/Nexo/protocol/http1"
	"github.com/23jdd/Nexo/server"
)

func TestClientCallsServerAndReusesConnection(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := server.New()
	s.GET("/hello", func(w *http1.ResponseWriter, r *http1.Request) {
		_ = w.Write([]byte("hello " + r.Query("name")))
	})
	done := make(chan error, 1)
	go func() { done <- s.Serve(lis) }()
	t.Cleanup(func() {
		_ = s.Close()
		<-done
	})

	c := New()
	transport := c.Transport.(*HTTP1Transport)
	t.Cleanup(transport.CloseIdleConnections)
	for _, name := range []string{"sam", "nexo"} {
		resp, err := c.Get("http://" + lis.Addr().String() + "/hello?name=" + name)
		if err != nil {
			t.Fatalf("Get(): %v", err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(body), "hello "+name; got != want {
			t.Fatalf("body = %q, want %q", got, want)
		}
	}
	transport.mu.Lock()
	connections := len(transport.pool)
	transport.mu.Unlock()
	if connections != 1 {
		t.Fatalf("connection pool size = %d, want 1", connections)
	}
}
