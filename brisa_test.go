package brisa_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/carlosxfelipe/brisa"
)

type CreateNoteReq struct {
	Title string `json:"title" doc:"Note title" example:"Buy coffee" validate:"required,min=3,max=50"`
	Tag   string `json:"tag" doc:"Category" example:"work" validate:"required,oneof=work,personal,urgent"`
}

type NoteRes struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Tag   string `json:"tag"`
}

type ArchiveNoteEndpoint struct{}

func (e *ArchiveNoteEndpoint) Handle(ctx *brisa.Context, req struct {
	ID string `json:"id" validate:"required"`
}) (brisa.Map, error) {
	return brisa.Map{"archived": req.ID}, nil
}

type MathService struct{}

type AddReq struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddRes struct {
	Sum int `json:"sum"`
}

func (m *MathService) Add(ctx *brisa.Context, req AddReq) (AddRes, error) {
	return AddRes{Sum: req.A + req.B}, nil
}

func (m *MathService) Ping() (string, error) {
	return "pong", nil
}

func TestREPRBindAndDispatch(t *testing.T) {
	app := brisa.New(brisa.Config{
		AppID: "com.test.notes",
		Title: "Notes Desktop",
		Store: brisa.StoreOptions{MemoryOnly: true},
	})

	brisa.Bind(app, "notes.create", func(ctx *brisa.Context, req CreateNoteReq) (NoteRes, error) {
		return NoteRes{
			ID:    "note_1",
			Title: req.Title,
			Tag:   req.Tag,
		}, nil
	}).Summary("Create a note").Tags("Notes")

	brisa.BindEndpoint(app, "notes.archive", &ArchiveNoteEndpoint{})

	// 1. Valid dispatch
	res, err := app.Dispatch(context.Background(), "notes.create", CreateNoteReq{
		Title: "Finish Brisa",
		Tag:   "work",
	})
	if err != nil {
		t.Fatalf("unexpected dispatch error: %v", err)
	}
	if res.Status != 200 || res.Error != nil {
		t.Fatalf("expected status 200, got %d (%+v)", res.Status, res.Error)
	}
	note, ok := res.Data.(NoteRes)
	if !ok || note.ID != "note_1" || note.Title != "Finish Brisa" {
		t.Fatalf("unexpected response data: %#v", res.Data)
	}

	// 2. Validation error (title too short + invalid oneof tag)
	badRes, err := app.Dispatch(context.Background(), "notes.create", CreateNoteReq{
		Title: "Hi",
		Tag:   "invalid",
	})
	if err != nil {
		t.Fatalf("unexpected dispatch error: %v", err)
	}
	if badRes.Status != 422 || badRes.Error == nil {
		t.Fatalf("expected 422 validation error, got %+v", badRes)
	}
	if len(badRes.Error.Errors["title"]) == 0 || len(badRes.Error.Errors["tag"]) == 0 {
		t.Fatalf("expected field errors for title and tag, got %+v", badRes.Error.Errors)
	}

	// 3. Struct Endpoint dispatch
	archRes, err := app.Dispatch(context.Background(), "notes.archive", map[string]any{"id": "note_1"})
	if err != nil || archRes.Status != 200 {
		t.Fatalf("expected archive endpoint 200, got %+v (%v)", archRes, err)
	}
}

func TestBindServiceAndGroups(t *testing.T) {
	app := brisa.New(brisa.Config{
		Store: brisa.StoreOptions{MemoryOnly: true},
	})

	brisa.BindService(app, "math", &MathService{})

	res, err := app.Dispatch(context.Background(), "math.add", AddReq{A: 15, B: 27})
	if err != nil || res.Status != 200 {
		t.Fatalf("expected math.add 200, got %+v (%v)", res, err)
	}
	addOut := res.Data.(AddRes)
	if addOut.Sum != 42 {
		t.Fatalf("expected sum=42, got %d", addOut.Sum)
	}

	pingRes, err := app.Dispatch(context.Background(), "math.ping", nil)
	if err != nil || pingRes.Status != 200 || pingRes.Data != "pong" {
		t.Fatalf("expected pong, got %+v", pingRes)
	}
}

func TestHTTPRPCBridgeAndSecurityToken(t *testing.T) {
	fsMap := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<!DOCTYPE html><html><head><title>App</title></head><body><h1>Hello Brisa</h1></body></html>`),
		},
	}

	app := brisa.New(brisa.Config{
		Title:  "Bridge Test",
		Assets: fsMap,
		Store:  brisa.StoreOptions{MemoryOnly: true},
	})

	brisa.Bind(app, "greet", func(ctx *brisa.Context, req struct {
		Name string `json:"name" validate:"required"`
	}) (brisa.Map, error) {
		return brisa.Map{"message": "Hello, " + req.Name}, nil
	}).Summary("Greet user")

	// 1. Index HTML should inject /brisa/sdk.js automatically
	reqIndex := httptest.NewRequest(http.MethodGet, "/", nil)
	recIndex := httptest.NewRecorder()
	app.ServeHTTP(recIndex, reqIndex)
	if recIndex.Code != http.StatusOK || !strings.Contains(recIndex.Body.String(), `/brisa/sdk.js`) {
		t.Fatalf("expected index.html with injected sdk.js, got status=%d body=%s", recIndex.Code, recIndex.Body.String())
	}

	// 2. SDK JS, Adwaita CSS, Schema JSON, TypeScript SDK, and Inspector HTML
	for _, endpoint := range []string{"/brisa/adwaita.css", "/brisa/sdk.js", "/brisa/schema.json", "/brisa/types.d.ts", "/brisa/inspector"} {
		r := httptest.NewRequest(http.MethodGet, endpoint, nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		if w.Code != http.StatusOK || w.Body.Len() == 0 {
			t.Fatalf("expected 200 OK for %s, got %d", endpoint, w.Code)
		}
	}

	// 3. Calling /brisa/rpc WITHOUT X-Brisa-Token must return 403 Forbidden
	body, _ := json.Marshal(map[string]any{
		"command": "greet",
		"payload": map[string]any{"name": "Carlos"},
	})
	unauthReq := httptest.NewRequest(http.MethodPost, "/brisa/rpc", bytes.NewReader(body))
	unauthRec := httptest.NewRecorder()
	app.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden without token, got %d", unauthRec.Code)
	}

	// 4. Calling /brisa/rpc WITH valid X-Brisa-Token succeeds
	authReq := httptest.NewRequest(http.MethodPost, "/brisa/rpc", bytes.NewReader(body))
	authReq.Header.Set("X-Brisa-Token", app.Token())
	authRec := httptest.NewRecorder()
	app.ServeHTTP(authRec, authReq)
	if authRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with token, got %d: %s", authRec.Code, authRec.Body.String())
	}

	// 5. Batch RPC endpoint
	batchBody, _ := json.Marshal([]map[string]any{
		{"command": "greet", "payload": map[string]any{"name": "Ana"}},
		{"command": "greet", "payload": map[string]any{"name": ""}}, // validation failure
	})
	batchReq := httptest.NewRequest(http.MethodPost, "/brisa/rpc/batch", bytes.NewReader(batchBody))
	batchReq.Header.Set("X-Brisa-Token", app.Token())
	batchRec := httptest.NewRecorder()
	app.ServeHTTP(batchRec, batchReq)
	if batchRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for batch, got %d", batchRec.Code)
	}
	var batchResults []brisa.DispatchResult
	if err := json.Unmarshal(batchRec.Body.Bytes(), &batchResults); err != nil || len(batchResults) != 2 {
		t.Fatalf("unexpected batch response: %v", err)
	}
	if batchResults[0].Status != 200 || batchResults[1].Status != 422 {
		t.Fatalf("expected statuses [200, 422], got [%d, %d]", batchResults[0].Status, batchResults[1].Status)
	}
}

func TestEncryptedStorePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "encrypted_store.json")

	s1, err := brisa.New(brisa.Config{
		AppID: "com.test.enc",
		Store: brisa.StoreOptions{
			Path:          storePath,
			EncryptionKey: "super-secret-desktop-key",
		},
	}).Store(), error(nil)
	if err != nil {
		t.Fatalf("failed to create encrypted store: %v", err)
	}

	if err := s1.Set("theme", "dark"); err != nil {
		t.Fatalf("failed to set key: %v", err)
	}
	if err := s1.Set("windowWidth", 1280); err != nil {
		t.Fatalf("failed to set int key: %v", err)
	}

	// Re-open with the same encryption key
	s2, err := brisa.New(brisa.Config{
		AppID: "com.test.enc",
		Store: brisa.StoreOptions{
			Path:          storePath,
			EncryptionKey: "super-secret-desktop-key",
		},
	}).Store(), error(nil)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}

	if got := s2.GetString("theme"); got != "dark" {
		t.Fatalf("expected 'dark', got %q", got)
	}
	if got := s2.GetInt("windowWidth"); got != 1280 {
		t.Fatalf("expected 1280, got %d", got)
	}
}

func TestMiddlewaresAndEvents(t *testing.T) {
	app := brisa.New(brisa.Config{
		Store: brisa.StoreOptions{MemoryOnly: true},
	})
	app.Use(brisa.Recover(), brisa.RequestID(), brisa.Timeout(50*time.Millisecond))

	brisa.Bind(app, "panic.cmd", func(ctx *brisa.Context, _ brisa.Empty) (brisa.Empty, error) {
		panic("boom!")
	})

	brisa.Bind(app, "slow.cmd", func(ctx *brisa.Context, _ brisa.Empty) (brisa.Empty, error) {
		select {
		case <-time.After(200 * time.Millisecond):
		case <-ctx.Context().Done():
		}
		return brisa.Empty{}, nil
	})

	brisa.Bind(app, "custom.err", func(ctx *brisa.Context, _ brisa.Empty) (brisa.Empty, error) {
		return brisa.Empty{}, brisa.NewError(409, "CONFLICT_STATE", "resource already locked")
	})

	// 1. Panic recovery
	panicRes, _ := app.Dispatch(context.Background(), "panic.cmd", nil)
	if panicRes.Status != 500 || panicRes.Error == nil || panicRes.Error.Code != "INTERNAL_PANIC" {
		t.Fatalf("expected 500 INTERNAL_PANIC, got %+v", panicRes)
	}

	// 2. Timeout middleware
	slowRes, _ := app.Dispatch(context.Background(), "slow.cmd", nil)
	if slowRes.Status != 408 || slowRes.Error == nil || slowRes.Error.Code != "COMMAND_TIMEOUT" {
		t.Fatalf("expected 408 COMMAND_TIMEOUT, got %+v", slowRes)
	}

	// 3. Structured RPCError
	errRes, _ := app.Dispatch(context.Background(), "custom.err", nil)
	if errRes.Status != 409 || errRes.Error == nil || errRes.Error.Code != "CONFLICT_STATE" {
		t.Fatalf("expected 409 CONFLICT_STATE, got %+v", errRes)
	}

	// 4. Event Bus
	received := make(chan string, 1)
	app.On("user:login", func(data any) {
		if s, ok := data.(string); ok {
			received <- s
		}
	})
	app.Emit("user:login", "carlos")

	select {
	case user := <-received:
		if user != "carlos" {
			t.Fatalf("expected carlos, got %s", user)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestBuiltinSystemCommands(t *testing.T) {
	app := brisa.New(brisa.Config{
		Store: brisa.StoreOptions{MemoryOnly: true},
	})

	// Store commands
	_, _ = app.Dispatch(context.Background(), "$brisa.store.set", map[string]any{"key": "lang", "value": "pt-BR"})
	getRes, _ := app.Dispatch(context.Background(), "$brisa.store.get", map[string]any{"key": "lang"})
	if getRes.Status != 200 {
		t.Fatalf("expected 200 from $brisa.store.get, got %d", getRes.Status)
	}
	m := getRes.Data.(brisa.Map)
	if m["value"] != "pt-BR" || m["exists"] != true {
		t.Fatalf("unexpected store value: %+v", m)
	}

	// Window commands
	_, _ = app.Dispatch(context.Background(), "$brisa.window.setTitle", map[string]any{"title": "Updated Title"})
	if app.Window().Title() != "Updated Title" {
		t.Fatalf("expected window title 'Updated Title', got %q", app.Window().Title())
	}

	// Clipboard fallback
	_, _ = app.Dispatch(context.Background(), "$brisa.clipboard.writeText", map[string]any{"text": "brisa-clip"})
	clipRes, _ := app.Dispatch(context.Background(), "$brisa.clipboard.readText", nil)
	if clipRes.Status != 200 {
		t.Fatalf("expected 200 from clipboard read, got %d", clipRes.Status)
	}
}

func TestFailFastOnInvalidValidateTag(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on invalid validate tag")
		}
	}()

	app := brisa.New(brisa.Config{Store: brisa.StoreOptions{MemoryOnly: true}})
	brisa.Bind(app, "bad.tag", func(ctx *brisa.Context, req struct {
		Name string `json:"name" validate:"nonexistent_rule"`
	}) (brisa.Empty, error) {
		return brisa.Empty{}, errors.New("unreachable")
	})
}
