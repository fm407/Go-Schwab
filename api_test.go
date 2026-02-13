package schwab

import (
	"testing"
)

func TestTrade_InvalidSide(t *testing.T) {
	c := NewClient(false)
	_, ok, err := c.Trade("AAPL", "INVALID", 1, "123", true)
	if err == nil {
		t.Fatal("expected error for invalid side")
	}
	if ok {
		t.Error("expected ok false for invalid side")
	}
	if err.Error() != "side must be 'Buy' or 'Sell'" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTrade_EmptySide(t *testing.T) {
	c := NewClient(false)
	_, _, err := c.Trade("AAPL", "", 1, "123", true)
	if err == nil {
		t.Fatal("expected error for empty side")
	}
}
