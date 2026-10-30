package core

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// cdpClient is a zero-dependency RFC 6455 WebSocket client for Chrome DevTools Protocol (CDP).
type cdpClient struct {
	conn    net.Conn
	reader  *bufio.Reader
	writeMu sync.Mutex
	seq     atomic.Int64
	pending sync.Map // map[int64]chan cdpResponse
	closed  atomic.Bool
	done    chan struct{}
}

type cdpRequest struct {
	ID        int64          `json:"id"`
	Method    string         `json:"method"`
	Params    map[string]any `json:"params,omitempty"`
	SessionID string         `json:"sessionId,omitempty"`
}

type cdpResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

func dialCDP(wsURL string, timeout time.Duration) (*cdpClient, error) {
	u, err := url.Parse(wsURL)
	if err != nil {
		return nil, err
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":80"
	}
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return nil, err
	}

	keyBytes := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, keyBytes)
	wsKey := base64.StdEncoding.EncodeToString(keyBytes)

	path := u.RequestURI()
	if path == "" {
		path = "/"
	}

	req := fmt.Sprintf(
		"GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		path, u.Host, wsKey,
	)
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if _, err := conn.Write([]byte(req)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	br := bufio.NewReaderSize(conn, 64*1024)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if !strings.Contains(statusLine, "101") {
		_ = conn.Close()
		return nil, fmt.Errorf("cdp websocket handshake failed: %s", strings.TrimSpace(statusLine))
	}

	var acceptHeader string
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if k, v, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(k), "Sec-WebSocket-Accept") {
			acceptHeader = strings.TrimSpace(v)
		}
	}

	h := sha1.New()
	h.Write([]byte(wsKey + wsGUID))
	expectedAccept := base64.StdEncoding.EncodeToString(h.Sum(nil))
	if acceptHeader != expectedAccept {
		_ = conn.Close()
		return nil, errors.New("cdp websocket invalid Sec-WebSocket-Accept")
	}

	_ = conn.SetDeadline(time.Time{})

	c := &cdpClient{
		conn:   conn,
		reader: br,
		done:   make(chan struct{}),
	}
	go c.readLoop()
	return c, nil
}

func (c *cdpClient) call(method string, params map[string]any, sessionID string) (json.RawMessage, error) {
	if c.closed.Load() {
		return nil, errors.New("cdp connection is closed")
	}
	id := c.seq.Add(1)
	ch := make(chan cdpResponse, 1)
	c.pending.Store(id, ch)
	defer c.pending.Delete(id)

	payload, err := json.Marshal(cdpRequest{
		ID:        id,
		Method:    method,
		Params:    params,
		SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}

	if err := c.writeFrame(0x1, payload); err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("cdp error (%d): %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	case <-c.done:
		return nil, errors.New("cdp connection closed")
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("cdp timeout calling %s", method)
	}
}

func (c *cdpClient) writeFrame(opcode byte, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	var header []byte
	header = append(header, 0x80|opcode) // FIN + opcode

	n := len(payload)
	// Client-to-server frames MUST be masked (bit 0x80 on second byte)
	if n < 126 {
		header = append(header, 0x80|byte(n))
	} else if n <= 65535 {
		header = append(header, 0x80|126, byte(n>>8), byte(n))
	} else {
		header = append(header, 0x80|127)
		var lenBytes [8]byte
		binary.BigEndian.PutUint64(lenBytes[:], uint64(n))
		header = append(header, lenBytes[:]...)
	}

	maskKey := make([]byte, 4)
	_, _ = io.ReadFull(rand.Reader, maskKey)
	header = append(header, maskKey...)

	masked := make([]byte, n)
	for i := 0; i < n; i++ {
		masked[i] = payload[i] ^ maskKey[i%4]
	}

	if _, err := c.conn.Write(header); err != nil {
		return err
	}
	_, err := c.conn.Write(masked)
	return err
}

func (c *cdpClient) readLoop() {
	defer func() {
		c.closed.Store(true)
		close(c.done)
		_ = c.conn.Close()
	}()

	for {
		opcode, payload, err := c.readFrame()
		if err != nil {
			return
		}
		switch opcode {
		case 0x8: // Close
			return
		case 0x9: // Ping
			_ = c.writeFrame(0xA, payload)
		case 0x1, 0x2: // Text or Binary
			var msg cdpResponse
			if err := json.Unmarshal(payload, &msg); err == nil && msg.ID != 0 {
				if v, ok := c.pending.Load(msg.ID); ok {
					select {
					case v.(chan cdpResponse) <- msg:
					default:
					}
				}
			}
		}
	}
}

func (c *cdpClient) readFrame() (byte, []byte, error) {
	var head [2]byte
	if _, err := io.ReadFull(c.reader, head[:]); err != nil {
		return 0, nil, err
	}
	opcode := head[0] & 0x0F
	masked := (head[1] & 0x80) != 0
	length := uint64(head[1] & 0x7F)

	if length == 126 {
		var ext [2]byte
		if _, err := io.ReadFull(c.reader, ext[:]); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	} else if length == 127 {
		var ext [8]byte
		if _, err := io.ReadFull(c.reader, ext[:]); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext[:])
	}

	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(c.reader, maskKey[:]); err != nil {
			return 0, nil, err
		}
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}
	return opcode, payload, nil
}

func (c *cdpClient) close() {
	if c.closed.Swap(true) {
		return
	}
	_ = c.writeFrame(0x8, nil)
	_ = c.conn.Close()
}
