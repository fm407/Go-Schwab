package schwab

import (
	"net/http"
)

// Client represents the Schwab API client
type Client struct {
	HttpClient  *http.Client
	Headers     map[string]string
	BearerToken string
	Debug       bool
	// AccountIDs optionally specifies account number(s) for API calls that require Schwab-Client-Ids (e.g. HoldingV2).
	// If set, GetAccountInfo will send the first ID in the Schwab-Client-Ids header.
	AccountIDs []string
}

// NewClient creates a new Schwab API client
func NewClient(debug bool) *Client {
	return &Client{
		HttpClient: &http.Client{},
		Headers:    make(map[string]string),
		Debug:      debug,
	}
}
