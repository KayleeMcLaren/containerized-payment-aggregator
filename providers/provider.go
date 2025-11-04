package providers

import (
	"context"
)

// PaymentRequest contains the necessary data for a transaction.
type PaymentRequest struct {
	TransactionID string
	Amount        float64
	Currency      string
	ProviderKey   string // e.g., 'MTN-12345'
}

// PaymentResponse holds the result of a transaction.
type PaymentResponse struct {
	Status        string // "SUCCESS", "FAILED", "TIMEOUT"
	ReferenceID   string
	ProviderName  string
	IsIdempotent  bool
	Message       string
}

// PaymentProvider defines the interface for all external payment integrations (Adapter Pattern).
type PaymentProvider interface {
	Name() string
	ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
}