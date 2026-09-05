// Package quickimport provides credential-free, short-lived import tickets.
package quickimport

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/redis/go-redis/v9"
	"time"
)

const TicketTTL = 5 * time.Minute

var ErrTicket = errors.New("configuration code expired or already used; generate a new command")

type Ticket struct {
	UserID  int64  `json:"user_id"`
	KeyID   int64  `json:"key_id"`
	Agent   string `json:"agent"`
	Model   string `json:"model"`
	BaseURL string `json:"base_url"`
}
type TicketStore struct{ client *redis.Client }

func NewTicketStore(client *redis.Client) *TicketStore { return &TicketStore{client: client} }
func ticketKey(code string) string {
	sum := sha256.Sum256([]byte(code))
	return "quick-import:v1:" + hex.EncodeToString(sum[:])
}
func (s *TicketStore) Issue(ctx context.Context, ticket Ticket) (string, error) {
	if s == nil || s.client == nil {
		return "", ErrTicket
	}
	// Do not retain API credentials in Redis. Scope is checked again on redemption.
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	code := hex.EncodeToString(raw)
	data, err := json.Marshal(ticket)
	if err != nil {
		return "", err
	}
	err = s.client.Set(ctx, ticketKey(code), data, TicketTTL).Err()
	return code, err
}
func (s *TicketStore) Consume(ctx context.Context, code string) (Ticket, error) {
	var ticket Ticket
	if s == nil || s.client == nil || len(code) != 64 {
		return ticket, ErrTicket
	}
	if _, err := hex.DecodeString(code); err != nil {
		return ticket, ErrTicket
	}
	data, err := s.client.GetDel(ctx, ticketKey(code)).Bytes()
	if err != nil {
		return ticket, ErrTicket
	}
	if json.Unmarshal(data, &ticket) != nil {
		return Ticket{}, ErrTicket
	}
	return ticket, nil
}
