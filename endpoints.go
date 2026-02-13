package schwab

const (
	HomepageUrl            = "https://www.schwab.com/"
	AccountSummaryUrl      = "https://client.schwab.com/clientapps/accounts/summary/"
	TradeTicketUrl         = "https://client.schwab.com/app/trade/tom/trade?ShowUN=YES"
	OrderVerificationV2Url = "https://ausgateway.schwab.com/api/is.TradeOrderManagementWeb/v1/TradeOrderManagementWebPort/orders"
	AccountInfoV2Url       = "https://ausgateway.schwab.com/api/is.TradeOrderManagementWeb/v1/TradeOrderManagementWebPort/customer/accounts"
	PositionsV2Url         = "https://ausgateway.schwab.com/api/is.Holdings/V1/Holdings/HoldingV2"
	TickerQuotesV2Url      = "https://ausgateway.schwab.com/api/is.TradeOrderManagementWeb/v1/TradeOrderManagementWebPort/market/quotes/list"
	OrdersV2Url            = "https://ausgateway.schwab.com/api/is.TradeOrderStatusWeb/ITradeOrderStatusWeb/ITradeOrderStatusWebPort/orders/listView?DateRange=All&OrderStatusType=All&SecurityType=AllSecurities&Type=All&ShowAdvanceOrder=true&SortOrder=Ascending&SortColumn=Status&CostMethod=M&IsSimOrManagedAccount=false&EnableDateFilterByActivity=true"
	CancelOrderV2Url       = "https://ausgateway.schwab.com/api/is.TradeOrderStatusWeb/ITradeOrderStatusWeb/ITradeOrderStatusWebPort/orders/cancelorder"
	TransactionHistoryV2Url = "https://ausgateway.schwab.com/api/is.TransactionHistoryWeb/TransactionHistoryInterface/TransactionHistory/brokerage/transactions/export"
	LotDetailsV2Url        = "https://ausgateway.schwab.com/api/is.Holdings/V1/Lots"
	OptionChainsV2Url      = "https://ausgateway.schwab.com/api/is.CSOptionChainsWeb/v1/OptionChainsPort/OptionChains/chains"

	// Old API
	PositionsDataUrl       = "https://client.schwab.com/api/PositionV2/PositionsDataV2"
	OrderVerificationUrl   = "https://client.schwab.com/api/ts/stamp/verifyOrder"
	OrderConfirmationUrl   = "https://client.schwab.com/api/ts/stamp/confirmorder"
)
