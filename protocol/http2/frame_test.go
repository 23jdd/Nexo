package http2

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestFrameHeaderRoundTrip(t *testing.T) {
	original := FrameHeader{Length: 0x10203, Type: FrameHeaders, Flags: FlagEndHeaders | FlagEndStream, StreamID: 0x1234567}
	wire, err := original.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var parsed FrameHeader
	if err := parsed.UnmarshalBinary(wire); err != nil {
		t.Fatal(err)
	}
	if parsed != original {
		t.Fatalf("header = %#v, want %#v", parsed, original)
	}
	wire[5] |= 0x80
	if err := parsed.UnmarshalBinary(wire); err != nil || parsed.StreamID != original.StreamID {
		t.Fatalf("reserved bit was not ignored: %#v, %v", parsed, err)
	}
}

func TestControlFramesRoundTrip(t *testing.T) {
	settings, err := NewSettingsFrame([]Setting{{ID: SettingInitialWindowSize, Value: 100000}, {ID: SettingMaxFrameSize, Value: 32768}}, false)
	if err != nil {
		t.Fatal(err)
	}
	ping := NewPingFrame([8]byte{1, 2, 3, 4, 5, 6, 7, 8}, true)
	goAway, err := NewGoAwayFrame(7, ErrorNo, []byte("shutdown"))
	if err != nil {
		t.Fatal(err)
	}
	rst, err := NewRSTStreamFrame(3, ErrorCancel)
	if err != nil {
		t.Fatal(err)
	}
	window, err := NewWindowUpdateFrame(3, 1024)
	if err != nil {
		t.Fatal(err)
	}
	for _, original := range []*Frame{settings, ping, goAway, rst, window} {
		var wire bytes.Buffer
		if err := WriteFrame(&wire, original); err != nil {
			t.Fatal(err)
		}
		parsed, err := ReadFrameLimit(&wire, MaxFrameSize)
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Header != original.Header || !bytes.Equal(parsed.Payload, original.Payload) {
			t.Fatalf("frame = %#v, want %#v", parsed, original)
		}
	}

	parsedSettings, err := ParseSettings(settings)
	if err != nil || len(parsedSettings) != 2 || parsedSettings[1].Value != 32768 {
		t.Fatalf("settings = %#v, err = %v", parsedSettings, err)
	}
	pong, err := ParsePing(ping)
	if err != nil || pong[7] != 8 {
		t.Fatalf("ping = %#v, err = %v", pong, err)
	}
	lastID, code, debug, err := ParseGoAway(goAway)
	if err != nil || lastID != 7 || code != ErrorNo || string(debug) != "shutdown" {
		t.Fatalf("GOAWAY = %d, %d, %q, %v", lastID, code, debug, err)
	}
	if code, err := ParseRSTStream(rst); err != nil || code != ErrorCancel {
		t.Fatalf("RST_STREAM = %d, %v", code, err)
	}
	if increment, err := ParseWindowUpdate(window); err != nil || increment != 1024 {
		t.Fatalf("WINDOW_UPDATE = %d, %v", increment, err)
	}
}

func TestDataAndHeadersFrames(t *testing.T) {
	data, err := NewDataFrame(1, []byte("hello"), true)
	if err != nil || data.Header.Flags&FlagEndStream == 0 {
		t.Fatalf("DATA = %#v, %v", data, err)
	}
	headers, err := NewHeadersFrame(1, []byte{0x82, 0x84}, true, false)
	if err != nil || headers.Header.Flags&FlagEndHeaders == 0 {
		t.Fatalf("HEADERS = %#v, %v", headers, err)
	}
	if _, err := NewDataFrame(0, nil, false); err == nil {
		t.Fatal("NewDataFrame accepted stream 0")
	}
}

func TestRejectsInvalidFrames(t *testing.T) {
	invalidSettings := newFrame(FrameSettings, FlagAck, 0, make([]byte, 6))
	if err := invalidSettings.Validate(); err == nil {
		t.Fatal("accepted SETTINGS ACK payload")
	}
	if _, err := NewSettingsFrame([]Setting{{ID: SettingEnablePush, Value: 2}}, false); err == nil {
		t.Fatal("accepted invalid SETTINGS_ENABLE_PUSH")
	}
	if _, err := NewWindowUpdateFrame(0, 0); err == nil {
		t.Fatal("accepted zero window increment")
	}

	var wire bytes.Buffer
	header, _ := (FrameHeader{Length: DefaultMaxFrameSize + 1, Type: FrameData, StreamID: 1}).MarshalBinary()
	wire.Write(header)
	wire.Write(make([]byte, DefaultMaxFrameSize+1))
	if _, err := ReadFrame(&wire); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("ReadFrame() error = %v", err)
	}
}

func TestFrameWireFormat(t *testing.T) {
	frame, err := NewWindowUpdateFrame(0, 65535)
	if err != nil {
		t.Fatal(err)
	}
	var wire bytes.Buffer
	if err := WriteFrame(&wire, frame); err != nil {
		t.Fatal(err)
	}
	data := wire.Bytes()
	if len(data) != FrameHeaderSize+4 || data[2] != 4 || data[3] != byte(FrameWindowUpdate) || binary.BigEndian.Uint32(data[9:]) != 65535 {
		t.Fatalf("wire = %x", data)
	}
}
