package client

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/23jdd/Nexo/protocol/http1"
)

type pooledConn struct {
	conn net.Conn
	br   *bufio.Reader
	mu   sync.Mutex
}

type HTTP1Transport struct {
	DialTimeout time.Duration
	mu          sync.Mutex
	pool        map[string]*pooledConn
}

func NewHTTP1Transport() *HTTP1Transport {
	return &HTTP1Transport{DialTimeout: 10 * time.Second, pool: make(map[string]*pooledConn)}
}

func (t *HTTP1Transport) RoundTrip(req *http1.Request) (*http1.Response, error) {
	if req == nil || req.URL == nil {
		return nil, fmt.Errorf("nil request or URL")
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme %q", req.URL.Scheme)
	}
	address := req.URL.Host
	if !strings.Contains(address, ":") {
		if req.URL.Scheme == "https" {
			address += ":443"
		} else {
			address += ":80"
		}
	}
	key := req.URL.Scheme + "://" + address
	pc, err := t.connection(key, address, req.URL.Scheme, req.URL.Hostname())
	if err != nil {
		return nil, err
	}
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if err := http1.WriteRequest(pc.conn, req); err != nil {
		t.discard(key, pc)
		return nil, err
	}
	resp, err := http1.ReadResponse(pc.br)
	if err != nil {
		t.discard(key, pc)
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.discard(key, pc)
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if strings.EqualFold(resp.Header.Get("Connection"), "close") {
		t.discard(key, pc)
	}
	return resp, nil
}

func (t *HTTP1Transport) connection(key, address, scheme, serverName string) (*pooledConn, error) {
	t.mu.Lock()
	if pc := t.pool[key]; pc != nil {
		t.mu.Unlock()
		return pc, nil
	}
	t.mu.Unlock()
	dialer := &net.Dialer{Timeout: t.DialTimeout}
	var conn net.Conn
	var err error
	if scheme == "https" {
		conn, err = tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
			ServerName: serverName, NextProtos: []string{"http/1.1"}, MinVersion: tls.VersionTLS12,
		})
	} else {
		conn, err = dialer.Dial("tcp", address)
	}
	if err != nil {
		return nil, err
	}
	pc := &pooledConn{conn: conn, br: bufio.NewReader(conn)}
	t.mu.Lock()
	if existing := t.pool[key]; existing != nil {
		t.mu.Unlock()
		_ = conn.Close()
		return existing, nil
	}
	t.pool[key] = pc
	t.mu.Unlock()
	return pc, nil
}

func (t *HTTP1Transport) discard(key string, pc *pooledConn) {
	t.mu.Lock()
	if t.pool[key] == pc {
		delete(t.pool, key)
	}
	t.mu.Unlock()
	_ = pc.conn.Close()
}

func (t *HTTP1Transport) CloseIdleConnections() {
	t.mu.Lock()
	pool := t.pool
	t.pool = make(map[string]*pooledConn)
	t.mu.Unlock()
	for _, pc := range pool {
		_ = pc.conn.Close()
	}
}
