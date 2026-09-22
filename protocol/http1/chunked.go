package http1

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/23jdd/Nexo/protocol"
)

type chunkedReader struct {
	r         *bufio.Reader
	trailer   protocol.Header
	remaining uint64
	needCRLF  bool
	done      bool
}

func newChunkedReader(r *bufio.Reader, trailer protocol.Header) io.ReadCloser {
	return &chunkedReader{r: r, trailer: trailer}
}

func (r *chunkedReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.done {
		return 0, io.EOF
	}
	if r.remaining == 0 {
		if r.needCRLF {
			line, err := readLine(r.r)
			if err != nil {
				return 0, err
			}
			if line != "" {
				return 0, fmt.Errorf("invalid chunk terminator")
			}
			r.needCRLF = false
		}
		line, err := readLine(r.r)
		if err != nil {
			return 0, err
		}
		sizeText := strings.TrimSpace(strings.SplitN(line, ";", 2)[0])
		if sizeText == "" {
			return 0, fmt.Errorf("invalid empty chunk size")
		}
		r.remaining, err = strconv.ParseUint(sizeText, 16, 63)
		if err != nil {
			return 0, fmt.Errorf("invalid chunk size %q", sizeText)
		}
		if r.remaining == 0 {
			if err := readTrailers(r.r, r.trailer); err != nil {
				return 0, err
			}
			r.done = true
			return 0, io.EOF
		}
	}
	limit := uint64(len(p))
	if limit > r.remaining {
		limit = r.remaining
	}
	n, err := io.ReadFull(r.r, p[:int(limit)])
	r.remaining -= uint64(n)
	if r.remaining == 0 {
		r.needCRLF = true
	}
	return n, err
}

func (r *chunkedReader) Close() error { return nil }

func readTrailers(r *bufio.Reader, trailer protocol.Header) error {
	for {
		line, err := readLine(r)
		if err != nil {
			return err
		}
		if line == "" {
			return nil
		}
		field := strings.SplitN(line, ":", 2)
		if len(field) != 2 || strings.TrimSpace(field[0]) == "" {
			return fmt.Errorf("invalid trailer line: %s", line)
		}
		trailer.Add(strings.TrimSpace(field[0]), strings.TrimSpace(field[1]))
	}
}

func writeChunked(w *bufio.Writer, body []byte, trailer protocol.Header) error {
	const chunkSize = 32 * 1024
	for len(body) > 0 {
		n := len(body)
		if n > chunkSize {
			n = chunkSize
		}
		if err := writeChunk(w, body[:n]); err != nil {
			return err
		}
		body = body[n:]
	}
	if _, err := io.WriteString(w, "0\r\n"); err != nil {
		return err
	}
	for key, values := range trailer {
		for _, value := range values {
			if _, err := fmt.Fprintf(w, "%s: %s\r\n", key, value); err != nil {
				return err
			}
		}
	}
	_, err := io.WriteString(w, "\r\n")
	return err
}

func writeChunk(w *bufio.Writer, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(w, "%x\r\n", len(data)); err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\r\n")
	return err
}

func announceTrailers(header, trailer protocol.Header) {
	if len(trailer) == 0 {
		return
	}
	keys := make([]string, 0, len(trailer))
	for key := range trailer {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	header.Set("Trailer", strings.Join(keys, ", "))
}

func isChunked(header protocol.Header) (bool, error) {
	values := header.Values("Transfer-Encoding")
	if len(values) == 0 {
		return false, nil
	}
	var codings []string
	for _, value := range values {
		for _, coding := range strings.Split(value, ",") {
			codings = append(codings, strings.ToLower(strings.TrimSpace(coding)))
		}
	}
	if len(codings) != 1 || codings[0] != "chunked" {
		return false, fmt.Errorf("unsupported Transfer-Encoding %q", strings.Join(codings, ", "))
	}
	return true, nil
}

func contentLength(header protocol.Header) (int64, bool, error) {
	values := header.Values("Content-Length")
	if len(values) == 0 {
		return 0, false, nil
	}
	var length int64
	for index, value := range values {
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || parsed < 0 {
			return 0, false, fmt.Errorf("invalid Content-Length %q", value)
		}
		if index > 0 && parsed != length {
			return 0, false, fmt.Errorf("conflicting Content-Length values")
		}
		length = parsed
	}
	return length, true, nil
}
