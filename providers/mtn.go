package providers

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type MTNProvider struct{}

func NewMTNProvider() *MTNProvider {
    // FIX: Seed the random number generator only once when the provider is created.
    // This ensures that the failure logic is truly random on each server run.
	rand.Seed(time.Now().UnixNano()) 
	return &MTNProvider{}
}

func (p *MTNProvider) Name() string {
	return "MTN_MOMO"
}

// ProcessPayment simulates interaction with the MTN MoMo API.
func (p *MTNProvider) ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	// Simulate Network Latency (200ms to 800ms)
	delay := time.Duration(rand.Intn(600)+200) * time.Millisecond
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // Handle context cancellation (timeout)
	case <-time.After(delay):
		// Continue
	}

	// 1. Simulate external API Errors (e.g., 10% chance of 500 server error)
	// rand.Float64() returns a float between 0.0 and 1.0. If it's < 0.10, it fails.
	if rand.Float64() < 0.10 {
		return &PaymentResponse{
			Status:       "FAILED",
			ReferenceID:  "N/A",
			ProviderName: p.Name(),
			Message:      "Provider internal server error (simulated 500)",
		}, nil
	}

	// 2. Simulate Success
	return &PaymentResponse{
		Status:        "SUCCESS",
		ReferenceID:   fmt.Sprintf("MTN-%d", time.Now().UnixNano()),
		ProviderName:  p.Name(),
		IsIdempotent:  false,
		Message:       "Transaction processed successfully.",
	}, nil
}