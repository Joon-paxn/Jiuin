package online

import (
	"bufio"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebSocketHandshakeAndInitialOnlineCount(t *testing.T) {
	manager := New(map[string]struct{}{"https://jiuin.cn": {}})
	server := httptest.NewServer(http.HandlerFunc(manager.Handle))
	t.Cleanup(func() { manager.Close(); server.Close() })
	address := strings.TrimPrefix(server.URL, "http://")
	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	key := base64.StdEncoding.EncodeToString([]byte("1234567890abcdef"))
	request := "GET /ws/online HTTP/1.1\r\nHost: " + address + "\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: " + key + "\r\nOrigin: https://jiuin.cn\r\n\r\n"
	if _, err := io.WriteString(connection, request); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(connection)
	response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodGet})
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("got %s", response.Status)
	}
	if response.Header.Get("Sec-WebSocket-Accept") != websocketAccept(key) {
		t.Fatal("invalid WebSocket accept value")
	}
	first, err := reader.ReadByte()
	if err != nil {
		t.Fatal(err)
	}
	second, err := reader.ReadByte()
	if err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, int(second&0x7f))
	if _, err := io.ReadFull(reader, payload); err != nil {
		t.Fatal(err)
	}
	if first != 0x81 || string(payload) != `{"type":"online_count","count":1}` {
		t.Fatalf("unexpected first frame: %x %q", first, payload)
	}
}
