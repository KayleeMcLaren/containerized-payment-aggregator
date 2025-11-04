package providers

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// AirtelProvider implements the PaymentProvider interface.
type AirtelProvider struct{}

func NewAirtelProvider() *AirtelProvider {
	return &AirtelProvider{}
}

func (p *AirtelProvider) Name() string {
	return "AIRTEL_MONEY"
}

// ProcessPayment simulates interaction with the Airtel Money API.
func (p *AirtelProvider) ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	// Simulate Network Latency (200ms to 800ms)
	delay := time.Duration(rand.Intn(600)+200) * time.Millisecond
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(delay):
		// Continue
	}

	// 1. Simulate external API Errors (80% chance of 500 server error)
	if rand.Float64() < 0.80 {
		// Create the response object
		res := &PaymentResponse{
			Status:       "FAILED",
			ReferenceID:  "N/A",
			ProviderName: p.Name(),
			Message:      "Airtel provider internal server error (simulated 500)",
		}
		// Return both the structured response AND a Go error to trip the Circuit Breaker
		return res, fmt.Errorf("provider failure: %s", res.Message)
	}

	// 2. Simulate Success
	return &PaymentResponse{
		Status:       "SUCCESS",
		ReferenceID:  fmt.Sprintf("AIRTEL-%d", time.Now().UnixNano()),
		ProviderName: p.Name(),
		IsIdempotent: false,
		Message:      "Transaction processed successfully via Airtel.",
	}, nil
}
