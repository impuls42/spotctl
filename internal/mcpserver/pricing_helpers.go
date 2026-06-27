package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PriceHistoryPoint represents a single price history data point
type PriceHistoryPoint struct {
	RunAt       int64   `json:"run_at"`       // Unix timestamp of the auction
	HammerPrice float64 `json:"hammer_price"` // Final clearing price in USD
}

// PriceHistory represents the price history response for a server class
type PriceHistory struct {
	Auction string              `json:"auction"` // Server class name
	History []PriceHistoryPoint `json:"history"`
}

// PercentilesData represents the percentile distribution response
type PercentilesData map[string]interface{}

// ComparablePricesData represents the comparable prices response
type ComparablePricesData map[string]interface{}

const (
	s3BaseURL   = "https://ngpc-prod-public-data.s3.us-east-2.amazonaws.com"
	httpTimeout = 30 * time.Second
)

// GetPriceHistory retrieves price history for a specific server class from S3
func GetPriceHistory(ctx context.Context, serverClass string) (*PriceHistory, error) {
	if serverClass == "" {
		return nil, fmt.Errorf("server class is required")
	}

	url := fmt.Sprintf("%s/price_history.json?server_class=%s", s3BaseURL, serverClass)
	return getPriceHistoryFromURL(ctx, url)
}

// getPriceHistoryFromURL is an internal helper to fetch from a URL
func getPriceHistoryFromURL(ctx context.Context, url string) (*PriceHistory, error) {
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch price history: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("S3 returned status %d: %s", resp.StatusCode, body)
	}

	var history PriceHistory
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, fmt.Errorf("failed to decode price history response: %w", err)
	}

	return &history, nil
}

// GetPricePercentiles retrieves price percentile distributions from S3
func GetPricePercentiles(ctx context.Context, region, serverClass string) (PercentilesData, error) {
	url := fmt.Sprintf("%s/percentiles.json", s3BaseURL)
	data, err := getPercentileDataFromURL(ctx, url)
	if err != nil {
		return nil, err
	}

	// Filter by region and/or serverclass if provided
	result := PercentilesData{}
	if region == "" && serverClass == "" {
		// Return all data
		return data, nil
	}

	// Filter the response
	if regionsData, ok := data["regions"].(map[string]interface{}); ok {
		filteredRegions := make(map[string]interface{})
		for r, rData := range regionsData {
			if region != "" && r != region {
				continue
			}
			if serverClass == "" {
				filteredRegions[r] = rData
			} else {
				// Filter by server class within this region
				rMap, ok := rData.(map[string]interface{})
				if !ok {
					continue
				}
				if serverclassesData, ok := rMap["serverclasses"].(map[string]interface{}); ok {
					filteredServerclasses := make(map[string]interface{})
					if scData, ok := serverclassesData[serverClass]; ok {
						filteredServerclasses[serverClass] = scData
					}
					if len(filteredServerclasses) > 0 {
						rMap["serverclasses"] = filteredServerclasses
						filteredRegions[r] = rMap
					}
				}
			}
		}
		if len(filteredRegions) > 0 {
			result["regions"] = filteredRegions
		}
	}

	return result, nil
}

// getPercentileDataFromURL is an internal helper to fetch from a URL
func getPercentileDataFromURL(ctx context.Context, url string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch percentiles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("S3 returned status %d: %s", resp.StatusCode, body)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode percentiles response: %w", err)
	}

	return data, nil
}

// GetComparablePrices retrieves comparable hyperscaler prices from S3
func GetComparablePrices(ctx context.Context, region, serverClass string) (ComparablePricesData, error) {
	url := fmt.Sprintf("%s/comparable_prices.json", s3BaseURL)
	data, err := getComparablePriceDataFromURL(ctx, url)
	if err != nil {
		return nil, err
	}

	// Filter by region and/or serverclass if provided
	result := ComparablePricesData{}
	if region == "" && serverClass == "" {
		// Return all data
		return data, nil
	}

	// Filter the response
	if regionsData, ok := data["regions"].(map[string]interface{}); ok {
		filteredRegions := make(map[string]interface{})
		for r, rData := range regionsData {
			if region != "" && r != region {
				continue
			}
			rMap, ok := rData.(map[string]interface{})
			if !ok {
				continue
			}
			if serverClass == "" {
				filteredRegions[r] = rData
			} else {
				// Filter by server class within this region
				if scData, ok := rMap[serverClass]; ok {
					filteredServerclasses := make(map[string]interface{})
					filteredServerclasses[serverClass] = scData
					filteredRegions[r] = filteredServerclasses
				}
			}
		}
		if len(filteredRegions) > 0 {
			result["regions"] = filteredRegions
		}
	}

	return result, nil
}

// getComparablePriceDataFromURL is an internal helper to fetch from a URL
func getComparablePriceDataFromURL(ctx context.Context, url string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch comparable prices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("S3 returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode comparable prices response: %w", err)
	}

	return data, nil
}
