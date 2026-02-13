# go-schwab

Go client for Charles Schwab that mirrors the Python [schwab-api](https://github.com/MaxxRK/schwab-api) / [NelsonDane/auto-rsa](https://github.com/NelsonDane/auto-rsa) flow: browser login via Playwright, then session + bearer token for API calls (account info, trading).

**Module:** `github.com/Auto-RSA-Safe/go-schwab`

---

## Features

- **Browser-based login** — Playwright + Chromium with username, password, and TOTP
- **Session capture** — Bearer token, cookies, and request headers (e.g. `Schwab-ChannelCode`) from the trade page
- **Account info** — Positions and balances via HoldingV2 (`GetAccountInfo`, `GetAccountInfoV2`)
- **Trading** — Market order verify/execute via `Trade` / `TradeV2` (Buy/Sell, dry run)
- **Python parity** — Same logical flow and response shapes as Python schwab-api where applicable; see [SCHWAB_API_1TO1.md](SCHWAB_API_1TO1.md) for mapping

---

## Prerequisites

- **Go** 1.21+ (or match [go.mod](go.mod), e.g. 1.25)
- **Playwright Chromium** (one-time install):
  ```bash
  go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
  ```
  Or with npm: `npx playwright install chromium`

---

## Installation

Clone the repo (or add as a Go module). From the `go-schwab` directory:

```bash
go mod download
go build ./...
```

No global install; use as a Go module.

---

## Configuration

Set environment variables (e.g. in a `.env` file in or above `go-schwab`). **Do not commit `.env`.**

| Variable | Description |
|----------|-------------|
| `SCHWAB` | `username:password:totpSecret`. Comma-separated for multiple accounts. Use `NA` for no TOTP. |
| `SCHWAB_ACCOUNT_NUMBERS` | Optional. Account number(s), colon-separated (e.g. `30110372`). Used for `Schwab-Client-Ids` on HoldingV2. |

The example loads `.env` from `../../.env`, `../.env`, or `.env` (see [cmd/example/main.go](cmd/example/main.go)).

---

## Usage

### Quick run

From `go-schwab`:

```bash
go run ./cmd/example
```

Requires `SCHWAB` (and optionally `SCHWAB_ACCOUNT_NUMBERS`) in env or `.env`.

### As a library

```go
package main

import (
    "github.com/Auto-RSA-Safe/go-schwab"
)

func main() {
    client := schwab.NewClient(true) // true = debug logging

    // Optional: required for GetAccountInfo when API expects Schwab-Client-Ids
    client.AccountIDs = []string{"30110372"}

    err := client.Login(username, password, totpSecret)
    if err != nil {
        // handle error
    }

    // Account info (raw API shape)
    accounts, err := client.GetAccountInfo()
    if err != nil {
        // handle error
    }
    for id, acc := range accounts {
        _ = id
        _ = acc.Totals.MarketValue
        _ = acc.GroupedPositions
    }

    // Account info (Python-shaped: account_value + positions)
    info, err := client.GetAccountInfoV2()
    if err != nil {
        // handle error
    }
    for accID, acc := range info {
        _ = accID
        _ = acc.AccountValue
        for _, pos := range acc.Positions {
            _ = pos.Symbol
            _ = pos.MarketValue
            _ = pos.Quantity
        }
    }

    // Trade (dry run)
    messages, success, err := client.TradeV2("AAPL", "Buy", 1, "30110372", true)
    if err != nil {
        // handle error
    }
    if !success {
        // use messages
    }
}
```

`Login` opens a headed Chromium window, performs the Schwab login flow (5s wait + page refresh), then captures token and cookies and closes the browser.

---

## Technical details

### Login flow

1. Playwright launches Chromium (headed), navigates to Schwab login.
2. Fills the login iframe (`iframe#schwablmslogin`) via FrameLocator: selects Trade landing, Login ID, password + TOTP, Enter.
3. Intercepts the `balancespositions` request and captures **all** request headers (Authorization, Schwab-ChannelCode, etc.).
4. Waits 5 seconds, then reloads the page so the session is fully established.
5. Waits for URL `app/trade` and selector `#_txtSymbol`, then captures cookies for `www.schwab.com`, `client.schwab.com`, and `ausgateway.schwab.com`.
6. Builds `Client.Headers` and `Client.BearerToken` for subsequent API calls.

### APIs used

- **Auth:** Bearer token from the intercepted `balancespositions` request; refresh via `https://client.schwab.com/api/auth/authorize/scope/{api|update}`.
- **Holdings:** [endpoints.go](endpoints.go) `PositionsV2Url` (HoldingV2). Account info sends `Schwab-Client-Ids` when `Client.AccountIDs` is set or `schwab-client-account` is in headers.
- **Trading:** [endpoints.go](endpoints.go) `OrderVerificationV2Url` for verify (POST JSON) and execute.

### Dependencies

See [go.mod](go.mod): [playwright-go](https://github.com/playwright-community/playwright-go), [godotenv](https://github.com/joho/godotenv), [otp](https://github.com/pquerna/otp) (TOTP). This project does not use an official Schwab SDK; it uses the same reverse-engineered endpoints as the Python libraries.

---

## Project layout

| File | Description |
|------|-------------|
| [auth.go](auth.go) | Playwright login, header/cookie capture |
| [api.go](api.go) | GetAccountInfo, GetAccountInfoV2, Trade, TradeV2, UpdateToken |
| [client.go](client.go) | Client struct, NewClient |
| [endpoints.go](endpoints.go) | URL constants |
| [models.go](models.go) | Response types (AccountV2, HoldingRow, OrderVerificationResponse, etc.) |
| [cmd/example/main.go](cmd/example/main.go) | Example: load env, login, fetch and print account info |
| [SCHWAB_API_1TO1.md](SCHWAB_API_1TO1.md) | Python ↔ Go API mapping |

---

## Python / auto-rsa parity

- Same login idea as Python (browser + session). Go does **not** cache the session to disk; each run performs a fresh browser login.
- Same API surface for account info and trading. See [SCHWAB_API_1TO1.md](SCHWAB_API_1TO1.md) for method and response mapping.
- Intended for use alongside or as a port of [NelsonDane/auto-rsa](https://github.com/NelsonDane/auto-rsa) and [MaxxRK/schwab-api](https://github.com/MaxxRK/schwab-api).

---

## Testing

Unit tests in `client_test.go`, `api_test.go`, and `endpoints_test.go`:

```bash
go test ./...
```

---

## Troubleshooting

- **Login fails** — Check `SCHWAB` format (`username:password:totpSecret`), ensure Playwright Chromium is installed, and that the login iframe/selectors are still valid on Schwab’s site.
- **400 / 431 from API** — Ensure `SCHWAB_ACCOUNT_NUMBERS` is set when required for HoldingV2; 431 can occur if the Cookie header is too large (this client limits cookies to Schwab domains only).
- **JSON unmarshal on `description`** — The API sometimes returns `description` as an object; [models.go](models.go) uses a `flexString` type to accept both string and object.

---

## Disclaimer

This project is **unofficial** and not affiliated with Charles Schwab. Use at your own risk. You are responsible for securing your credentials and for compliance with Schwab’s terms of use.

---

## License

See the repository license.
