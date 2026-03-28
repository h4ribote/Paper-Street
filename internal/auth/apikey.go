package auth

import (
	"encoding/hex"
	"errors"
	"strings"
	"sync"
)

const (
	APIKeyByteLength = 10
	APIKeyHexLength  = APIKeyByteLength * 2
)

var ErrInvalidAPIKeyLength = errors.New("api key must be 20 hex characters")

type APIKey [APIKeyByteLength]byte

func ParseAPIKeyHex(value string) (APIKey, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) != APIKeyHexLength {
		return APIKey{}, ErrInvalidAPIKeyLength
	}
	decoded, err := hex.DecodeString(trimmed)
	if err != nil {
		return APIKey{}, err
	}
	var key APIKey
	copy(key[:], decoded)
	return key, nil
}

func (k APIKey) String() string {
	return hex.EncodeToString(k[:])
}

type APIKeyCache struct {
	mu   sync.RWMutex
	keys map[APIKey]struct{}
}

func NewAPIKeyCache() *APIKeyCache {
	return &APIKeyCache{
		keys: make(map[APIKey]struct{}),
	}
}

func (c *APIKeyCache) Add(key APIKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys[key] = struct{}{}
}

func (c *APIKeyCache) AddHex(value string) error {
	key, err := ParseAPIKeyHex(value)
	if err != nil {
		return err
	}
	c.Add(key)
	return nil
}

func (c *APIKeyCache) Remove(key APIKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.keys, key)
}

func (c *APIKeyCache) RemoveHex(value string) error {
	key, err := ParseAPIKeyHex(value)
	if err != nil {
		return err
	}
	c.Remove(key)
	return nil
}

func (c *APIKeyCache) Contains(key APIKey) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.keys[key]
	return ok
}

func (c *APIKeyCache) ContainsHex(value string) bool {
	key, err := ParseAPIKeyHex(value)
	if err != nil {
		return false
	}
	return c.Contains(key)
}
