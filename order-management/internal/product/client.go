package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

type RemoteClient interface {
	FetchProductPrice(ctx context.Context, productID uint) (float64, error)
}

type remoteClient struct {
	baseURL    string
	httpClient *http.Client
	cb         *gobreaker.CircuitBreaker
}

func NewRemoteClient(baseURL string) RemoteClient {
	st := gobreaker.Settings{
		Name:        "ProductServiceCircuitBreaker",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
	}

	return &remoteClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		cb: gobreaker.NewCircuitBreaker(st),
	}
}

func (c *remoteClient) FetchProductPrice(ctx context.Context, productID uint) (float64, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		var price float64
		var execErr error

		// Retry with Exponential Backoff
		backoff := 50 * time.Millisecond
		maxRetries := 3

		for attempt := 0; attempt < maxRetries; attempt++ {
			select {
			case <-ctx.Done():
				return 0.0, ctx.Err()
			default:
			}

			price, execErr = c.doHTTPRequest(ctx, productID)
			if execErr == nil {
				return price, nil
			}

			time.Sleep(backoff)
			backoff *= 2
		}

		return 0.0, fmt.Errorf("failed after %d attempts: %w", maxRetries, execErr)
	})

	if err != nil {
		return 0.0, err
	}
	return result.(float64), nil
}

func (c *remoteClient) doHTTPRequest(ctx context.Context, productID uint) (float64, error) {
	reqURL := fmt.Sprintf("%s/api/v1/products/%d", c.baseURL, productID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("remote service error: status %d", resp.StatusCode)
	}

	var payload struct {
		Price float64 `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, errors.New("failed to parse response body")
	}

	return payload.Price, nil
}