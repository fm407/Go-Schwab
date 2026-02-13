package schwab

import (
	"strings"
	"testing"
)

func TestEndpoints_NonEmpty(t *testing.T) {
	urls := []string{
		HomepageUrl,
		PositionsV2Url,
		OrderVerificationV2Url,
		TickerQuotesV2Url,
	}
	for _, u := range urls {
		if u == "" {
			t.Errorf("endpoint is empty")
		}
		if !strings.HasPrefix(u, "https://") {
			t.Errorf("endpoint %q should use https", u)
		}
	}
}
