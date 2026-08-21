package online

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Manager uses the RFC 6455 subset required by browser clients. Keeping this
// in the standard library makes the backup binary buildable on a locked-down
// production host without a new WebSocket dependency.
type Manager struct {
	mu      sync.Mutex
	clients map[*client]struct{}
	allowed map[string]struct{}
}

type client struct {
	connection net.Conn
	writeMu    sync.Mutex
}

func New(allowed map[string]struct{}) *Manager {
	return &Manager{clients: make(map[*client]struct{}), allowed: allowed}
}

func (m *Manager) Handle(w http.ResponseWriter, r *http.Request) {
	if !m.validUpgrade(r) {
		http.Error(w, "WebSocket upgrade required", http.StatusBadRequest)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "WebSocket is unavailable", http.StatusInternalServerError)
		return
	}
	connection, reader, err := hijacker.Hijack()
	if err != nil {
		return
	}
	accept := websocketAccept(r.Header.Get("Sec-WebSocket-Key"))
	if _, err := connection.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\n\r\n")); err != nil {
		_ = connection.Close()
		return
	}
	item := &client{connection: connection}
	m.mu.Lock()
	m.clients[item] = struct{}{}
	m.broadcastLocked()
	m.mu.Unlock()
	defer func() { m.remove(item) }()

	readDone := make(chan struct{})
	go func() { defer close(readDone); m.readLoop(item, reader) }()
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-readDone:
			return
		case <-ping.C:
			if err := item.writeFrame(0x9, []byte("ping")); err != nil {
				return
			}
		}
	}
}

func (m *Manager) validUpgrade(r *http.Request) bool {
	if r.Method != http.MethodGet || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || !headerContains(r.Header.Get("Connection"), "upgrade") || r.Header.Get("Sec-WebSocket-Version") != "13" {
		return false
	}
	key, err := base64.StdEncoding.DecodeString(r.Header.Get("Sec-WebSocket-Key"))
	if err != nil || len(key) != 16 {
		return false
	}
	_, ok := m.allowed[r.Header.Get("Origin")]
	return ok
}

func (m *Manager) readLoop(item *client, reader *bufio.ReadWriter) {
	for {
		_ = item.connection.SetReadDeadline(time.Now().Add(70 * time.Second))
		opcode, payload, err := readFrame(reader.Reader)
		if err != nil {
			return
		}
		switch opcode {
		case 0x8:
			_ = item.writeFrame(0x8, payload)
			return
		case 0x9:
			if err := item.writeFrame(0xA, payload); err != nil {
				return
			}
		case 0xA, 0x1, 0x2:
			// Browser messages are ignored. Keeping the frame parser active is
			// necessary for automatic Pong replies and disconnect detection.
		default:
			return
		}
	}
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for item := range m.clients {
		_ = item.writeFrame(0x8, []byte{0x03, 0xE9})
		_ = item.connection.Close()
	}
	m.clients = make(map[*client]struct{})
}

func (m *Manager) remove(item *client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.clients[item]; !exists {
		return
	}
	delete(m.clients, item)
	_ = item.connection.Close()
	m.broadcastLocked()
}

func (m *Manager) broadcastLocked() {
	payload, _ := json.Marshal(struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}{Type: "online_count", Count: len(m.clients)})
	for item := range m.clients {
		if err := item.writeFrame(0x1, payload); err != nil {
			_ = item.connection.Close()
			delete(m.clients, item)
		}
	}
}

func (c *client) writeFrame(opcode byte, payload []byte) error {
	if len(payload) > 125 {
		return errors.New("control or notification frame too large")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.connection.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := c.connection.Write(append([]byte{0x80 | opcode, byte(len(payload))}, payload...))
	return err
}

func readFrame(reader io.Reader) (byte, []byte, error) {
	head := make([]byte, 2)
	if _, err := io.ReadFull(reader, head); err != nil {
		return 0, nil, err
	}
	if head[0]&0x80 == 0 || head[1]&0x80 == 0 {
		return 0, nil, errors.New("fragmented or unmasked frame")
	}
	length := uint64(head[1] & 0x7F)
	if length == 126 {
		var value uint16
		if err := binary.Read(reader, binary.BigEndian, &value); err != nil {
			return 0, nil, err
		}
		length = uint64(value)
	}
	if length == 127 {
		if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
			return 0, nil, err
		}
	}
	if length > 1024 {
		return 0, nil, errors.New("frame too large")
	}
	mask := make([]byte, 4)
	if _, err := io.ReadFull(reader, mask); err != nil {
		return 0, nil, err
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return 0, nil, err
	}
	for i := range payload {
		payload[i] ^= mask[i%4]
	}
	return head[0] & 0x0F, payload, nil
}

func websocketAccept(key string) string {
	hash := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(hash[:])
}
func headerContains(value, wanted string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), wanted) {
			return true
		}
	}
	return false
}
