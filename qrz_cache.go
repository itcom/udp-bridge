package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const qrzCacheFile = "qrz_cache.json"

type qrzCacheEntry struct {
	Data      *qrzCall  `json:"data"`
	FetchedAt time.Time `json:"fetched_at"`
}

type qrzCache struct {
	mu   sync.RWMutex
	data map[string]*qrzCacheEntry
	ttl  time.Duration
}

func qrzCachePath() string {
	return filepath.Join(appDataDir(), qrzCacheFile)
}

// newQRZCache returns a new qrzCache with the given time-to-live (TTL).
// The returned cache is initialized with an empty data map and the given TTL.
// It also loads any existing cache data from a file named "qrz_cache.json" into the cache.
// If there is an error reading the file, or unmarshaling the JSON, it does not return an error.
func newQRZCache(ttl time.Duration) *qrzCache {
	c := &qrzCache{
		data: make(map[string]*qrzCacheEntry),
		ttl:  ttl,
	}
	c.load()
	return c
}

// load loads the QRZ cache from a file named "qrz_cache.json".
// It reads the file and unmarshals the JSON data into the cache's data map.
// If there is an error reading the file, it returns immediately.
// If there is an error unmarshaling the JSON, it also returns immediately.
func (c *qrzCache) load() {
	b, err := os.ReadFile(qrzCachePath())
	if err != nil {
		return
	}
	_ = json.Unmarshal(b, &c.data)
}

// save saves the cache to a file named "qrz_cache.json".
// It marshals the cache's data into JSON and writes it to the file with permissions 0644.
// The function locks the cache for writing while it saves the data.
func (c *qrzCache) save() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	b, _ := json.MarshalIndent(c.data, "", "  ")
	_ = os.WriteFile(qrzCachePath(), b, 0644)
}

// get retrieves the QRZ data for the given call from the cache.
// If the data is not found in the cache, it returns nil and false.
// If the data is found in the cache but is older than the cache's TTL, it is deleted from the cache and returns nil and false.
// Otherwise, it returns the QRZ data and true.
func (c *qrzCache) get(call string) (*qrzCall, bool) {
	key := strings.ToUpper(call)

	c.mu.RLock()
	entry, ok := c.data[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Since(entry.FetchedAt) > c.ttl {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		return nil, false
	}

	return entry.Data, true
}

// Set sets the QRZ data for the given call in the cache.
// If the call already exists in the cache, the existing data is overwritten.
// The cache is then saved to disk.
// The call is case-insensitive: "ABC" and "abc" are treated as the same call.
func (c *qrzCache) set(call string, data *qrzCall) {
	key := strings.ToUpper(call)

	c.mu.Lock()
	c.data[key] = &qrzCacheEntry{
		Data:      data,
		FetchedAt: time.Now(),
	}
	c.mu.Unlock()

	c.save()
}
