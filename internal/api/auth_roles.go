package api

import (
	"sort"

	"github.com/h4ribote/Paper-Street/internal/models"
)

func (s *MarketStore) APIKeys() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	keys := make([]string, 0, len(s.apiKeyToUser))
	for key := range s.apiKeyToUser {
		keys = append(keys, key)
	}
	s.mu.RUnlock()
	sort.Strings(keys)
	return keys
}

func (s *MarketStore) APIKeyForRole(role string) (string, models.User, bool) {
	if s == nil {
		return "", models.User{}, false
	}
	normalized := normalizeRole(role)
	if normalized == "" {
		return "", models.User{}, false
	}
	s.mu.RLock()
	userID, ok := s.roleToUserID[normalized]
	key := s.roleToAPIKey[normalized]
	user := s.users[userID]
	s.mu.RUnlock()
	if !ok || key == "" || userID == 0 {
		return "", models.User{}, false
	}
	if user.ID == 0 && s != nil {
		user, _ = s.User(userID)
	}
	return key, user, true
}
