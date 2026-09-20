package httpbroker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"example.com/resilient-broker-purchases/internal/domain"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{}}
}

func (c *Client) Submit(ctx context.Context, purchase domain.PurchaseRequest, key string) (domain.BrokerResult, error) {
	body, err := json.Marshal(purchase)
	if err != nil {
		return domain.BrokerResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/orders", bytes.NewReader(body))
	if err != nil {
		return domain.BrokerResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	response, err := c.http.Do(req)
	if err != nil {
		return domain.BrokerResult{}, err
	}
	defer response.Body.Close()
	var result domain.BrokerResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return domain.BrokerResult{}, fmt.Errorf("decode broker response: %w", err)
	}
	result.StatusCode = response.StatusCode
	return result, nil
}

func (c *Client) Lookup(ctx context.Context, key string) (domain.BrokerResult, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/orders/"+url.PathEscape(key), nil)
	if err != nil {
		return domain.BrokerResult{}, false, err
	}
	response, err := c.http.Do(req)
	if err != nil {
		return domain.BrokerResult{}, false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return domain.BrokerResult{}, false, nil
	}
	if response.StatusCode >= 500 {
		return domain.BrokerResult{}, false, fmt.Errorf("broker lookup returned %d", response.StatusCode)
	}
	var result domain.BrokerResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return domain.BrokerResult{}, false, err
	}
	result.StatusCode = response.StatusCode
	return result, true, nil
}
