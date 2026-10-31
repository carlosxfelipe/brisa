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

func TestCheckSecurityToken(t *testing.T) {
	app := &App{
		token: "secret-token-123",
	}

	// 1. Valid token in header
	req, _ := http.NewRequest(http.MethodGet, "/brisa/events", nil)
	req.Header.Set("X-Brisa-Token", "secret-token-123")
	if !app.checkSecurityToken(req) {
		t.Fatal("expected valid token in header to pass")
	}

	// 2. Valid token in query param
	req, _ = http.NewRequest(http.MethodGet, "/brisa/events?token=secret-token-123", nil)
	if !app.checkSecurityToken(req) {
		t.Fatal("expected valid token in query to pass")
	}

	// 3. Invalid token
	req, _ = http.NewRequest(http.MethodGet, "/brisa/events?token=wrong-token", nil)
	if app.checkSecurityToken(req) {
		t.Fatal("expected wrong token to fail")
	}

	// 4. Missing token
	req, _ = http.NewRequest(http.MethodGet, "/brisa/events", nil)
	if app.checkSecurityToken(req) {
		t.Fatal("expected missing token to fail")
	}

	// 5. Both empty tokens should not authenticate
	emptyApp := &App{token: ""}
	req, _ = http.NewRequest(http.MethodGet, "/brisa/events", nil)
	if emptyApp.checkSecurityToken(req) {
		t.Fatal("expected empty token to fail when app token is empty")
	}

	// 6. DisableCSRFToken bypass
	disabledApp := &App{
		config: Config{DisableCSRFToken: true},
	}
	req, _ = http.NewRequest(http.MethodGet, "/brisa/events", nil)
	if !disabledApp.checkSecurityToken(req) {
		t.Fatal("expected DisableCSRFToken to allow request")
	}
}

func TestRejectsForeignHost(t *testing.T) {
	app := New(Config{AppID: "com.test.host", Store: StoreOptions{MemoryOnly: true}})

	// 1. Foreign domain with port should be rejected (DNS rebinding attempt)
	r := httptest.NewRequest(http.MethodGet, "/brisa/sdk.js", nil)
	r.Host = "evil.com:8080"
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for foreign host, got %d", w.Code)
	}

	// 2. Empty Host header should be rejected
	r = httptest.NewRequest(http.MethodGet, "/brisa/sdk.js", nil)
	r.Host = ""
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for empty host, got %d", w.Code)
	}

	// 3. IPv4 loopback with port should succeed
	r = httptest.NewRequest(http.MethodGet, "/brisa/sdk.js", nil)
	r.Host = "127.0.0.1:54321"
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for IPv4 loopback, got %d", w.Code)
	}

	// 4. IPv6 loopback with port should succeed
	r = httptest.NewRequest(http.MethodGet, "/brisa/sdk.js", nil)
	r.Host = "[::1]:54321"
	w = httptest.NewRecorder()
	app.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for IPv6 loopback, got %d", w.Code)
	}

	// 5. Explicitly configured Addr host should succeed
	customApp := New(Config{
		AppID: "com.test.customhost",
		Addr:  "192.168.1.50:8080",
		Store: StoreOptions{MemoryOnly: true},
	})
	r = httptest.NewRequest(http.MethodGet, "/brisa/sdk.js", nil)
	r.Host = "192.168.1.50:8080"
	w = httptest.NewRecorder()
	customApp.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for configured Addr host, got %d", w.Code)
	}
}
