package schwab

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/pquerna/otp/totp"
)

// Login performs the login flow using Playwright (same approach as Python schwab-api / MaxxRK fork).
func (c *Client) Login(username, password, totpSecret string) error {
	var fullPassword = password
	if totpSecret != "" {
		code, err := totp.GenerateCode(totpSecret, time.Now())
		if err != nil {
			return fmt.Errorf("failed to generate TOTP code: %w", err)
		}
		fullPassword = password + code
	}

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("failed to start Playwright: %w", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--disable-automation",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"),
		Viewport:  &playwright.Size{Width: 1920, Height: 1080},
	})
	if err != nil {
		return fmt.Errorf("failed to create context: %w", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	// Capture all headers from balancespositions request (like Python: self.headers = await route.request.all_headers()).
	// The browser sends Authorization, Schwab-ChannelCode, and other headers required by the API.
	headersChan := make(chan map[string]string, 1)
	balancesRe := regexp.MustCompile(`balancespositions`)
	err = page.Route(balancesRe, func(route playwright.Route) {
		req := route.Request()
		all, _ := req.AllHeaders()
		if all != nil && all["authorization"] != "" {
			select {
			case headersChan <- all:
			default:
			}
		}
		_ = route.Continue()
	})
	if err != nil {
		return fmt.Errorf("failed to set route: %w", err)
	}

	if c.Debug {
		log.Println("Navigating to Schwab login page...")
	}

	_, err = page.Goto(HomepageUrl, playwright.PageGotoOptions{Timeout: playwright.Float(60000)})
	if err != nil {
		return fmt.Errorf("failed to navigate to login page: %w", err)
	}

	_, err = page.WaitForSelector("iframe#schwablmslogin", playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(30000)})
	if err != nil {
		return fmt.Errorf("failed to wait for login iframe: %w", err)
	}

	// Use FrameLocator so we target the iframe by selector; page.Frame(Name) can be nil until frame is attached.
	fl := page.FrameLocator("iframe#schwablmslogin")

	if c.Debug {
		log.Println("Entering credentials...")
	}

	// Match Python: select_option(landingPageOptions, index=3) for Trade (two selects in iframe; use first)
	_, err = fl.Locator("select#landingPageOptions").First().SelectOption(playwright.SelectOptionValues{Indexes: &[]int{3}})
	if err != nil {
		return fmt.Errorf("failed to select Trade: %w", err)
	}

	loginSel := `[placeholder="Login ID"]`
	passSel := `[placeholder="Password"]`

	err = fl.Locator(loginSel).Fill(username)
	if err != nil {
		return fmt.Errorf("failed to fill login ID: %w", err)
	}
	err = fl.Locator(loginSel).Press("Tab")
	if err != nil {
		return fmt.Errorf("failed to tab to password: %w", err)
	}
	err = fl.Locator(passSel).Fill(fullPassword)
	if err != nil {
		return fmt.Errorf("failed to fill password: %w", err)
	}
	err = fl.Locator(passSel).Press("Enter")
	if err != nil {
		return fmt.Errorf("failed to submit login: %w", err)
	}

	// Wait 5 seconds after login then refresh so the session is fully established
	page.WaitForTimeout(5000)
	_, err = page.Reload(playwright.PageReloadOptions{Timeout: playwright.Float(30000)})
	if err != nil {
		return fmt.Errorf("refresh after login: %w", err)
	}

	if c.Debug {
		log.Println("Waiting for login to complete and token capture...")
	}

	var capturedHeaders map[string]string
	select {
	case capturedHeaders = <-headersChan:
		c.BearerToken = capturedHeaders["authorization"]
		if c.Debug {
			log.Println("Captured Bearer Token!")
		}
	case <-time.After(60 * time.Second):
		return fmt.Errorf("timed out waiting for authorization header")
	}

	// Match Python: wait for app/trade and #_txtSymbol before capturing cookies
	err = page.WaitForURL(regexp.MustCompile(`app/trade`), playwright.PageWaitForURLOptions{Timeout: playwright.Float(60000)})
	if err != nil {
		return fmt.Errorf("wait for trade URL: %w", err)
	}
	_, err = page.WaitForSelector("#_txtSymbol", playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(30000)})
	if err != nil {
		return fmt.Errorf("wait for trade page: %w", err)
	}
	// Refresh the trade page so session/cookies are fully established (same as manual refresh).
	_, err = page.Reload(playwright.PageReloadOptions{Timeout: playwright.Float(30000)})
	if err != nil {
		return fmt.Errorf("refresh trade page: %w", err)
	}
	_, err = page.WaitForSelector("#_txtSymbol", playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(30000)})
	if err != nil {
		return fmt.Errorf("wait for trade page after refresh: %w", err)
	}
	page.WaitForTimeout(1500) // let cookies settle after refresh

	// Cookies for Schwab API domains only (avoid 431 Request Header Fields Too Large from sending every cookie).
	byName := make(map[string]string)
	for _, u := range []string{
		"https://www.schwab.com",
		"https://client.schwab.com",
		"https://ausgateway.schwab.com",
	} {
		cookies, err := context.Cookies(u)
		if err != nil {
			continue
		}
		for _, k := range cookies {
			byName[k.Name] = k.Value
		}
	}
	var parts []string
	for name, value := range byName {
		parts = append(parts, fmt.Sprintf("%s=%s", name, value))
	}
	cookiesStr := strings.Join(parts, "; ")
	if cookiesStr == "" {
		return fmt.Errorf("no session cookies captured for Schwab domains")
	}

	// Use captured request headers (includes Schwab-ChannelCode, etc.); override with our Cookie and Bearer.
	c.Headers = make(map[string]string)
	for k, v := range capturedHeaders {
		c.Headers[k] = v
	}
	c.Headers["Authorization"] = c.BearerToken
	c.Headers["Cookie"] = cookiesStr

	return nil
}

// updateToken updates the bearer token if expired.
func (c *Client) UpdateToken(tokenType string) error {
	url := fmt.Sprintf("https://client.schwab.com/api/auth/authorize/scope/%s", tokenType)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		if c.Debug {
			log.Printf("Token update failed with status: %d. Re-logging in...", resp.StatusCode)
		}
		return fmt.Errorf("failed to update token, status: %d", resp.StatusCode)
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	c.BearerToken = "Bearer " + result.Token
	c.Headers["Authorization"] = c.BearerToken
	return nil
}
