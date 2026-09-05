package quickimport

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTicketExpiryAndAtomicConsumption(t *testing.T) {
	r := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: r.Addr()})
	t.Cleanup(func() { client.Close() })
	store := NewTicketStore(client)
	ctx := context.Background()
	code, err := store.Issue(ctx, Ticket{UserID: 1, KeyID: 2, Agent: "opencode"})
	if err != nil {
		t.Fatal(err)
	}
	var success atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticket, e := store.Consume(ctx, code)
			if e == nil {
				if ticket.KeyID != 2 {
					t.Error("wrong key")
				}
				success.Add(1)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 {
		t.Fatalf("consumed %d times", success.Load())
	}
	code, err = store.Issue(ctx, Ticket{UserID: 1, KeyID: 2, Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	r.FastForward(6 * time.Minute)
	if _, err = store.Consume(ctx, code); err == nil {
		t.Fatal("expired ticket accepted")
	}
}

func TestConfigTargets(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "opencode"} {
		c, err := BuildConfig(agent, "deepseek", false, "https://example.com/v1", "mock-key", "")
		if err != nil || c.Agent != agent || c.Model != "deepseek-v4-pro" {
			t.Fatalf("%s: %#v %v", agent, c, err)
		}
	}
	if _, err := BuildConfig("claude", "openai", false, "https://example.com", "mock-key", ""); err == nil {
		t.Fatal("unsupported messages dispatch accepted")
	}
	if _, err := BuildConfig("chatgpt", "openai", false, "https://example.com", "mock-key", ""); err == nil {
		t.Fatal("unknown agent accepted")
	}
	c, err := BuildConfig("opencode", "gemini", false, "https://example.com/v1", "mock-key", "")
	if err != nil || c.BaseURL != "https://example.com/v1beta" {
		t.Fatalf("bad gemini endpoint: %#v %v", c, err)
	}
}
