package schwab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

// GetAccountInfo retrieves the account positions and balances.
func (c *Client) GetAccountInfo() (map[int64]AccountV2, error) {
	if err := c.UpdateToken("api"); err != nil && c.Debug {
		log.Printf("UpdateToken(api) warning: %v", err)
	}

	// Set header for account info (Python: Schwab-Client-Ids required for HoldingV2 in some cases)
	if acc, ok := c.Headers["schwab-client-account"]; ok {
		c.Headers["Schwab-Client-Ids"] = acc
	} else if len(c.AccountIDs) > 0 {
		c.Headers["Schwab-Client-Ids"] = c.AccountIDs[0]
	}

	req, err := http.NewRequest("GET", PositionsV2Url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		msg := string(body)
		if len(msg) > 500 {
			msg = msg[:500] + "..."
		}
		return nil, fmt.Errorf("API request failed with status: %d: %s", resp.StatusCode, msg)
	}

	var data AccountInfoV2Response
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	accounts := make(map[int64]AccountV2)
	for _, acc := range data.Accounts {
		id, _ := strconv.ParseInt(acc.AccountID, 10, 64)
		accounts[id] = acc
	}

	return accounts, nil
}

// GetAccountInfoV2 returns account info in the same shape as Python schwab-api get_account_info_v2()
// for 1:1 porting: map[accountID] -> { account_value, positions: [{ symbol, market_value, quantity }] }.
func (c *Client) GetAccountInfoV2() (map[string]AccountInfoV2Compat, error) {
	raw, err := c.GetAccountInfo()
	if err != nil {
		return nil, err
	}
	out := make(map[string]AccountInfoV2Compat)
	for id, acc := range raw {
		accID := strconv.FormatInt(id, 10)
		compat := AccountInfoV2Compat{
			AccountValue: acc.Totals.AccountValue,
			Positions:    nil,
		}
		for _, group := range acc.GroupedPositions {
			for _, row := range group.HoldingsRows {
				compat.Positions = append(compat.Positions, PositionRow{
					Symbol:      row.Symbol.Symbol,
					MarketValue: row.MarketValue.Val,
					Quantity:    row.Qty.Qty,
				})
			}
		}
		out[accID] = compat
	}
	return out, nil
}

// TradeV2 is the same as Trade; name matches Python schwab-api trade_v2() for 1:1 porting.
func (c *Client) TradeV2(ticker, side string, qty float64, accountID string, dryRun bool) ([]string, bool, error) {
	return c.Trade(ticker, side, qty, accountID, dryRun)
}

// Trade executes or verifies a trade.
// ticker: Symbol to trade
// side: "Buy" or "Sell"
// qty: Quantity
// accountId: Account ID
// dryRun: If true, only verifies the order
func (c *Client) Trade(ticker, side string, qty float64, accountId string, dryRun bool) ([]string, bool, error) {
	if side != "Buy" && side != "Sell" {
		return nil, false, fmt.Errorf("side must be 'Buy' or 'Sell'")
	}

	buySellCode := "49"
	if side == "Sell" {
		buySellCode = "50"
	}

	c.UpdateToken("update")

	// Construct request body
	requestBody := map[string]interface{}{
		"UserContext": map[string]interface{}{
			"AccountId":    accountId,
			"AccountColor": 0,
		},
		"OrderStrategy": map[string]interface{}{
			"PrimarySecurityType": 46, // Stock
			"CostBasisRequest": map[string]interface{}{
				"costBasisMethod":        "FIFO",
				"defaultCostBasisMethod": "FIFO",
			},
			"OrderType":         "49", // Market
			"LimitPrice":        "0",
			"StopPrice":         "0",
			"Duration":          "48", // Day
			"AllNoneIn":         false,
			"DoNotReduceIn":     false,
			"OrderStrategyType": 1,
			"OrderLegs": []map[string]interface{}{
				{
					"Quantity":       fmt.Sprintf("%f", qty),
					"LeavesQuantity": fmt.Sprintf("%f", qty),
					"Instrument":     map[string]interface{}{"Symbol": ticker},
					"SecurityType":   46,
					"Instruction":    buySellCode,
				},
			},
		},
		"OrderProcessingControl": 1, // Verification
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, false, err
	}

	// Make request
	req, err := http.NewRequest("POST", OrderVerificationV2Url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, false, err
	}

	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}
	// Required header
	req.Header.Set("schwab-resource-version", "1.0")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if c.Debug {
		log.Printf("Verification Response: %s", string(bodyBytes))
	}

	if resp.StatusCode != 200 {
		return []string{string(bodyBytes)}, false, nil
	}

	var verifyResp OrderVerificationResponse
	if err := json.Unmarshal(bodyBytes, &verifyResp); err != nil {
		return nil, false, err
	}

	messages := []string{}
	for _, msg := range verifyResp.OrderStrategy.OrderMessages {
		messages = append(messages, msg.Message)
	}

	// Check return codes (0, 10 are usually success/warning)
	validCodes := map[int]bool{0: true, 10: true}
	if !validCodes[verifyResp.OrderStrategy.OrderReturnCode] {
		return messages, false, nil
	}

	if dryRun {
		return messages, true, nil
	}

	// Proceed to execution
	orderId := verifyResp.OrderStrategy.OrderId
	if len(verifyResp.OrderStrategy.OrderLegs) > 0 {
		leg := verifyResp.OrderStrategy.OrderLegs[0]
		// Need to update ItemIssueId
		if legs, ok := requestBody["OrderStrategy"].(map[string]interface{})["OrderLegs"].([]map[string]interface{}); ok {
			legs[0]["Instrument"].(map[string]interface{})["ItemIssueId"] = leg.SchwabSecurityId
		}
	}

	// Update for execution
	requestBody["UserContext"].(map[string]interface{})["CustomerId"] = 0
	requestBody["OrderStrategy"].(map[string]interface{})["OrderId"] = orderId
	requestBody["OrderProcessingControl"] = 2 // Execution

	execBody, _ := json.Marshal(requestBody)

	c.UpdateToken("update")

	reqExec, _ := http.NewRequest("POST", OrderVerificationV2Url, bytes.NewBuffer(execBody))
	for k, v := range c.Headers {
		reqExec.Header.Set(k, v)
	}
	reqExec.Header.Set("schwab-resource-version", "1.0")
	reqExec.Header.Set("Content-Type", "application/json")

	respExec, err := c.HttpClient.Do(reqExec)
	if err != nil {
		return nil, false, err
	}
	defer respExec.Body.Close()

	execBytes, _ := io.ReadAll(respExec.Body)
	if c.Debug {
		log.Printf("Execution Response: %s", string(execBytes))
	}

	// Parse again
	// Re-using struct as response is similar
	var execResp OrderVerificationResponse
	json.Unmarshal(execBytes, &execResp)

	execMessages := []string{}
	for _, msg := range execResp.OrderStrategy.OrderMessages {
		execMessages = append(execMessages, msg.Message)
	}

	if validCodes[execResp.OrderStrategy.OrderReturnCode] {
		return execMessages, true, nil
	}

	return execMessages, false, nil
}
