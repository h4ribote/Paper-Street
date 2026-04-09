package api

import (
	"time"

	"fmt"
	"sort"
	"strings"

	"github.com/h4ribote/Paper-Street/internal/models"
)

const discordRolePrefix = "discord:"

func discordRoleForUser(userID int64) string {
	if userID == 0 {
		return ""
	}
	return fmt.Sprintf("%s%d", discordRolePrefix, userID)
}

func isDiscordRole(role string) bool {
	return strings.HasPrefix(role, discordRolePrefix)
}

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
	if isDiscordRole(normalized) {
		return "", models.User{}, false
	}
	s.mu.RLock()
	userID, ok := s.roleToUserID[normalized]
	key := s.roleToAPIKey[normalized]
	s.mu.RUnlock()
	
	if !ok || key == "" || userID == 0 {
		s.mu.Lock()
		userID, ok = s.roleToUserID[normalized]
		if !ok || userID == 0 {
			// If there is no user mapped, we generate a user id since a bot is requesting it.
			userID = s.nextUserID
			s.nextUserID++
			s.roleToUserID[normalized] = userID
			user := models.User{ID: userID, Username: normalized, Role: "bot"}
			s.testUsers[userID] = user
			if s.queries != nil {
				ctx, cancel := s.dbContext()
				_ = s.queries.UpsertUser(ctx, user, time.Now().UTC())
				cancel()
			}
		}
		key = s.roleToAPIKey[normalized]
		if key == "" {
			newKey, err := generateAPIKeyHex()
			if err == nil {
				key = newKey
				s.roleToAPIKey[normalized] = key
				s.apiKeyToUser[key] = userID
				s.persistAPIKey(normalized, key, userID)
			}
		}
		s.mu.Unlock()
	}
	if key == "" || userID == 0 {
		return "", models.User{}, false
	}
	user, _ := s.User(userID)
	return key, user, true
}
