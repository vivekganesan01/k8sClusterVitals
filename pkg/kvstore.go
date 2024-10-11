package helpers

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/patrickmn/go-cache"
)

type KeyValueStore struct {
	cache   *bigcache.BigCache
	gocache *cache.Cache
	keys    map[string][]byte
	mu      sync.Mutex // Mutex for protecting access to keys
}

func NewKeyValueStore() *KeyValueStore {
	// todo: to see how best we can parameterise this
	cacheConfig := bigcache.Config{
		Shards:           1024,
		LifeWindow:       45 * time.Second,
		CleanWindow:      60 * time.Second,
		MaxEntrySize:     800,
		Verbose:          true,
		HardMaxCacheSize: 8192,
	}
	bigcache, _ := bigcache.New(context.Background(), cacheConfig)
	gocache := cache.New(5*time.Minute, 10*time.Minute)
	return &KeyValueStore{cache: bigcache, gocache: gocache, keys: make(map[string][]byte)}
}

func (kvs *KeyValueStore) GoCacheSet(key string, value interface{}) error {
	return kvs.gocache.Add(key, value, cache.NoExpiration)
}

func (kvs *KeyValueStore) GoCacheDelete(key string) {
	kvs.gocache.Delete(key)
}

func (kvs *KeyValueStore) GoCacheGetAll() interface{} {
	return kvs.gocache.Items()
}

func (kvs *KeyValueStore) GoCacheGet(key string, value interface{}) (interface{}, error) {
	v, ok := kvs.gocache.Get(key)
	if ok {
		return v, nil
	}
	return nil, errors.New("gocache key not found")
}

func (kvs *KeyValueStore) Set(key string, value []byte) error {
	kvs.mu.Lock()         // Lock the mutex before modifying keys
	defer kvs.mu.Unlock() // Ensure the mutex is unlocked after the function returns
	kvs.keys[key] = value
	return kvs.cache.Set(key, value)
}

func (kvs *KeyValueStore) Delete(key string) error {
	kvs.mu.Lock()         // Lock the mutex before modifying keys
	defer kvs.mu.Unlock() // Ensure the mutex is unlocked after the function returns
	delete(kvs.keys, key)
	return kvs.cache.Delete(key)
}

func (kvs *KeyValueStore) LenAll() int {
	kvs.mu.Lock()         // Lock the mutex for reading keys
	defer kvs.mu.Unlock() // Ensure the mutex is unlocked after reading
	return len(kvs.keys)
}

func (kvs *KeyValueStore) GetAll() (map[string]interface{}, error) {
	allValues := make(map[string]interface{})
	kvs.mu.Lock()         // Lock the mutex for reading keys
	defer kvs.mu.Unlock() // Ensure the mutex is unlocked after reading
	for key := range kvs.keys {
		value, err := kvs.cache.Get(key)
		if err == nil {
			allValues[key] = string(value)
		} else {
			delete(allValues, key) // Ensure the map has latest updates and remove the old ones
			// todo: best approach is to have a callback on key expiration to check and clean up the map but computation on map will be higher
			// for now lazy delete is good.
		}
	}
	return allValues, nil
}
