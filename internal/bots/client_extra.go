package bots

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/h4ribote/Paper-Street/internal/engine"
)

func (c *APIClient) Assets(ctx context.Context, assetType, sector string) ([]Asset, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	endpoint := fmt.Sprintf("%s/api/assets", c.baseURL)
	query := url.Values{}
	if assetType != "" {
		query.Set("type", assetType)
	}
	if sector != "" {
		query.Set("sector", sector)
	}
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}
	var assets []Asset
	if err := c.getJSON(ctx, endpoint, &assets); err != nil {
		return nil, err
	}
	return assets, nil
}

func (c *APIClient) Asset(ctx context.Context, assetID int64) (*Asset, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	if assetID <= 0 {
		return nil, fmt.Errorf("asset_id must be positive")
	}
	endpoint := fmt.Sprintf("%s/api/assets/%d", c.baseURL, assetID)
	var asset Asset
	if err := c.getJSON(ctx, endpoint, &asset); err != nil {
		return nil, err
	}
	return &asset, nil
}

func (c *APIClient) Candles(ctx context.Context, assetID int64, timeframe string, limit int) ([]Candle, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	if assetID <= 0 {
		return nil, fmt.Errorf("asset_id must be positive")
	}
	endpoint := fmt.Sprintf("%s/api/market/candles/%d", c.baseURL, assetID)
	query := url.Values{}
	if timeframe != "" {
		query.Set("timeframe", timeframe)
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}
	var candles []Candle
	if err := c.getJSON(ctx, endpoint, &candles); err != nil {
		return nil, err
	}
	return candles, nil
}

func (c *APIClient) Bonds(ctx context.Context) ([]PerpetualBondInfo, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	var bonds []PerpetualBondInfo
	if err := c.getJSON(ctx, fmt.Sprintf("%s/api/bonds", c.baseURL), &bonds); err != nil {
		return nil, err
	}
	return bonds, nil
}

func (c *APIClient) MacroIndicators(ctx context.Context) ([]MacroIndicator, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	var indicators []MacroIndicator
	if err := c.getJSON(ctx, fmt.Sprintf("%s/api/macro/indicators", c.baseURL), &indicators); err != nil {
		return nil, err
	}
	return indicators, nil
}

func (c *APIClient) TheoreticalFXRates(ctx context.Context) ([]TheoreticalFXRate, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	var rates []TheoreticalFXRate
	if err := c.getJSON(ctx, fmt.Sprintf("%s/api/fx/theoretical", c.baseURL), &rates); err != nil {
		return nil, err
	}
	return rates, nil
}

func (c *APIClient) Pools(ctx context.Context) ([]LiquidityPool, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	var pools []LiquidityPool
	if err := c.getJSON(ctx, fmt.Sprintf("%s/api/pools", c.baseURL), &pools); err != nil {
		return nil, err
	}
	return pools, nil
}

func (c *APIClient) SwapPool(ctx context.Context, poolID int64, request PoolSwapRequest) (*PoolSwapResult, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	if poolID < 0 {
		return nil, fmt.Errorf("pool_id must be non-negative")
	}
	endpoint := fmt.Sprintf("%s/api/pools/%d/swap", c.baseURL, poolID)
	if poolID == 0 {
		endpoint = fmt.Sprintf("%s/api/pools/0/swap", c.baseURL)
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	var result PoolSwapResult
	if err := c.postJSON(ctx, endpoint, payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *APIClient) IndexAction(ctx context.Context, assetID int64, action string, request IndexActionRequest) (*IndexActionResult, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	if assetID <= 0 {
		return nil, fmt.Errorf("asset_id must be positive")
	}
	if action == "" {
		return nil, fmt.Errorf("action required")
	}
	endpoint := fmt.Sprintf("%s/api/indices/%d/%s", c.baseURL, assetID, action)
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	var result IndexActionResult
	if err := c.postJSON(ctx, endpoint, payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *APIClient) Balances(ctx context.Context, userID int64) ([]Balance, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	endpoint := fmt.Sprintf("%s/api/portfolio/balances", c.baseURL)
	if userID != 0 {
		endpoint = fmt.Sprintf("%s?user_id=%d", endpoint, userID)
	}
	var balances []Balance
	if err := c.getJSON(ctx, endpoint, &balances); err != nil {
		return nil, err
	}
	return balances, nil
}

func (c *APIClient) PortfolioAssets(ctx context.Context, userID int64) ([]PortfolioAsset, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	endpoint := fmt.Sprintf("%s/api/portfolio/assets", c.baseURL)
	if userID != 0 {
		endpoint = fmt.Sprintf("%s?user_id=%d", endpoint, userID)
	}
	var assets []PortfolioAsset
	if err := c.getJSON(ctx, endpoint, &assets); err != nil {
		return nil, err
	}
	return assets, nil
}

func (c *APIClient) MarginPositions(ctx context.Context, userID int64) ([]MarginPosition, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	endpoint := fmt.Sprintf("%s/api/margin/positions", c.baseURL)
	if userID != 0 {
		endpoint = fmt.Sprintf("%s?user_id=%d", endpoint, userID)
	}
	var positions []MarginPosition
	if err := c.getJSON(ctx, endpoint, &positions); err != nil {
		return nil, err
	}
	return positions, nil
}

func (c *APIClient) Orders(ctx context.Context, status string, assetID, userID int64, limit, offset int) ([]engine.Order, error) {
	if c == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	endpoint := fmt.Sprintf("%s/api/orders", c.baseURL)
	query := url.Values{}
	if status != "" {
		query.Set("status", status)
	}
	if assetID != 0 {
		query.Set("asset_id", strconv.FormatInt(assetID, 10))
	}
	if userID != 0 {
		query.Set("user_id", strconv.FormatInt(userID, 10))
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		query.Set("offset", strconv.Itoa(offset))
	}
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, query.Encode())
	}
	var orders []engine.Order
	if err := c.getJSON(ctx, endpoint, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func (c *APIClient) getJSON(ctx context.Context, endpoint string, target interface{}) error {
	if c == nil {
		return fmt.Errorf("api client is nil")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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
	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *APIClient) postJSON(ctx context.Context, endpoint string, payload []byte, target interface{}) error {
	if c == nil {
		return fmt.Errorf("api client is nil")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return decodeAPIError(resp)
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
