package gh

import (
	"sync"
)

// APICallCounter tracks the number of GitHub API calls made
type APICallCounter struct {
	counts map[string]int
	mu     sync.Mutex
}

// Global instance of the counter
var Counter = NewAPICallCounter()

// NewAPICallCounter creates a new APICallCounter
func NewAPICallCounter() *APICallCounter {
	return &APICallCounter{
		counts: make(map[string]int),
	}
}

// Increment increments the counter for a specific API operation
func (c *APICallCounter) Increment(operation string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts[operation]++
}

// Reset resets all counters to zero
func (c *APICallCounter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts = make(map[string]int)
}

// GetCount returns the count for a specific operation
func (c *APICallCounter) GetCount(operation string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[operation]
}

// GetCounts returns a copy of all counts
func (c *APICallCounter) GetCounts() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make(map[string]int, len(c.counts))
	for k, v := range c.counts {
		result[k] = v
	}
	return result
}

// GetTotalCount returns the total number of API operations
func (c *APICallCounter) GetTotalCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	total := 0
	for _, count := range c.counts {
		total += count
	}
	return total
}
