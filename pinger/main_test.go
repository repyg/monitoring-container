package main

import (
	"testing"
	"time"
)

func TestPingResult(t *testing.T) {
	result := &PingResult{
		IP:          "127.0.0.1",
		PingTime:    10.5,
		LastSuccess: time.Now().Format(time.RFC3339),
		Name:        "test-container",
		Status:      "Up",
		Created:     time.Now().Format(time.RFC3339),
	}

	if result.IP != "127.0.0.1" {
		t.Errorf("expected IP to be 127.0.0.1, got %s", result.IP)
	}
}
