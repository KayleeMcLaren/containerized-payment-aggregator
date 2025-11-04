package providers

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

func init() {
	// Use the init function to seed the global generator exactly ONCE upon package load
	rand.Seed(time.Now().UnixNano())
}

type MTNProvider struct{}

func NewMTNProvider() *MTNProvider {
	// FIX: Seed the random number generator only once when the provider is created.
	// This ensures that the failure logic is truly random on each server run.
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

	// 1. Simulate external API Errors (80% chance of 500 server error)
	if rand.Float64() < 0.80 {
		// Create the response object
		res := &PaymentResponse{
			Status:       "FAILED",
			ReferenceID:  "N/A",
			ProviderName: p.Name(),
			Message:      "Provider internal server error (simulated 500)",
		}
		// RETURN BOTH THE RESPONSE AND A NEW ERROR OBJECT
		return res, fmt.Errorf("provider failure: %s", res.Message) // <-- CRITICAL CHANGE HERE
	}

	// 2. Simulate Success
	return &PaymentResponse{
		Status:       "SUCCESS",
		ReferenceID:  fmt.Sprintf("MTN-%d", time.Now().UnixNano()),
		ProviderName: p.Name(),
		IsIdempotent: false,
		Message:      "Transaction processed successfully.",
	}, nil // Success returns nil error
}
