package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Auto-RSA-Safe/go-schwab"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env from project root (assuming we run from go-schwab or root)
	// Try loading from parent directory if not found in current
	if err := godotenv.Load("../../.env"); err != nil {
		if err := godotenv.Load("../.env"); err != nil {
			log.Println("Could not load ../../.env or ../.env, trying .env")
			godotenv.Load(".env")
		}
	}

	schwabCreds := os.Getenv("SCHWAB")
	if schwabCreds == "" {
		log.Fatal("SCHWAB environment variable not set")
	}

	// Format: username:password:totpSecret, ...
	accounts := strings.Split(schwabCreds, ",")
	if len(accounts) == 0 {
		log.Fatal("No accounts found in SCHWAB env var")
	}

	// Just use the first account for testing
	firstAccount := strings.Split(accounts[0], ":")
	if len(firstAccount) < 3 {
		log.Fatal("Invalid account format. Expected username:password:totpSecret")
	}

	username := firstAccount[0]
	password := firstAccount[1]
	totpSecret := firstAccount[2]

	if totpSecret == "NA" {
		totpSecret = ""
	}

	log.Printf("Attempting login for user: %s", username)

	client := schwab.NewClient(true) // Enable debug
	if accountNumbers := os.Getenv("SCHWAB_ACCOUNT_NUMBERS"); accountNumbers != "" {
		client.AccountIDs = strings.Split(strings.TrimSpace(accountNumbers), ":")
	}
	err := client.Login(username, password, totpSecret)
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	fmt.Println("Login Successful!")
	fmt.Printf("Bearer Token: %s...\n", client.BearerToken[:20]) // Print partial token

	fmt.Println("Fetching Account Info...")
	accountsMap, err := client.GetAccountInfo()
	if err != nil {
		log.Fatalf("Failed to get account info: %v", err)
	}

	for id, acc := range accountsMap {
		fmt.Printf("Account ID: %d\n", id)
		fmt.Printf("  Market Value: %.2f\n", acc.Totals.MarketValue)
		fmt.Printf("  Cash: %.2f\n", acc.Totals.CashInvestments)
		for _, group := range acc.GroupedPositions {
			fmt.Printf("  Group: %s\n", group.GroupName)
			for _, pos := range group.HoldingsRows {
				fmt.Printf("    - %s (%s): %.2f shares @ $%.2f\n",
					pos.Symbol.Symbol, string(pos.Description), pos.Qty.Qty, pos.MarketValue.Val)
			}
		}
	}
}
