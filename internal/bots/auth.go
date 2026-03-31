package bots

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type authResponse struct {
	APIKey string `json:"api_key"`
	User   struct {
		ID int64 `json:"id"`
	} `json:"user"`
}

type AuthResult struct {
	APIKey string
	UserID int64
}

func DefaultAPIKeyFile(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return ""
	}
	dir := filepath.Join(os.TempDir(), "paper-street")
	return filepath.Join(dir, fmt.Sprintf("%s.key", role))
}

func ResolveAuth(baseURL, apiKey, role, adminPassword, apiKeyFile string, timeout time.Duration) (AuthResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey != "" {
		return AuthResult{APIKey: apiKey}, nil
	}
	if apiKeyFile == "" {
		apiKeyFile = DefaultAPIKeyFile(role)
	}
	if apiKeyFile != "" {
		if key, err := readAPIKeyFile(apiKeyFile); err == nil && key != "" {
			return AuthResult{APIKey: key}, nil
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return AuthResult{}, err
		}
	}
	role = strings.TrimSpace(role)
	if role == "" {
		return AuthResult{}, errors.New("BOT_ROLE is required when API_KEY is not set")
	}
	adminPassword = strings.TrimSpace(adminPassword)
	if adminPassword == "" {
		return AuthResult{}, errors.New("ADMIN_PASSWORD is required when API_KEY is not set")
	}
	key, userID, err := requestAPIKey(baseURL, role, adminPassword, timeout)
	if err != nil {
		return AuthResult{}, err
	}
	if apiKeyFile != "" {
		if err := writeAPIKeyFile(apiKeyFile, key); err != nil {
			return AuthResult{}, err
		}
	}
	return AuthResult{APIKey: key, UserID: userID}, nil
}

func requestAPIKey(baseURL, role, adminPassword string, timeout time.Duration) (string, int64, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "", 0, errors.New("API_BASE_URL is required")
	}
	payload, err := json.Marshal(map[string]string{
		"role":           role,
		"admin_password": adminPassword,
	})
	if err != nil {
		return "", 0, err
	}
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/auth/bot", bytes.NewReader(payload))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("auth request failed: status %d", resp.StatusCode)
	}
	var decoded authResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", 0, err
	}
	if strings.TrimSpace(decoded.APIKey) == "" {
		return "", 0, errors.New("auth response missing api_key")
	}
	return strings.TrimSpace(decoded.APIKey), decoded.User.ID, nil
}

func readAPIKeyFile(path string) (string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}

func writeAPIKeyFile(path, key string) error {
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(key)), 0o600)
}
