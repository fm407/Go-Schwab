package schwab

import "encoding/json"

// flexString unmarshals a JSON value that may be a string or an object (e.g. {"description":"..."}); returns a string.
type flexString string

func (s *flexString) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*s = ""
		return nil
	}
	if data[0] == '"' {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		*s = flexString(str)
		return nil
	}
	if data[0] == '{' {
		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return err
		}
		for _, key := range []string{"description", "text", "value"} {
			if v, ok := obj[key]; ok {
				if str, ok := v.(string); ok {
					*s = flexString(str)
					return nil
				}
			}
		}
	}
	*s = ""
	return nil
}

// AccountInfoV2Response represents the response from PositionsV2Url
type AccountInfoV2Response struct {
	Accounts []AccountV2 `json:"accounts"`
}

type AccountV2 struct {
	AccountID        string            `json:"accountId"`
	Totals           AccountTotals     `json:"totals"`
	GroupedPositions []GroupedPosition `json:"groupedPositions"`
}

type AccountTotals struct {
	MarketValue     float64 `json:"marketValue"`
	CashInvestments float64 `json:"cashInvestments"`
	AccountValue    float64 `json:"accountValue"`
	CostBasis       float64 `json:"costBasis"`
}

type GroupedPosition struct {
	GroupName    string       `json:"groupName"`
	HoldingsRows []HoldingRow `json:"holdingsRows"`
}

type HoldingRow struct {
	Symbol      SymbolInfo    `json:"symbol"`
	Description flexString    `json:"description"` // API may send string or object
	Qty         QtyInfo       `json:"qty"`
	CostBasis   CostBasisInfo `json:"costBasis"`
	MarketValue MarketValInfo `json:"marketValue"`
}

type SymbolInfo struct {
	Symbol string `json:"symbol"`
	SSID   int64  `json:"ssId"`
}

type QtyInfo struct {
	Qty float64 `json:"qty"`
}

type CostBasisInfo struct {
	CostBasis float64 `json:"cstBasis"`
}

type MarketValInfo struct {
	Val float64 `json:"val"`
}

// OrderVerificationResponse represents the response for order verification
type OrderVerificationResponse struct {
	OrderStrategy OrderStrategy `json:"orderStrategy"`
}

type OrderStrategy struct {
	OrderId         int64          `json:"orderId"`
	OrderMessages   []OrderMessage `json:"orderMessages"`
	OrderReturnCode int            `json:"orderReturnCode"`
	OrderLegs       []OrderLeg     `json:"orderLegs"`
}

type OrderMessage struct {
	Message string `json:"message"`
}

type OrderLeg struct {
	SchwabSecurityId int64 `json:"schwabSecurityId"`
}

// AccountInfoV2Compat matches the Python schwab-api get_account_info_v2() shape for 1:1 porting.
type AccountInfoV2Compat struct {
	AccountValue float64         `json:"account_value"`
	Positions    []PositionRow   `json:"positions"`
}

// PositionRow matches one entry in the Python "positions" list (symbol, market_value, quantity).
type PositionRow struct {
	Symbol      string  `json:"symbol"`
	MarketValue float64 `json:"market_value"`
	Quantity    float64 `json:"quantity"`
}
