package helpers

import (
	"encoding/json"
	"reflect"
	"runtime"
	"strings"

	"github.com/allegro/bigcache/v3"
)

type ScrapeConfiguration struct {
	WatchedSecrets    []WatchedResource `yaml:"watched-secrets"`
	WatchedConfigMaps []WatchedResource `yaml:"watched-configmaps"`
}
type WatchedResource struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

func LogMsg(args ...string) string {
	return strings.Join(args, "")
}

func GetFn(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

// Helper function to serialize array and set it to cache
func SetArray(cache *bigcache.BigCache, key string, arr []string) error {
	// Serialize the array to JSON
	data, err := json.Marshal(arr)
	if err != nil {
		return err
	}
	// Set serialized data to cache
	return cache.Set(key, data)
}

// Helper function to get and deserialize the array from cache
func GetArray(cache *bigcache.BigCache, key string) ([]string, error) {
	// Get the data from cache
	entry, err := cache.Get(key)
	if err != nil {
		return nil, err
	}
	// Deserialize the data back to array
	var arr []string
	err = json.Unmarshal(entry, &arr)
	if err != nil {
		return nil, err
	}
	return arr, nil
}

// Add an item to the cached array
func AddToArray(cache *bigcache.BigCache, key string, value string) error {
	// Get the current array
	arr, err := GetArray(cache, key)
	if err != nil {
		// Initialize empty array if key doesn't exist
		arr = []string{}
	}
	// Append new value to array
	arr = append(arr, value)
	// Save updated array back to cache
	return SetArray(cache, key, arr)
}

// Remove the last item from the cached array
func PopFromArray(cache *bigcache.BigCache, key string) error {
	// Get the current array
	arr, err := GetArray(cache, key)
	if err != nil {
		return err
	}
	// Ensure the array is not empty before popping
	if len(arr) > 0 {
		arr = arr[:len(arr)-1] // Remove the last element
	}
	// Save the updated array back to cache
	return SetArray(cache, key, arr)
}
