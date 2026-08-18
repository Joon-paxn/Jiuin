package online

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	webSocketGUID   = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	maxFramePayload = 1 << 20
	readDeadline    = 70 * time.Second
	pingInterval    = 25 * time.Second
	writeDeadline   = 5 * time.Second
)

type message struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type client struct {
	connection net.Conn
	buffer     *bufio.ReadWriter
	writeMu    sync.Mutex
	done       chan struct{}
	closeOnce  sync.Once
}

func (client *client) close() {
	client.closeOnce.Do(func() {
		close(client.done)
		_ = client.connection.Close()
	})
}

func (client *client) shutdown() {
	_ = client.writeFrame(0x8, []byte{0x03, 0xE9}) // 1001: going away
	client.close()
}

// Manager tracks browser WebSocket connections and broadcasts the current
// count. It implements the small RFC 6455 server surface needed by browsers
// without adding a runtime dependency to the backend.
type Manager struct {
	mu      sync.Mutex
	clients map[*client]struct{}
	closed  bool
	origins map[string]struct{}
	logger  *slog.Logger
}

func NewManager(allowedOrigins []string, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origins[origin] = struct{}{}
	}
	return &Manager{clients: make(map[*client]struct{}), origins: origins, logger: logger}
}

func (manager *Manager) Count() int {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return len(manager.clients)
}

// Close disconnects every active browser connection before HTTP shutdown.
func (manager *Manager) Close() {
	manager.mu.Lock()
	if manager.closed {
		manager.mu.Unlock()
		return
	}
	manager.closed = true
	clients := make([]*client, 0, len(manager.clients))
	for connectedClient := range manager.clients {
		clients = append(clients, connectedClient)
	}
	manager.clients = make(map[*client]struct{})
	manager.mu.Unlock()
	for _, connectedClient := range clients {
		connectedClient.shutdown()
	}
}

func (manager *Manager) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(responseWriter, "WebSocket endpoint requires GET", http.StatusMethodNotAllowed)
		return
	}
	if !isWebSocketUpgrade(request) {
		http.Error(responseWriter, "WebSocket upgrade required", http.StatusBadRequest)
		return
	}
	if !manager.originAllowed(request.Header.Get("Origin")) {
		http.Error(responseWriter, "request origin is not allowed", http.StatusForbidden)
		return
	}
	key := strings.TrimSpace(request.Header.Get("Sec-WebSocket-Key"))
	if key == "" || request.Header.Get("Sec-WebSocket-Version") != "13" {
		http.Error(responseWriter, "unsupported WebSocket handshake", http.StatusBadRequest)
		return
	}

	manager.mu.Lock()
	closed := manager.closed
	manager.mu.Unlock()
	if closed {
		http.Error(responseWriter, "WebSocket service is shutting down", http.StatusServiceUnavailable)
		return
	}

	hijacker, ok := responseWriter.(http.Hijacker)
	if !ok {
		http.Error(responseWriter, "WebSocket is unavailable", http.StatusInternalServerError)
		return
	}
	connection, buffer, err := hijacker.Hijack()
	if err != nil {
		manager.logger.Error("online websocket hijack failed", "error", err)
		return
	}
	accept := websocketAccept(key)
	if _, err := buffer.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\nCache-Control: no-store\r\n\r\n"); err != nil {
		_ = connection.Close()
		return
	}
	if err := buffer.Flush(); err != nil {
		_ = connection.Close()
		return
	}

	connectedClient := &client{connection: connection, buffer: buffer, done: make(chan struct{})}
	if !manager.add(connectedClient) {
		connectedClient.close()
		return
	}
	defer manager.remove(connectedClient)
	manager.logger.Info("online websocket connected", "online", manager.Count())
	manager.broadcast()

	go manager.pingLoop(connectedClient)
	manager.readLoop(connectedClient)
}

func (manager *Manager) add(connectedClient *client) bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.closed {
		return false
	}
	manager.clients[connectedClient] = struct{}{}
	return true
}

func (manager *Manager) remove(connectedClient *client) {
	manager.mu.Lock()
	_, existed := manager.clients[connectedClient]
	delete(manager.clients, connectedClient)
	manager.mu.Unlock()
	connectedClient.close()
	if existed {
		manager.logger.Info("online websocket disconnected", "online", manager.Count())
		manager.broadcast()
	}
}

func (manager *Manager) broadcast() {
	payload, err := json.Marshal(message{Type: "online_count", Count: manager.Count()})
	if err != nil {
		return
	}
	manager.mu.Lock()
	clients := make([]*client, 0, len(manager.clients))
	for connectedClient := range manager.clients {
		clients = append(clients, connectedClient)
	}
	manager.mu.Unlock()
	for _, connectedClient := range clients {
		if err := connectedClient.writeFrame(0x1, payload); err != nil {
			connectedClient.close()
		}
	}
}

func (manager *Manager) pingLoop(connectedClient *client) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := connectedClient.writeFrame(0x9, nil); err != nil {
				connectedClient.close()
				return
			}
		case <-connectedClient.done:
			return
		}
	}
}

func (manager *Manager) readLoop(connectedClient *client) {
	_ = connectedClient.connection.SetReadDeadline(time.Now().Add(readDeadline))
	for {
		opcode, payload, err := readFrame(connectedClient.buffer.Reader)
		if err != nil {
			return
		}
		switch opcode {
		case 0x8:
			_ = connectedClient.writeFrame(0x8, payload)
			return
		case 0x9:
			if err := connectedClient.writeFrame(0xA, payload); err != nil {
				return
			}
		case 0xA:
			_ = connectedClient.connection.SetReadDeadline(time.Now().Add(readDeadline))
		case 0x1, 0x2, 0x0:
			_ = connectedClient.connection.SetReadDeadline(time.Now().Add(readDeadline))
		default:
			return
		}
	}
}

func (client *client) writeFrame(opcode byte, payload []byte) error {
	if len(payload) > maxFramePayload {
		return errors.New("websocket payload is too large")
	}
	client.writeMu.Lock()
	defer client.writeMu.Unlock()
	if err := client.connection.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
		return err
	}
	frame := make([]byte, 0, len(payload)+10)
	frame = append(frame, 0x80|opcode)
	switch {
	case len(payload) < 126:
		frame = append(frame, byte(len(payload)))
	case len(payload) <= 65535:
		frame = append(frame, 126, byte(len(payload)>>8), byte(len(payload)))
	default:
		frame = append(frame, 127, 0, 0, 0, 0, byte(len(payload)>>24), byte(len(payload)>>16), byte(len(payload)>>8), byte(len(payload)))
	}
	frame = append(frame, payload...)
	_, err := client.buffer.Write(frame)
	if err != nil {
		return err
	}
	return client.buffer.Flush()
}

func readFrame(reader *bufio.Reader) (byte, []byte, error) {
	first, err := reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	second, err := reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	if first&0x70 != 0 || first&0x80 == 0 {
		return 0, nil, errors.New("websocket RSV bits are not supported")
	}
	masked := second&0x80 != 0
	length := uint64(second & 0x7f)
	if length == 126 {
		var extended [2]byte
		if _, err := io.ReadFull(reader, extended[:]); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(extended[:]))
	} else if length == 127 {
		var extended [8]byte
		if _, err := io.ReadFull(reader, extended[:]); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(extended[:])
	}
	if length > maxFramePayload || !masked {
		return 0, nil, errors.New("invalid websocket client frame")
	}
	var mask [4]byte
	if _, err := io.ReadFull(reader, mask[:]); err != nil {
		return 0, nil, err
	}
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(reader, payload); err != nil {
		return 0, nil, err
	}
	for index := range payload {
		payload[index] ^= mask[index%4]
	}
	return first & 0x0f, payload, nil
}

func isWebSocketUpgrade(request *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(request.Header.Get("Upgrade")), "websocket") && headerContainsToken(request.Header.Get("Connection"), "upgrade")
}

func headerContainsToken(value, wanted string) bool {
	for _, token := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(token), wanted) {
			return true
		}
	}
	return false
}

func (manager *Manager) originAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	_, ok := manager.origins[origin]
	return ok
}

func websocketAccept(key string) string {
	hash := sha1.Sum([]byte(key + webSocketGUID))
	return base64.StdEncoding.EncodeToString(hash[:])
}

var _ http.Handler = (*Manager)(nil)
