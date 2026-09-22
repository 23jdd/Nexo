# Nexo
A modern HTTP library for Go, supporting both client and server with HTTP/1.1 and HTTP/2.

The first milestone provides an HTTP/1.1 server and client with request/response
parsing, case-insensitive headers, Content-Length framing, routing, persistent
connections, and a small connection pool.

```go
s := server.New()
s.GET("/hello", func(w *http1.ResponseWriter, r *http1.Request) {
    _ = w.Write([]byte("hello"))
})
log.Fatal(s.Run("127.0.0.1", 8080))
```

```go
c := client.New()
resp, err := c.Get("http://127.0.0.1:8080/hello")
```
