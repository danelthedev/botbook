package main

import (
	"sync"
	"time"
)

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// TTLCache is a simple thread-safe in-memory cache with per-entry TTL.
type TTLCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

func newTTLCache() *TTLCache {
	c := &TTLCache{entries: make(map[string]cacheEntry)}
	go c.janitor()
	return c
}

// Get returns the cached value for key, or (nil, false) if missing or expired.
func (c *TTLCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

// Set stores value under key with the given TTL.
func (c *TTLCache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{value: value, expiresAt: time.Now().Add(ttl)}
}

// janitor periodically removes expired entries to free memory.
func (c *TTLCache) janitor() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}

// appCache is the process-wide cache instance shared across all handlers.
var appCache = newTTLCache()
