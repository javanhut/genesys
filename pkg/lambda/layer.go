package lambda

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Layer represents a Lambda layer
type Layer struct {
	Name               string
	Description        string
	Version            int
	CompatibleRuntimes []string
	Size               int64
	SHA256             string
	CreatedAt          time.Time
	LayerArn           string
	LayerVersionArn    string
}

// LayerBuilder handles Lambda layer creation
type LayerBuilder struct {
	name        string
	description string
	runtime     *Runtime
	sourcePath  string
	cachePath   string
}

// NewLayerBuilder creates a new layer builder
func NewLayerBuilder(name, description string, runtime *Runtime, sourcePath string) *LayerBuilder {
	// Default cache path
	cachePath := filepath.Join(os.TempDir(), "genesys-lambda-layers")
	os.MkdirAll(cachePath, 0755)

	return &LayerBuilder{
		name:        name,
		description: description,
		runtime:     runtime,
		sourcePath:  sourcePath,
		cachePath:   cachePath,
	}
}

// Build creates a Lambda layer
func (lb *LayerBuilder) Build() (*Layer, string, error) {
	// Check if we need to build (dependency file changed)
	needsBuild, hash := lb.checkIfBuildNeeded()

	layerPath := filepath.Join(lb.cachePath, fmt.Sprintf("%s-%s-%s.zip", lb.name, lb.runtime.Name, hash[:8]))

	if !needsBuild && fileExists(layerPath) {
		// Use cached layer
		fmt.Printf("Using cached layer: %s\n", layerPath)
		return lb.createLayerFromZip(layerPath)
	}

	// Build new layer
	fmt.Printf("Building new layer for %s runtime...\n", lb.runtime.Name)

	// Use Podman builder
	builder := NewPodmanBuilder(lb.runtime, lb.sourcePath, layerPath)
	builder.SetDebug(true)

	if err := builder.BuildLayer(); err != nil {
		return nil, "", fmt.Errorf("failed to build layer: %w", err)
	}

	return lb.createLayerFromZip(layerPath)
}

// checkIfBuildNeeded checks if dependencies have changed
func (lb *LayerBuilder) checkIfBuildNeeded() (bool, string) {
	// Calculate hash of dependency files
	hash := sha256.New()

	for _, depFile := range lb.runtime.DependencyFiles {
		path := filepath.Join(lb.sourcePath, depFile)
		if file, err := os.Open(path); err == nil {
			io.Copy(hash, file)
			file.Close()
		}
	}

	hashSum := fmt.Sprintf("%x", hash.Sum(nil))

	// Check if cached layer exists
	cachedLayerPath := filepath.Join(lb.cachePath, fmt.Sprintf("%s-%s-%s.zip", lb.name, lb.runtime.Name, hashSum[:8]))
	if !fileExists(cachedLayerPath) {
		return true, hashSum
	}

	// Check if dependency files are newer than cached layer
	cachedInfo, _ := os.Stat(cachedLayerPath)
	for _, depFile := range lb.runtime.DependencyFiles {
		path := filepath.Join(lb.sourcePath, depFile)
		if info, err := os.Stat(path); err == nil {
			if info.ModTime().After(cachedInfo.ModTime()) {
				return true, hashSum
			}
		}
	}

	return false, hashSum
}

// createLayerFromZip creates a Layer object from a ZIP file
func (lb *LayerBuilder) createLayerFromZip(zipPath string) (*Layer, string, error) {
	// Get file info
	info, err := os.Stat(zipPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to stat layer zip: %w", err)
	}

	// Calculate SHA256
	file, err := os.Open(zipPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open layer zip: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, "", fmt.Errorf("failed to calculate SHA256: %w", err)
	}

	layer := &Layer{
		Name:               lb.name,
		Description:        lb.description,
		CompatibleRuntimes: []string{lb.runtime.Name},
		Size:               info.Size(),
		SHA256:             fmt.Sprintf("%x", hash.Sum(nil)),
		CreatedAt:          time.Now(),
	}

	return layer, zipPath, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// LayerCache manages cached layers
type LayerCache struct {
	cachePath string
	maxAge    time.Duration
}

// NewLayerCache creates a new layer cache
func NewLayerCache(cachePath string) *LayerCache {
	return &LayerCache{
		cachePath: cachePath,
		maxAge:    7 * 24 * time.Hour, // 7 days
	}
}

// CleanOldLayers removes old cached layers
func (lc *LayerCache) CleanOldLayers() error {
	now := time.Now()

	return filepath.Walk(lc.cachePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".zip" {
			if now.Sub(info.ModTime()) > lc.maxAge {
				os.Remove(path)
			}
		}

		return nil
	})
}

// GetCachedLayer retrieves a cached layer if it exists
func (lc *LayerCache) GetCachedLayer(name, runtime, hash string) (string, bool) {
	pattern := fmt.Sprintf("%s-%s-%s.zip", name, runtime, hash[:8])
	path := filepath.Join(lc.cachePath, pattern)

	if fileExists(path) {
		return path, true
	}

	return "", false
}
