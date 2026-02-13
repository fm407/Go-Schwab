package schwab

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	// debug off
	c := NewClient(false)
	if c == nil {
		t.Fatal("NewClient(false) returned nil")
	}
	if c.HttpClient == nil {
		t.Error("HttpClient is nil")
	}
	if c.Headers == nil {
		t.Error("Headers is nil")
	}
	if c.Debug != false {
		t.Errorf("Debug = %v, want false", c.Debug)
	}
	if c.BearerToken != "" {
		t.Errorf("BearerToken = %q, want empty", c.BearerToken)
	}

	// debug on
	c2 := NewClient(true)
	if c2 == nil {
		t.Fatal("NewClient(true) returned nil")
	}
	if !c2.Debug {
		t.Error("Debug should be true")
	}
}
