package miniappauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct{ client *redis.Client }

func NewRedisStore(client *redis.Client) *RedisStore { return &RedisStore{client: client} }
func storeKey(kind, token string) string {
	sum := sha256.Sum256([]byte(token))
	return "telegram-miniapp:" + kind + ":" + hex.EncodeToString(sum[:])
}
func (s *RedisStore) ReserveLaunch(ctx context.Context, fingerprint string) (bool, error) {
	reserved, err := s.client.SetNX(ctx, storeKey("used", fingerprint), "1", 6*time.Minute).Result()
	if err != nil {
		return false, ErrUnavailable
	}
	return reserved, nil
}
func (s *RedisStore) Put(ctx context.Context, kind, token string, grant Grant) error {
	body, err := json.Marshal(grant)
	if err != nil {
		return ErrUnavailable
	}
	stored, err := s.client.SetNX(ctx, storeKey(kind, token), body, 2*time.Minute).Result()
	if err != nil || !stored {
		return ErrUnavailable
	}
	return nil
}
func readGrant(value string, err error) (*Grant, error) {
	if errors.Is(err, redis.Nil) {
		return nil, ErrExpired
	}
	if err != nil || len(value) > 4096 {
		return nil, ErrUnavailable
	}
	var grant Grant
	if json.Unmarshal([]byte(value), &grant) != nil {
		return nil, ErrUnavailable
	}
	return &grant, nil
}
func (s *RedisStore) Read(ctx context.Context, kind, token string) (*Grant, error) {
	value, err := s.client.Get(ctx, storeKey(kind, token)).Result()
	return readGrant(value, err)
}
func (s *RedisStore) Consume(ctx context.Context, kind, token string) (*Grant, error) {
	// GETDEL is atomic in Redis/Dragonfly: concurrent workers cannot redeem the same ticket.
	value, err := s.client.GetDel(ctx, storeKey(kind, token)).Result()
	return readGrant(value, err)
}
