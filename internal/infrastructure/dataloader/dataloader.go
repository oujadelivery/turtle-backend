package dataloader

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// GENERIC DATALOADER
// Production-grade implementation with batching and caching
// ============================================================================

// DataLoader is a generic data loading utility that provides:
// 1. Batching: Multiple Load() calls are batched into a single fetch
// 2. Caching: Results are cached for the duration of the request
// 3. Deduplication: Duplicate keys in a batch are only fetched once
type DataLoader[K comparable, V any] struct {
	// BatchFn is the function that loads data for a batch of keys
	batchFn BatchFunc[K, V]

	// Wait is the time to wait before dispatching a batch
	wait time.Duration

	// MaxBatch is the maximum number of keys per batch (0 = no limit)
	maxBatch int

	// Cache stores loaded values (per-request cache)
	cache map[K]*result[V]
	mu    sync.RWMutex

	// Batch channels
	batch chan *batchRequest[K, V]

	// Stats for monitoring
	stats *LoaderStats
}

// BatchFunc is a function that loads values for a batch of keys
// It should return a map of key -> value
// Missing keys should not be included in the map (will result in NotFound error)
type BatchFunc[K comparable, V any] func(ctx context.Context, keys []K) (map[K]V, error)

// result holds the result of a load operation
type result[V any] struct {
	value V
	err   error
}

// batchRequest represents a single load request
type batchRequest[K comparable, V any] struct {
	key     K
	channel chan *result[V]
}

// LoaderStats tracks DataLoader performance metrics
type LoaderStats struct {
	// Total number of Load() calls
	TotalLoads int64

	// Number of cache hits
	CacheHits int64

	// Number of batches dispatched
	BatchCount int64

	// Total keys loaded from database
	KeysLoaded int64

	mu sync.RWMutex
}

// NewDataLoader creates a new DataLoader instance
func NewDataLoader[K comparable, V any](batchFn BatchFunc[K, V], wait time.Duration, maxBatch int) *DataLoader[K, V] {
	loader := &DataLoader[K, V]{
		batchFn:  batchFn,
		wait:     wait,
		maxBatch: maxBatch,
		cache:    make(map[K]*result[V]),
		batch:    make(chan *batchRequest[K, V], 1000),
		stats:    &LoaderStats{},
	}

	// Start batch processor
	go loader.batchProcessor()

	return loader
}

// Load loads a single value by key
// Multiple concurrent calls are automatically batched
func (l *DataLoader[K, V]) Load(ctx context.Context, key K) (V, error) {
	// Update stats
	l.stats.mu.Lock()
	l.stats.TotalLoads++
	l.stats.mu.Unlock()

	// Check cache first
	l.mu.RLock()
	if cached, ok := l.cache[key]; ok {
		l.mu.RUnlock()

		// Cache hit
		l.stats.mu.Lock()
		l.stats.CacheHits++
		l.stats.mu.Unlock()

		return cached.value, cached.err
	}
	l.mu.RUnlock()

	// Create response channel
	resultCh := make(chan *result[V], 1)

	// Send load request to batch
	select {
	case l.batch <- &batchRequest[K, V]{
		key:     key,
		channel: resultCh,
	}:
	case <-ctx.Done():
		var zero V
		return zero, ctx.Err()
	}

	// Wait for result
	select {
	case res := <-resultCh:
		return res.value, res.err
	case <-ctx.Done():
		var zero V
		return zero, ctx.Err()
	}
}

// LoadMany loads multiple values by keys
// This is more efficient than calling Load() multiple times
func (l *DataLoader[K, V]) LoadMany(ctx context.Context, keys []K) ([]V, error) {
	results := make([]V, len(keys))
	errors := make([]error, len(keys))

	// Use goroutines to load all keys concurrently
	var wg sync.WaitGroup
	for i, key := range keys {
		wg.Add(1)
		go func(index int, k K) {
			defer wg.Done()
			value, err := l.Load(ctx, k)
			results[index] = value
			errors[index] = err
		}(i, key)
	}

	wg.Wait()

	// Check if any errors occurred
	for _, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("batch load failed: %w", err)
		}
	}

	return results, nil
}

// Clear removes a key from the cache
func (l *DataLoader[K, V]) Clear(key K) {
	l.mu.Lock()
	delete(l.cache, key)
	l.mu.Unlock()
}

// ClearAll removes all keys from the cache
func (l *DataLoader[K, V]) ClearAll() {
	l.mu.Lock()
	l.cache = make(map[K]*result[V])
	l.mu.Unlock()
}

// Prime adds a value to the cache without loading it
// Useful for optimistically caching values
func (l *DataLoader[K, V]) Prime(key K, value V) {
	l.mu.Lock()
	l.cache[key] = &result[V]{value: value}
	l.mu.Unlock()
}

// Stats returns current loader statistics
func (l *DataLoader[K, V]) Stats() LoaderStats {
	l.stats.mu.RLock()
	defer l.stats.mu.RUnlock()

	return LoaderStats{
		TotalLoads: l.stats.TotalLoads,
		CacheHits:  l.stats.CacheHits,
		BatchCount: l.stats.BatchCount,
		KeysLoaded: l.stats.KeysLoaded,
	}
}

// batchProcessor processes batches in the background
func (l *DataLoader[K, V]) batchProcessor() {
	for {
		l.processBatch()
	}
}

// processBatch processes a single batch
func (l *DataLoader[K, V]) processBatch() {
	// Collect requests for a batch
	var requests []*batchRequest[K, V]

	// Wait for first request or timeout
	timer := time.NewTimer(l.wait)
	defer timer.Stop()

	select {
	case req := <-l.batch:
		requests = append(requests, req)
	case <-timer.C:
		return
	}

	// Collect more requests until timeout or max batch size
	timeout := time.After(l.wait)

	for {
		// Check if we've reached max batch size
		if l.maxBatch > 0 && len(requests) >= l.maxBatch {
			break
		}

		select {
		case req := <-l.batch:
			requests = append(requests, req)
		case <-timeout:
			goto dispatch
		}
	}

dispatch:
	if len(requests) == 0 {
		return
	}

	// Update stats
	l.stats.mu.Lock()
	l.stats.BatchCount++
	l.stats.mu.Unlock()

	// Extract unique keys (deduplication)
	keys := make([]K, 0, len(requests))
	keyIndex := make(map[K]int)

	for _, req := range requests {
		if _, exists := keyIndex[req.key]; !exists {
			keyIndex[req.key] = len(keys)
			keys = append(keys, req.key)
		}
	}

	// Update stats
	l.stats.mu.Lock()
	l.stats.KeysLoaded += int64(len(keys))
	l.stats.mu.Unlock()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call batch function
	results, err := l.batchFn(ctx, keys)

	// Process results
	for _, req := range requests {
		var res *result[V]

		if err != nil {
			// Batch error - all keys fail
			res = &result[V]{err: err}
		} else if value, ok := results[req.key]; ok {
			// Key found
			res = &result[V]{value: value}
		} else {
			// Key not found
			var zero V
			res = &result[V]{
				value: zero,
				err:   fmt.Errorf("key not found: %v", req.key),
			}
		}

		// Cache result
		l.mu.Lock()
		l.cache[req.key] = res
		l.mu.Unlock()

		// Send result to requester
		select {
		case req.channel <- res:
		default:
		}
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// CacheHitRate returns the cache hit rate as a percentage
func (s *LoaderStats) CacheHitRate() float64 {
	if s.TotalLoads == 0 {
		return 0
	}
	return float64(s.CacheHits) / float64(s.TotalLoads) * 100
}

// AverageBatchSize returns the average number of keys per batch
func (s *LoaderStats) AverageBatchSize() float64 {
	if s.BatchCount == 0 {
		return 0
	}
	return float64(s.KeysLoaded) / float64(s.BatchCount)
}

// String returns a formatted string of statistics
func (s *LoaderStats) String() string {
	return fmt.Sprintf(
		"Loads: %d, Cache Hits: %d (%.1f%%), Batches: %d, Avg Batch Size: %.1f",
		s.TotalLoads,
		s.CacheHits,
		s.CacheHitRate(),
		s.BatchCount,
		s.AverageBatchSize(),
	)
}
