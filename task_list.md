# HTTP Protocol Summary

本文档用于开发一个 Go HTTP Library，目标支持：

* HTTP Server
* HTTP Client
* HTTP/1.1
* HTTP/2
* TLS
* ALPN
* Routing
* Middleware
* Streaming
* Connection Management

---

## 1. HTTP 协议结构

HTTP 可以拆成两层：

```text
HTTP Semantics
      │
      ├── Method
      ├── Status Code
      ├── Header
      ├── Content
      ├── Cache
      └── Authentication
      │
      ▼
HTTP Wire Protocol
      │
      ├── HTTP/1.1
      ├── HTTP/2
      └── HTTP/3
```

其中：

```text
RFC 9110 → HTTP Semantics
RFC 9112 → HTTP/1.1
RFC 9113 → HTTP/2
```

HTTP/1.1 和 HTTP/2 的编码方式不同，但共享相同的 HTTP 语义。

例如：

```http
GET /users HTTP/1.1
```

HTTP/2 中对应：

```text
:method = GET
:path   = /users
```

---

# 2. 核心 RFC

## RFC 9110 — HTTP Semantics

主要定义：

```text
Request
Response
Method
Status Code
Header Field
Content
URI
Authentication
Conditional Request
Range Request
```

常见 Method：

```text
GET
HEAD
POST
PUT
DELETE
CONNECT
OPTIONS
TRACE
```

常见状态码：

```text
1xx Informational
2xx Success
3xx Redirection
4xx Client Error
5xx Server Error
```

典型抽象：

```go
type Request struct {
    Method string
    URL    *URL
    Header Header
    Body   io.ReadCloser
}

type Response struct {
    StatusCode int
    Header     Header
    Body       io.ReadCloser
}
```

参考：

```text
RFC 9110
https://www.rfc-editor.org/rfc/rfc9110.html
```

---

# 3. HTTP/1.1

规范：

```text
RFC 9112
```

HTTP/1.1 是文本协议。

Request 示例：

```http
POST /users?id=10 HTTP/1.1
Host: example.com
Content-Type: application/json
Content-Length: 16

{"name":"sam"}
```

Response：

```http
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 15

{"status":"ok"}
```

---

## 3.1 HTTP/1.1 Message

基本结构：

```text
start-line
header-field
header-field
header-field
CRLF
body
```

Request：

```text
request-line
headers
CRLF
body
```

Response：

```text
status-line
headers
CRLF
body
```

---

## 3.2 Request Line

格式：

```text
METHOD SP request-target SP HTTP-version CRLF
```

例如：

```text
GET /users/10 HTTP/1.1\r\n
```

可以解析成：

```go
type RequestLine struct {
    Method string
    Target string
    Proto  string
}
```

---

## 3.3 Status Line

格式：

```text
HTTP-version SP status-code SP reason-phrase CRLF
```

例如：

```text
HTTP/1.1 200 OK\r\n
```

---

# 4. Header

Header 基本形式：

```text
Name: Value
```

例如：

```text
Host: example.com
Content-Type: application/json
Content-Length: 123
Connection: keep-alive
```

Go 中可以设计：

```go
type Header map[string][]string
```

API：

```go
func (h Header) Get(key string) string
func (h Header) Set(key, value string)
func (h Header) Add(key, value string)
func (h Header) Del(key string)
func (h Header) Values(key string) []string
```

注意 Header 名称：

```text
case-insensitive
```

也就是说：

```text
Content-Type
content-type
CONTENT-TYPE
```

语义上相同。

---

# 5. Message Body

HTTP Message 是否包含 Body 取决于：

```text
Method
Status Code
Content-Length
Transfer-Encoding
Connection state
```

常见情况：

```text
Content-Length: 123
```

表示读取 123 bytes。

---

# 6. Content-Length

例如：


```http
Content-Length: 5

hello
```

Reader：

```text
读取 exactly 5 bytes
```

不能：

```text
一直读到 EOF
```

因为 HTTP/1.1 Connection 可能会继续用于下一个请求。

---

# 7. Transfer-Encoding: chunked

HTTP/1.1 可以使用：

```http
Transfer-Encoding: chunked
```

数据：

```text
5\r\n
hello\r\n
6\r\n
 world\r\n
0\r\n
\r\n
```

格式：

```text
chunk-size CRLF
chunk-data CRLF

chunk-size CRLF
chunk-data CRLF

0 CRLF
CRLF
```

实现：

```go
type ChunkedReader struct {
    r *bufio.Reader
}
```

需要处理：

```text
hex chunk size
chunk extension
final chunk
trailers
```

---

# 8. Persistent Connection

HTTP/1.1 默认支持 persistent connection。

例如：

```text
TCP Connection
│
├── Request 1
├── Response 1
│
├── Request 2
├── Response 2
│
└── Request 3
```

不需要每个 HTTP Request 都重新创建 TCP Connection。

Server Connection Loop：

```go
func serveConn(conn net.Conn) {
    for {
        req, err := readRequest(conn)
        if err != nil {
            return
        }

        resp := handle(req)

        if err := writeResponse(conn, resp); err != nil {
            return
        }
    }
}
```

---

# 9. Connection Close

可以通过：

```http
Connection: close
```

告诉对方当前请求结束后关闭连接。

Server：

```text
read request
    ↓
handle request
    ↓
write response
    ↓
close TCP
```

---

# 10. HTTP/2

规范：

```text
RFC 9113
```

HTTP/2 是 binary protocol。

不再使用：

```text
GET / HTTP/1.1\r\n
Host: ...
```

而是：

```text
Frame
```

基本结构：

```text
TCP Connection
│
├── Stream 1
│   ├── HEADERS
│   └── DATA
│
├── Stream 3
│   ├── HEADERS
│   ├── DATA
│   └── DATA
│
└── Stream 5
    ├── HEADERS
    └── DATA
```

---

# 11. HTTP/2 Connection Preface

HTTP/2 Client 建立连接后会发送 connection preface：

```text
PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n
```

然后通常发送：

```text
SETTINGS
```

Server 也会发送自己的 SETTINGS。

---

# 12. HTTP/2 Frame

HTTP/2 Frame Header 固定为：

```text
9 bytes
```

结构：

```text
+-----------------------------------------------+
| Length                  | 24 bits             |
+-----------------------------------------------+
| Type                    | 8 bits              |
+-----------------------------------------------+
| Flags                   | 8 bits              |
+-----------------------------------------------+
| R | Stream Identifier   | 1 + 31 bits         |
+-----------------------------------------------+
```

Go：

```go
type FrameHeader struct {
    Length   uint32
    Type     uint8
    Flags    uint8
    StreamID uint32
}
```

---

# 13. HTTP/2 Frame Types

主要 Frame：

```text
DATA
HEADERS
PRIORITY
RST_STREAM
SETTINGS
PUSH_PROMISE
PING
GOAWAY
WINDOW_UPDATE
CONTINUATION
```

开发初期重点：

```text
SETTINGS
HEADERS
DATA
RST_STREAM
PING
GOAWAY
WINDOW_UPDATE
```

---

# 14. Stream

HTTP/2 的核心是：

```text
Stream
```

每个 Request / Response 通常运行在一个 Stream 上。

例如：

```text
Connection
│
├── Stream 1
│   └── GET /users
│
├── Stream 3
│   └── GET /posts
│
└── Stream 5
    └── POST /login
```

这些 Stream 可以同时存在。

这就是：

```text
Multiplexing
```

---

# 15. Stream ID

HTTP/2 Stream ID：

```text
31 bits
```

客户端创建的 Stream：

```text
odd
```

例如：

```text
1
3
5
7
```

Server initiated stream 使用：

```text
even
```

---

# 16. Stream State

HTTP/2 Stream 有状态机：

```text
idle
  ↓
open
  ↓
half-closed
  ↓
closed
```

可能状态：

```text
idle
reserved
open
half-closed(local)
half-closed(remote)
closed
```

实现 HTTP/2 时，Stream State Machine 很重要。

---

# 17. HEADERS Frame

HTTP/2 Request 不使用：

```text
GET /users HTTP/1.1
```

而使用 pseudo-header。

例如：

```text
:method: GET
:scheme: https
:authority: example.com
:path: /users
```

Response：

```text
:status: 200
```

普通 Header：

```text
content-type: application/json
content-length: 100
```

---

# 18. HPACK

规范：

```text
RFC 7541
```

HTTP/2 使用 HPACK 压缩 Header。

主要组成：

```text
Static Table
Dynamic Table
Indexed Representation
Literal Representation
Huffman Coding
```

例如：

```text
content-type
user-agent
accept
authorization
```

不需要每个 Request 都完整重复传输。

可以设计：

```go
type HeaderField struct {
    Name  string
    Value string
}

type DynamicTable struct {
    entries []HeaderField
    maxSize uint32
}
```

以及：

```go
type HPACKEncoder struct {
}

type HPACKDecoder struct {
}
```

---

# 19. HTTP/2 Flow Control

HTTP/2 有 Flow Control：

```text
Connection Flow Control

+

Stream Flow Control
```

通过：

```text
WINDOW_UPDATE
```

控制发送方允许发送多少 DATA。

例如：

```text
window = 65535

send 10000 bytes

window = 55535
```

收到：

```text
WINDOW_UPDATE +20000
```

变成：

```text
75535
```

只作用于：

```text
DATA Frame
```

---

# 20. SETTINGS

SETTINGS 用于配置 HTTP/2 Connection。

常见：

```text
SETTINGS_HEADER_TABLE_SIZE

SETTINGS_ENABLE_PUSH

SETTINGS_MAX_CONCURRENT_STREAMS

SETTINGS_INITIAL_WINDOW_SIZE

SETTINGS_MAX_FRAME_SIZE

SETTINGS_MAX_HEADER_LIST_SIZE
```

双方建立 HTTP/2 Connection 时会交换 SETTINGS。

---

# 21. PING

用于：

```text
keep alive
RTT measurement
connection health check
```

PING payload：

```text
8 bytes
```

接收者返回：

```text
PING + ACK
```

---

# 22. RST_STREAM

用于终止单个 Stream。

例如：

```text
Client
   │
   ├── HEADERS
   ├── DATA
   │
   └── cancel
        ↓
    RST_STREAM
```

不会关闭整个 HTTP/2 Connection。

---

# 23. GOAWAY

GOAWAY 用于停止整个 HTTP/2 Connection。

例如：

```text
Server graceful shutdown
      ↓
GOAWAY
      ↓
不再创建新的 Stream
      ↓
等待已有 Stream 完成
      ↓
close connection
```

---

# 24. TLS

HTTPS：

```text
HTTP
 ↓
TLS
 ↓
TCP
```

Server：

```go
listener, err := tls.Listen(
    "tcp",
    ":443",
    tlsConfig,
)
```

---

# 25. ALPN

规范：

```text
RFC 7301
```

TLS Handshake 时协商协议。

Client：

```text
ALPN:
    h2
    http/1.1
```

Server：

```text
select h2
```

然后：

```text
HTTP/2
```

Go TLS：

```go
tls.Config{
    NextProtos: []string{
        "h2",
        "http/1.1",
    },
}
```

---

# 26. h2c

h2c：

```text
HTTP/2 Cleartext
```

也就是：

```text
HTTP/2
without TLS
```

主要用于：

```text
internal service
development
microservice
```

---

# 27. URI

规范：

```text
RFC 3986
```

结构：

```text
scheme://authority/path?query#fragment
```

例如：

```text
https://example.com:8080/users?id=10#profile
```

拆分：

```text
scheme    = https

authority = example.com:8080

host      = example.com

port      = 8080

path      = /users

query     = id=10

fragment  = profile
```

Go：

```go
type URL struct {
    Scheme   string
    Host     string
    Port     string
    Path     string
    RawQuery string
    Fragment string
}
```

---

# 28. Cookie

规范：

```text
RFC 6265
```

Server：

```http
Set-Cookie: session=abc123
```

Client：

```http
Cookie: session=abc123
```

需要考虑：

```text
Domain
Path
Expires
Max-Age
Secure
HttpOnly
SameSite
```

---

# 29. multipart/form-data

规范：

```text
RFC 7578
```

用于：

```text
HTML Form
File Upload
```

例如：

```http
Content-Type: multipart/form-data; boundary=abc123
```

Body：

```text
--abc123
Content-Disposition: form-data; name="username"

sam
--abc123
Content-Disposition: form-data; name="file"; filename="a.txt"
Content-Type: text/plain

hello
--abc123--
```

---

# 30. HTTP Cache

规范：

```text
RFC 9111
```

主要 Header：

```text
Cache-Control
Age
ETag
Last-Modified
If-None-Match
If-Modified-Since
Expires
Vary
```

典型流程：

```text
Client
   │
GET /image
   ↓
Server

ETag: "abc"
   ↓

Client Cache

   │
GET /image
If-None-Match: "abc"
   ↓

Server

304 Not Modified
```

---

# 31. Server Architecture

建议：

```text
Listener
   ↓
Accept
   ↓
Connection
   ↓
Protocol Detection
   │
   ├── HTTP/1.1
   │
   └── HTTP/2
   ↓
Request
   ↓
Router
   ↓
Middleware
   ↓
Handler
   ↓
Response
```

代码：

```go
type Server struct {
    Router *Router
}

func (s *Server) Serve(l net.Listener) error {
    for {
        conn, err := l.Accept()
        if err != nil {
            return err
        }

        go s.serveConn(conn)
    }
}
```

---

# 32. Client Architecture

```text
Request
   ↓
URL
   ↓
Connection Pool
   ↓
Dial
   ↓
TCP
   ↓
TLS / ALPN
   ↓
HTTP/1.1 or HTTP/2
   ↓
Write Request
   ↓
Read Response
```

结构：

```go
type Client struct {
    Transport Transport
}

type Transport interface {
    RoundTrip(*Request) (*Response, error)
}
```

这是一个很重要的抽象。

---

# 33. HTTP/1 Transport

```go
type HTTP1Transport struct {
    pool *ConnPool
}
```

负责：

```text
TCP connection
keep-alive
write request
read response
connection reuse
```

---

# 34. HTTP/2 Transport

```go
type HTTP2Transport struct {
    conns *H2ConnPool
}
```

但是 HTTP/2：

```text
一个 Connection
可以同时运行多个 Requests
```

所以它和 HTTP/1 connection pool 模型不同。

HTTP/1：

```text
Request
  ↓
Connection
```

HTTP/2：

```text
Request ─┐
Request ─┼── Connection
Request ─┘
```

---

# 35. Router

基础 API：

```go
router.GET("/users/:id", handler)

router.POST("/users", handler)
```

内部可以先：

```text
map
```

以后改：

```text
Radix Tree
```

例如：

```text
/
└── users
    ├── :id
    │   └── posts
    │       └── :postID
    │
    └── search
```

---

# 36. Middleware

```go
type Handler func(*Context) error

type Middleware func(Handler) Handler
```

执行：

```text
Request
   ↓
Recover
   ↓
Logger
   ↓
Auth
   ↓
Router
   ↓
Handler
```

---

# 37. Context

```go
type Context struct {
    Request  *Request
    Response *ResponseWriter

    Params map[string]string
}
```

常用 API：

```go
func (c *Context) Param(name string) string

func (c *Context) Query(name string) string

func (c *Context) Header(name string) string

func (c *Context) JSON(status int, v any) error

func (c *Context) String(status int, v string) error

func (c *Context) Bind(v any) error
```

---

# 38. Streaming

Server：

```go
func handler(c *Context) error {
    for {
        if _, err := c.Write(data); err != nil {
            return err
        }

        c.Flush()
    }
}
```

适合：

```text
Large Files
SSE
LLM Streaming
Proxy
```

HTTP/2 Streaming 本质是：

```text
HEADERS

DATA
DATA
DATA
DATA

END_STREAM
```

---

# 39. Graceful Shutdown

Server：

```text
Receive shutdown signal
       ↓
Stop accepting new connections
       ↓
HTTP/2 → GOAWAY
       ↓
Wait active requests
       ↓
Close connections
```

API：

```go
func (s *Server) Shutdown(ctx context.Context) error
```

---

# 40. Recommended Project Structure

```text
samnet/
│
├── server.go
├── client.go
│
├── request.go
├── response.go
├── header.go
├── url.go
├── context.go
├── handler.go
├── router.go
├── middleware.go
│
├── http1/
│   ├── conn.go
│   ├── request.go
│   ├── response.go
│   ├── parser.go
│   ├── writer.go
│   └── chunked.go
│
├── http2/
│   ├── conn.go
│   ├── stream.go
│   ├── frame.go
│   ├── frame_reader.go
│   ├── frame_writer.go
│   ├── settings.go
│   ├── flow.go
│   └── hpack/
│
├── transport/
│   ├── transport.go
│   ├── tcp.go
│   └── tls.go
│
├── router/
│   ├── router.go
│   └── node.go
│
└── middleware/
    ├── logger.go
    └── recover.go
```

---

# 41. Recommended Implementation Order

## Phase 1 — HTTP Core

实现：

```text
Header
URL
Request
Response
Body
```

---

## Phase 2 — HTTP/1 Parser

实现：

```text
Request Line
Status Line
Header Parser
Content-Length
```

API：

```go
func ReadRequest(r *bufio.Reader) (*Request, error)

func ReadResponse(r *bufio.Reader) (*Response, error)

func WriteRequest(w io.Writer, req *Request) error

func WriteResponse(w io.Writer, resp *Response) error
```

---

## Phase 3 — HTTP/1 Server

```text
net.Listen
Accept
Connection Loop
Read Request
Handler
Write Response
Keep Alive
```

---

## Phase 4 — HTTP/1 Client

```text
Dial
Write Request
Read Response
Connection reuse
Connection Pool
```

---

## Phase 5 — HTTP/1 Advanced

实现：

```text
chunked encoding
trailers
multipart
cookies
streaming
timeout
```

---

## Phase 6 — HTTP/2 Frames

首先只实现：

```text
Frame Header
SETTINGS
PING
GOAWAY
RST_STREAM
WINDOW_UPDATE
```

然后：

```text
HEADERS
DATA
```

---

## Phase 7 — HPACK

实现：

```text
Static Table
Integer Encoding
String Encoding
Indexed Header
Literal Header
Dynamic Table
Huffman
```

---

## Phase 8 — HTTP/2 Stream

实现：

```text
Stream ID
Stream State Machine
HEADERS
DATA
END_STREAM
RST_STREAM
```

---

## Phase 9 — Multiplexing

实现：

```text
Connection
│
├── Stream 1
├── Stream 3
├── Stream 5
└── Stream 7
```

需要：

```text
goroutine
channel
mutex
stream map
```

例如：

```go
type H2Conn struct {
    mu sync.Mutex

    streams map[uint32]*Stream
}
```

---

## Phase 10 — Flow Control

实现：

```text
connection window
stream window
WINDOW_UPDATE
DATA blocking
```

---

## Phase 11 — TLS + ALPN

支持：

```text
http/1.1
h2
```

TLS：

```go
tls.Config{
    NextProtos: []string{
        "h2",
        "http/1.1",
    },
}
```

---

## Phase 12 — Router / Middleware

最后再完善框架层：

```text
Router
Middleware
Context
JSON
Form
Static Files
Recovery
Logging
```

---

# 42. Most Important RFCs

开发时建议收藏：

```text
HTTP Semantics
RFC 9110
https://www.rfc-editor.org/rfc/rfc9110.html

HTTP Cache
RFC 9111
https://www.rfc-editor.org/rfc/rfc9111.html

HTTP/1.1
RFC 9112
https://www.rfc-editor.org/rfc/rfc9112.html

HTTP/2
RFC 9113
https://www.rfc-editor.org/rfc/rfc9113.html

HPACK
RFC 7541
https://www.rfc-editor.org/rfc/rfc7541.html

URI
RFC 3986
https://www.rfc-editor.org/rfc/rfc3986.html

ALPN
RFC 7301
https://www.rfc-editor.org/rfc/rfc7301.html

Cookie
RFC 6265
https://www.rfc-editor.org/rfc/rfc6265.html

multipart/form-data
RFC 7578
https://www.rfc-editor.org/rfc/rfc7578.html
```

HTTP Working Group：

```text
https://httpwg.org/specs/
```

---

# 43. Minimal First Milestone

第一阶段不要直接冲 HTTP/2。

先做到：

```text
net.Conn
   ↓
bufio.Reader
   ↓
ReadRequest()
   ↓
Request
   ↓
Handler
   ↓
Response
   ↓
WriteResponse()
```

目标：

```bash
curl http://localhost:8080/hello
```

Server：

```go
server.GET("/hello", func(c *Context) error {
    return c.String(200, "hello")
})
```

返回：

```http
HTTP/1.1 200 OK
Content-Length: 5

hello
```

然后让你自己的 Client：

```go
resp, err := client.Get("http://localhost:8080/hello")
```

能够成功调用你自己的 Server。

做到这里，就已经拥有了：

```text
HTTP parser
HTTP writer
Server
Client
TCP connection
keep-alive
Request
Response
```

之后再进入：

```text
HTTP/2
```

会顺很多。
