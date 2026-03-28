package bots

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/engine"
)

type APIClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type OrderRequest struct {
	AssetID   int64  `json:"asset_id"`
	UserID    int64  `json:"user_id"`
	Side      string `json:"side"`
	Type      string `json:"type"`
	Quantity  int64  `json:"quantity"`
	Price     int64  `json:"price,omitempty"`
	StopPrice int64  `json:"stop_price,omitempty"`
}

type orderResponse struct {
	Order engine.Order `json:"order"`
}

type apiError struct {
	Error string `json:"error"`
}

func NewAPIClient(baseURL, apiKey string, timeout time.Duration) *APIClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &APIClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *APIClient) OrderBook(ctx context.Context, assetID int64, depth int) (engine.OrderBookSnapshot, error) {
	if c == nil {
		return engine.OrderBookSnapshot{}, fmt.Errorf("api client is nil")
	}
	url := fmt.Sprintf("%s/market/orderbook/%d", c.baseURL, assetID)
	if depth > 0 {
		url = fmt.Sprintf("%s?depth=%d", url, depth)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return engine.OrderBookSnapshot{}, err
	}
	c.addAuth(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return engine.OrderBookSnapshot{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return engine.OrderBookSnapshot{}, decodeAPIError(resp)
	}
	var snapshot engine.OrderBookSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return engine.OrderBookSnapshot{}, err
	}
	return snapshot, nil
}

func (c *APIClient) SubmitOrder(ctx context.Context, request OrderRequest) (*engine.Order, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/orders", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, decodeAPIError(resp)
	}
	var response orderResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response.Order, nil
}

func (c *APIClient) CancelOrder(ctx context.Context, orderID int64) error {
	if c == nil {
		return fmt.Errorf("api client is nil")
	}
	url := fmt.Sprintf("%s/orders/%d", c.baseURL, orderID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	c.addAuth(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return decodeAPIError(resp)
	}
	return nil
}

func (c *APIClient) addAuth(req *http.Request) {
	if c.apiKey == "" || req == nil {
		return
	}
	req.Header.Set("X-API-Key", c.apiKey)
}

func decodeAPIError(resp *http.Response) error {
	if resp == nil {
		return fmt.Errorf("api error: no response")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("api error: status %d", resp.StatusCode)
	}
	if len(body) == 0 {
		return fmt.Errorf("api error: status %d", resp.StatusCode)
	}
	var apiErr apiError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error != "" {
		return fmt.Errorf("api error: %s", apiErr.Error)
	}
	return fmt.Errorf("api error: status %d", resp.StatusCode)
}
