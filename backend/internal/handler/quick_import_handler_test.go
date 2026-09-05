package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/quickimport"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type quickKeysFake struct{ key *service.APIKey }

func (f *quickKeysFake) GetByID(context.Context, int64) (*service.APIKey, error) { return f.key, nil }

type quickSettingsFake struct{}

func (quickSettingsFake) GetPublicSettings(context.Context) (*service.PublicSettings, error) {
	return &service.PublicSettings{APIBaseURL: "https://gateway.example.com"}, nil
}
func TestQuickImportOwnershipRevocationAndExchange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	defer client.Close()
	keys := &quickKeysFake{key: &service.APIKey{ID: 2, UserID: 1, Key: "mock-private-key", Status: service.StatusActive, Group: &service.Group{Platform: "openai", Status: service.StatusActive}}}
	h := &QuickImportHandler{keys: keys, settings: quickSettingsFake{}, tickets: quickimport.NewTicketStore(client)}
	request := func(method gin.HandlerFunc, body string, user int64) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		if user > 0 {
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: user})
		}
		method(c)
		return w
	}
	if w := request(h.Issue, `{"key_id":2,"agent":"opencode"}`, 3); w.Code == 200 {
		t.Fatal("other user accepted")
	}
	if w := request(h.Issue, `{"key_id":2,"agent":"opencode"}`, 0); w.Code != 401 {
		t.Fatal("missing authentication accepted")
	}
	issue := func() string {
		w := request(h.Issue, `{"key_id":2,"agent":"opencode"}`, 1)
		if w.Code != 200 {
			t.Fatalf("issue: %d %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), keys.key.Key) {
			t.Fatal("key leaked during issue")
		}
		var body struct {
			Data struct {
				Ticket string `json:"ticket"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Data.Ticket
	}
	code := issue()
	body := `{"ticket":"` + code + `","agent":"opencode"}`
	w := request(h.Exchange, body, 0)
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Body.String(), "mock-private-key") {
		t.Fatalf("exchange failed %d", w.Code)
	}
	if w = request(h.Exchange, body, 0); w.Code == 200 {
		t.Fatal("replay accepted")
	}
	code = issue()
	keys.key.Status = service.StatusAPIKeyDisabled
	if w = request(h.Exchange, `{"ticket":"`+code+`","agent":"opencode"}`, 0); w.Code == 200 {
		t.Fatal("revoked key accepted")
	}
	keys.key.Status = service.StatusActive
	keys.key.Group.ClaudeCodeOnly = true
	if w = request(h.Issue, `{"key_id":2,"agent":"codex"}`, 1); w.Code == 200 {
		t.Fatal("Claude-only group accepted a Codex import")
	}
	keys.key.Group.ClaudeCodeOnly = false
	code = issue()
	if w = request(h.Exchange, `{"ticket":"`+code+`","agent":"claude"}`, 0); w.Code == 200 {
		t.Fatal("wrong agent accepted")
	}
}
