// /home/kaylee-dev/Desktop/payment-gateway-aggregator/main.go (Final Phase II Version)

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"payment-gateway-aggregator/cache"
	"payment-gateway-aggregator/providers"
	"time"

	"github.com/sony/gobreaker" // NEW IMPORT
)

// Aggregator now holds references to providers, the store, and the circuit breakers
type Aggregator struct {
	Providers map[string]providers.PaymentProvider
	Store     cache.IdempotencyStore
	Breakers  map[string]*gobreaker.CircuitBreaker // NEW FIELD: Map of breakers
}

// newAggregator initializes the service with all providers, cache, and circuit breakers.
func newAggregator() *Aggregator {
	// 1. Initialize Redis Store
	redisStore := cache.NewRedisStore("localhost:6379", "", 0)

	// 2. Define Circuit Breaker Settings (Using ReadyToTrip for failure rate logic)
	settings := gobreaker.Settings{
		Name: "MTN-Breaker",
		// The maximum number of requests allowed in the half-open state.
		// Setting to 1 allows one trial request after the Timeout expires.
		MaxRequests: 1,
		// The period of the open state (the delay before the circuit tries to close)
		Timeout: 30 * time.Second,
		// The rolling window size to clear counts
		Interval: 5 * time.Second,

		// THIS IS THE CORRECT FIELD: Determines when to open the circuit (Closed -> Open).
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Ensure we have a minimum number of requests (e.g., 3) to start calculating the ratio
			if counts.Requests < 3 {
				return false
			}

			// Calculate the failure ratio using TotalFailures since the last clear/reset
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)

			// Return true (OPEN the circuit) if the failure ratio is 60% or higher
			return failureRatio >= 0.6
		},

		// This function defines what an error means. Any non-nil error from ProcessPayment is a failure.
		IsSuccessful: func(err error) bool {
			return err == nil
		},
	}

	// 3. Initialize Breaker and Aggregator
	breakerMTN := gobreaker.NewCircuitBreaker(settings)

	return &Aggregator{
		Providers: map[string]providers.PaymentProvider{
			"MTN": providers.NewMTNProvider(),
		},
		Store: redisStore,
		Breakers: map[string]*gobreaker.CircuitBreaker{ // ASSIGN BREAKER
			"MTN": breakerMTN,
		},
	}
}

// PayHandler processes the API request, now with Idempotency and Circuit Breaker logic.
func (a *Aggregator) PayHandler(w http.ResponseWriter, r *http.Request) {
	// ... (Initial setup, method check, and request decoding remain the same) ...
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" { // (Keep this)
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method Not Allowed"})
		return
	}

	var req providers.PaymentRequest                             // (Keep this)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { // (Keep this)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid Request Body"})
		return
	}

	// --- IDEMPOTENCY CHECK --- (Keep this section)
	isDuplicate, err := a.Store.CheckOrSetInProgress(r.Context(), req.TransactionID)
	if err != nil && err.Error() == "transaction already in progress" {
		w.WriteHeader(http.StatusTooEarly)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Duplicate transaction ID detected",
			"message": "A transaction with this ID is currently being processed. Please wait.",
		})
		return
	}
	if isDuplicate {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Duplicate transaction ID detected",
			"message": "This transaction ID has already been successfully completed.",
		})
		return
	}
	// --- IDEMPOTENCY CHECK END ---

	// --- Provider Routing & Circuit Breaker Lookup ---
	providerName := "MTN"
	provider, ok := a.Providers[providerName]
	if !ok {
		// ... (Error handling remains the same) ...
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Provider %s not found", providerName)})
		return
	}

	breaker, ok := a.Breakers[providerName]
	if !ok {
		// Fallback for providers without a defined breaker (shouldn't happen here)
		log.Printf("Warning: No circuit breaker found for %s", providerName)
	}

	// Set a 1-second timeout for the external provider call
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	log.Printf("Starting transaction %s via %s", req.TransactionID, provider.Name())

	// --- CIRCUIT BREAKER EXECUTION ---
	// The Execute function handles the core CB logic:
	// 1. Checks if the circuit is Open (fails immediately with gobreaker.ErrOpenState).
	// 2. If Closed, runs the request function.
	// 3. If Half-Open, permits a trial request.
	result, errCB := breaker.Execute(func() (interface{}, error) {
		// The actual provider call happens inside the circuit breaker wrapper
		return provider.ProcessPayment(ctx, req)
	})

	// Check if the error came from the Circuit Breaker itself (circuit is OPEN)
	if errCB == gobreaker.ErrOpenState {
		w.WriteHeader(http.StatusServiceUnavailable) // 503 is standard for CB open
		log.Printf("Circuit Breaker OPEN for %s. Bypassing request.", provider.Name())
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Service Unavailable",
			"message": fmt.Sprintf("Provider %s is currently experiencing high failure rates and has been temporarily taken offline.", provider.Name()),
		})
		return
	}

	// Check for other errors (timeout or provider internal error)
	if errCB != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Provider/CB Error: %v", errCB)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Processing error: %v", errCB)})
		return
	}

	// Cast the result back to the expected type
	res := result.(*providers.PaymentResponse)

	// --- IDEMPOTENCY COMPLETION --- (Keep this section)
	if res.Status == "SUCCESS" {
		if err := a.Store.SetCompleted(r.Context(), req.TransactionID); err != nil {
			log.Printf("Warning: Failed to set transaction %s as COMPLETED in Redis: %v", req.TransactionID, err)
		}
		res.IsIdempotent = true
	}
	// --- IDEMPOTENCY COMPLETION END ---

	// Send the response back to the client
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

func main() {
	aggregator := newAggregator()
	// ... (The rest of main() remains the same) ...
	http.HandleFunc("/v1/pay", aggregator.PayHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s...", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
