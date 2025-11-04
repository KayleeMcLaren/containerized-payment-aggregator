package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"payment-gateway-aggregator/providers" // Import our providers package
	"time"
)

// Aggregator holds references to all providers
type Aggregator struct {
	Providers map[string]providers.PaymentProvider
}

// newAggregator initializes the service with all providers.
func newAggregator() *Aggregator {
	return &Aggregator{
		Providers: map[string]providers.PaymentProvider{
			"MTN": providers.NewMTNProvider(),
			// Add other providers here later (e.g., "AIRTEL": providers.NewAirtelProvider(),)
		},
	}
}

// PayHandler processes the API request.
func (a *Aggregator) PayHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method Not Allowed"})
		return
	}

	var req providers.PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid Request Body"})
		return
	}
	
	// --- Input Validation and Routing ---
	providerName := "MTN" // Simple routing for now
	provider, ok := a.Providers[providerName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Provider %s not found", providerName)})
		return
	}

	// Set a 1-second timeout for the external provider call
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()
	
	log.Printf("Starting transaction %s via %s", req.TransactionID, provider.Name())

	// Call the provider interface
	res, err := provider.ProcessPayment(ctx, req)
	
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Provider Error: %v", err)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Processing error: %v", err)})
		return
	}

	// Send the response back to the client
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

func main() {
	aggregator := newAggregator()

	// Define routes
	http.HandleFunc("/v1/pay", aggregator.PayHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s...", port)
	
	// Start the server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}