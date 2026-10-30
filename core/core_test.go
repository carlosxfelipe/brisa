package core

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCDPWebSocketClient(t *testing.T) {
	// Spin up a minimal RFC 6455 WebSocket test server to verify our zero-dependency CDP client
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("webserver doesn't support hijacking")
		}
		key := r.Header.Get("Sec-WebSocket-Key")
		h := sha1.New()
		h.Write([]byte(key + wsGUID))
		accept := base64.StdEncoding.EncodeToString(h.Sum(nil))

		conn, bufrw, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()

		resp := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
		_, _ = bufrw.WriteString(resp)
		_ = bufrw.Flush()

		// Read one masked frame from client and reply with a CDP response frame
		clientSide := &cdpClient{conn: conn, reader: bufrw.Reader}
		_, payload, err := clientSide.readFrame()
		if err != nil {
			return
		}
		var req cdpRequest
		_ = json.Unmarshal(payload, &req)

		replyBytes, _ := json.Marshal(map[string]any{
			"id":     req.ID,
			"result": map[string]any{"windowId": 77},
		})
		// Server-to-client frame (unmasked)
		frame := append([]byte{0x81, byte(len(replyBytes))}, replyBytes...)
		_, _ = conn.Write(frame)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, err := dialCDP(wsURL, 2*time.Second)
	if err != nil {
		t.Fatalf("dialCDP failed: %v", err)
	}
	defer client.close()

	raw, err := client.call("Browser.getWindowForTarget", map[string]any{"targetId": "t1"}, "")
	if err != nil {
		t.Fatalf("cdp call failed: %v", err)
	}
	if !strings.Contains(string(raw), `"windowId":77`) {
		t.Fatalf("unexpected cdp result: %s", string(raw))
	}
}
