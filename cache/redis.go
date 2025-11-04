package cache

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

// Define status constants for transactions stored in Redis
const (
    StatusInProgress = "IN_PROGRESS"
    StatusCompleted  = "COMPLETED"
    // Use a short, meaningful expiration for the "IN_PROGRESS" key
    InProgressExpiry = 10 * time.Second 
    // Use a long, meaningful expiry for the "COMPLETED" key
    CompletedExpiry  = 24 * time.Hour 
)

// IdempotencyStore interface defines the required methods for our cache layer.
type IdempotencyStore interface {
    CheckOrSetInProgress(ctx context.Context, transactionID string) (bool, error)
    SetCompleted(ctx context.Context, transactionID string) error
    CheckCompleted(ctx context.Context, transactionID string) (bool, error)
}

// RedisStore implements the IdempotencyStore interface.
type RedisStore struct {
    client *redis.Client
}

// NewRedisStore creates a new Redis client instance.
func NewRedisStore(addr string, password string, db int) *RedisStore {
    rdb := redis.NewClient(&redis.Options{
        Addr:     addr,     // e.g., "localhost:6379"
        Password: password, // no password set
        DB:       db,       // use default DB
    })

    // In a production environment, you would add logic here to check the connection (rdb.Ping).
    
    return &RedisStore{
        client: rdb,
    }
}

// CheckOrSetInProgress checks if a transaction is already COMPLETED or sets it to IN_PROGRESS.
// Returns (true, nil) if the transaction is a duplicate (COMPLETED or IN_PROGRESS by another call).
// Returns (false, nil) if the transaction is new and is now marked as IN_PROGRESS.
// The IN_PROGRESS state uses a short timeout (10s) to prevent deadlocks if the server crashes.
func (r *RedisStore) CheckOrSetInProgress(ctx context.Context, transactionID string) (bool, error) {
    key := fmt.Sprintf("txn:%s", transactionID)

    // Check if the transaction is already COMPLETED
    completedStatus, err := r.client.Get(ctx, key).Result()
    if err == nil && completedStatus == StatusCompleted {
        // Already completed, this is a duplicate request
        return true, nil
    }

    // Try to set the key to IN_PROGRESS using SET NX (Set if Not eXists)
    // This atomically checks and sets the value, which is crucial for concurrency.
    set, err := r.client.SetNX(ctx, key, StatusInProgress, InProgressExpiry).Result()
    if err != nil {
        return false, fmt.Errorf("redis SETNX error: %w", err)
    }

    if !set {
        // The key already existed (it was IN_PROGRESS by another goroutine/call)
        return true, errors.New("transaction already in progress")
    }

    // Key was successfully set, this is a new, valid transaction
    return false, nil
}

// SetCompleted sets the transaction status to COMPLETED with a long expiry.
func (r *RedisStore) SetCompleted(ctx context.Context, transactionID string) error {
    key := fmt.Sprintf("txn:%s", transactionID)
    return r.client.Set(ctx, key, StatusCompleted, CompletedExpiry).Err()
}

// CheckCompleted checks if a transaction is already set to COMPLETED.
func (r *RedisStore) CheckCompleted(ctx context.Context, transactionID string) (bool, error) {
    key := fmt.Sprintf("txn:%s", transactionID)
    status, err := r.client.Get(ctx, key).Result()

    if err == redis.Nil {
        return false, nil // Key not found (not completed)
    }
    if err != nil {
        return false, fmt.Errorf("redis GET error: %w", err)
    }
    
    return status == StatusCompleted, nil
}