package online

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestManagerBroadcastsConnectionCount(t *testing.T) {
	manager := NewManager(nil, nil)
	server := httptest.NewServer(manager)
	defer server.Close()
	defer manager.Close()

	first, firstReader := dialWebSocket(t, server.URL)
	defer first.Close()
	assertOnlineCount(t, first, firstReader, 1)

	second, secondReader := dialWebSocket(t, server.URL)
	assertOnlineCount(t, first, firstReader, 2)
	assertOnlineCount(t, second, secondReader, 2)
	_ = second.Close()
	assertOnlineCount(t, first, firstReader, 1)
}

func dialWebSocket(t *testing.T, serverURL string) (net.Conn, *bufio.Reader) {
	t.Helper()
	parsed, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	connection, err := net.Dial("tcp", parsed.Host)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	request := "GET / HTTP/1.1\r\n" +
		"Host: " + parsed.Host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n"
	if _, err := io.WriteString(connection, request); err != nil {
		_ = connection.Close()
		t.Fatalf("write websocket handshake: %v", err)
	}
	reader := bufio.NewReader(connection)
	response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodGet})
	if err != nil {
		_ = connection.Close()
		t.Fatalf("read websocket handshake: %v", err)
	}
	if response.StatusCode != http.StatusSwitchingProtocols {
		_ = connection.Close()
		t.Fatalf("websocket status = %d, want 101", response.StatusCode)
	}
	return connection, reader
}

func assertOnlineCount(t *testing.T, connection net.Conn, reader *bufio.Reader, want int) {
	t.Helper()
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	payload, err := readServerTextFrame(reader)
	if err != nil {
		t.Fatalf("read online message: %v", err)
	}
	var message message
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatalf("decode online message: %v", err)
	}
	if message.Type != "online_count" || message.Count != want {
		t.Fatalf("online message = %#v, want online_count %d", message, want)
	}
}

func readServerTextFrame(reader *bufio.Reader) ([]byte, error) {
	first, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	second, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	if first != 0x81 || second&0x80 != 0 {
		return nil, io.ErrUnexpectedEOF
	}
	length := uint64(second & 0x7f)
	if length == 126 {
		var extended [2]byte
		if _, err := io.ReadFull(reader, extended[:]); err != nil {
			return nil, err
		}
		length = uint64(binary.BigEndian.Uint16(extended[:]))
	}
	payload := make([]byte, length)
	_, err = io.ReadFull(reader, payload)
	return payload, err
}
