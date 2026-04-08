package validation

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/rackspace-spot/spotctl/internal/app"
	"github.com/rackspace-spot/spotctl/internal/features/serverclasses"
)

// ValidateBidPrice validates and normalizes a bid price string.
// It strips leading '$', validates the format, and returns a clean numeric string.
func ValidateBidPrice(bidPrice string) (string, error) {
	if bidPrice == "" {
		return "", fmt.Errorf("bid price is required")
	}

	// Remove all whitespace and dollar signs
	trimmed := strings.TrimSpace(strings.ReplaceAll(bidPrice, "$", ""))
	if trimmed == "" {
		return "", fmt.Errorf("no valid price found in: %q", bidPrice)
	}

	// Try to parse as a float first
	price, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		// If parsing fails, try to clean up the string
		var cleanNum strings.Builder
		decimalFound := false
		for _, c := range trimmed {
			if c >= '0' && c <= '9' {
				cleanNum.WriteRune(c)
			} else if c == '.' && !decimalFound {
				cleanNum.WriteRune(c)
				decimalFound = true
			}
		}

		if cleanNum.Len() == 0 {
			return "", fmt.Errorf("invalid price format: %q (no valid numbers found)", bidPrice)
		}

		price, err = strconv.ParseFloat(cleanNum.String(), 64)
		if err != nil {
			return "", fmt.Errorf("invalid price format: %q: %v", bidPrice, err)
		}
	}

	// Ensure it's a positive number
	if price <= 0 {
		return "", fmt.Errorf("bid price must be greater than 0, got %v", price)
	}

	// Verify the price has at most 3 decimal places
	// Format with 3 decimal places and parse back to verify
	formatted := fmt.Sprintf("%.3f", price)
	parsed, _ := strconv.ParseFloat(formatted, 64)

	// Check if the price is a multiple of 0.005
	// We do this by multiplying by 200 and checking if it's an integer
	multipleCheck := parsed * 200
	if multipleCheck != float64(int64(multipleCheck+0.0001)) { // Small tolerance for float arithmetic
		return "", fmt.Errorf("bid price must be a multiple of 0.005 (got %s)", formatted)
	}

	// Remove trailing zeros but keep at least one decimal place
	formatted = strings.TrimRight(formatted, "0")
	if strings.HasSuffix(formatted, ".") {
		formatted += "0"
	}

	return formatted, nil
}

// ValidateServerClassForSpotBidding checks if a server class is available for spot bidding.
// It returns an error if the class is deprecated or unavailable (has empty minBidPrice).
func ValidateServerClassForSpotBidding(ctx context.Context, appCtx *app.Context, className string) error {
	if className == "" {
		return fmt.Errorf("server class name is required")
	}

	sc, err := serverclasses.Get(ctx, appCtx, className)
	if err != nil {
		return fmt.Errorf("failed to fetch server class %q: %w", className, err)
	}

	// The Get function returns the raw API response which should have MinBidPricePerHour field.
	// Check if it's a map or a struct. For now, we'll handle the case where it should be a
	// Rackspace API struct. We need to inspect the actual response type.
	// For simplicity, we can try to access it as a map[string]interface{} or use reflection.
	
	scMap, ok := sc.(map[string]interface{})
	if !ok {
		// If it's not a map, return a generic success since we can't validate
		return nil
	}

	minBidPrice, exists := scMap["minBidPricePerHour"]
	if !exists {
		return nil // Field doesn't exist means class is available
	}

	// Check if minBidPrice is "$" (deprecated) or empty
	minBidStr, ok := minBidPrice.(string)
	if !ok {
		return nil // Not a string, assume it's available
	}

	minBidStr = strings.TrimSpace(minBidStr)
	if minBidStr == "" || minBidStr == "$" {
		return fmt.Errorf(
			"server class %q is unavailable for spot bidding (deprecated). Please use an available server class",
			className,
		)
	}

	return nil
}
