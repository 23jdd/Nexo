package server

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/23jdd/Nexo/protocol"
	"github.com/23jdd/Nexo/protocol/http1"
)

type HttpServer struct {
	lis          net.Listener
	m            map[string]map[string]Handler
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func New() *HttpServer {
	return &HttpServer{
		m: make(map[string]map[string]Handler),
	}
}
func (hs *HttpServer) Run(address string, port int) error {
	host := net.JoinHostPort(address, strconv.Itoa(port))
	lis, err := net.Listen("tcp", host)
	if err != nil {
		return err
	}
	return hs.Serve(lis)
}

// Serve accepts HTTP/1.x connections from lis until the listener is closed.
func (hs *HttpServer) Serve(lis net.Listener) error {
	hs.lis = lis
	defer func() {
		if err := lis.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			log.Println(err)
		}
	}()
	for {
		con, err := lis.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go hs.handler(con)
	}
}

func (hs *HttpServer) handler(con net.Conn) {
	defer con.Close()
	reader := bufio.NewReader(con)
	firstRequest := true
	for {
		readTimeout := hs.IdleTimeout
		if firstRequest || readTimeout == 0 {
			readTimeout = hs.ReadTimeout
		}
		if readTimeout > 0 {
			_ = con.SetReadDeadline(time.Now().Add(readTimeout))
		}
		request, err := http1.ReadRequest(reader)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Println(err)
			}
			return
		}
		firstRequest = false
		if hs.WriteTimeout > 0 {
			_ = con.SetWriteDeadline(time.Now().Add(hs.WriteTimeout))
		}
		path := request.URL.Path
		handler := hs.lookup(request.Method, path)
		writer := http1.NewResponseWriter(con, &http1.Response{
			Protocol: request.Protocol, Header: make(protocol.Header), StatusCode: protocol.StatusOK,
		})
		if handler == nil {
			writer.StatusCode(protocol.StatusNotFound)
			_ = writer.Write([]byte("404 page not found\n"))
		} else {
			handler(writer, request)
		}
		// Consume any unread request bytes before parsing the next message on a
		// persistent connection.
		_, _ = io.Copy(io.Discard, request.Body)
		_ = request.Body.Close()
		if strings.EqualFold(request.Header.Get("Connection"), "close") {
			writer.Header().Set("Connection", "close")
		}
		if err := writer.Finish(); err != nil {
			return
		}
		if strings.EqualFold(request.Header.Get("Connection"), "close") || request.Protocol == "HTTP/1.0" {
			return
		}
	}
}

func (hs *HttpServer) lookup(method, path string) Handler {
	if methods := hs.m[path]; methods != nil {
		if handler := methods[method]; handler != nil {
			return handler
		}
		return methods[""]
	}
	return nil
}

// Close stops accepting new connections.
func (hs *HttpServer) Close() error {
	if hs.lis == nil {
		return nil
	}
	return hs.lis.Close()
}
