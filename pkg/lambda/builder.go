package lambda

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Builder orchestrates the Lambda function and layer build process
type Builder struct {
	sourcePath   string
	functionName string
	runtime      *Runtime
	handler      string
	environment  map[string]string
	tags         map[string]string
	memory       int
	timeout      int
}

// BuildConfig contains configuration for building Lambda functions
type BuildConfig struct {
	SourcePath   string
	FunctionName string
	Runtime      string
	Handler      string
	Environment  map[string]string
	Tags         map[string]string
	Memory       int
	Timeout      int
	BuildLayers  bool
}

// NewBuilder creates a new Lambda builder
func NewBuilder(config *BuildConfig) (*Builder, error) {
	// Validate source path
	if _, err := os.Stat(config.SourcePath); err != nil {
		return nil, fmt.Errorf("source path does not exist: %w", err)
	}

	// Get runtime
	runtime, err := GetRuntimeByName(config.Runtime)
	if err != nil {
		// Try to detect runtime
		detector := NewRuntimeDetector(config.SourcePath)
		runtime, err = detector.DetectRuntime()
		if err != nil {
			return nil, fmt.Errorf("failed to detect runtime: %w", err)
		}
	}

	// Detect handler if not provided
	handler := config.Handler
	if handler == "" {
		detector := NewRuntimeDetector(config.SourcePath)
		handler, err = detector.DetectHandler(runtime)
		if err != nil {
			return nil, fmt.Errorf("failed to detect handler: %w", err)
		}
	}

	// Set defaults
	if config.Memory == 0 {
		config.Memory = 512
	}
	if config.Timeout == 0 {
		config.Timeout = 30
	}

	return &Builder{
		sourcePath:   config.SourcePath,
		functionName: config.FunctionName,
		runtime:      runtime,
		handler:      handler,
		environment:  config.Environment,
		tags:         config.Tags,
		memory:       config.Memory,
		timeout:      config.Timeout,
	}, nil
}

// Build performs the complete build process
func (b *Builder) Build() (*BuildResult, error) {
	// Check Podman availability
	if err := CheckPodmanAvailable(); err != nil {
		return nil, err
	}

	result := &BuildResult{
		FunctionName: b.functionName,
		Runtime:      b.runtime.Name,
		Handler:      b.handler,
		Memory:       b.memory,
		Timeout:      b.timeout,
		Environment:  b.environment,
		Tags:         b.tags,
	}

	// Build layer if dependencies exist
	if b.hasDependencies() {
		fmt.Println("Building Lambda layer for dependencies...")
		layer, layerPath, err := b.buildLayer()
		if err != nil {
			return nil, fmt.Errorf("failed to build layer: %w", err)
		}
		result.Layer = layer
		result.LayerZipPath = layerPath
	}

	// Build function
	fmt.Println("Building Lambda function package...")
	functionPath, err := b.buildFunction()
	if err != nil {
		return nil, fmt.Errorf("failed to build function: %w", err)
	}
	result.FunctionZipPath = functionPath

	// Calculate sizes
	if info, err := os.Stat(functionPath); err == nil {
		result.FunctionSize = info.Size()
	}
	if result.LayerZipPath != "" {
		if info, err := os.Stat(result.LayerZipPath); err == nil {
			result.LayerSize = info.Size()
		}
	}

	return result, nil
}

// BuildResult contains the results of a build
type BuildResult struct {
	FunctionName    string
	Runtime         string
	Handler         string
	Memory          int
	Timeout         int
	Environment     map[string]string
	Tags            map[string]string
	FunctionZipPath string
	FunctionSize    int64
	Layer           *Layer
	LayerZipPath    string
	LayerSize       int64
}

// hasDependencies checks if the project has dependencies
func (b *Builder) hasDependencies() bool {
	for _, depFile := range b.runtime.DependencyFiles {
		if _, err := os.Stat(filepath.Join(b.sourcePath, depFile)); err == nil {
			return true
		}
	}
	return false
}

// buildLayer builds the Lambda layer
func (b *Builder) buildLayer() (*Layer, string, error) {
	layerName := fmt.Sprintf("%s-deps", b.functionName)
	layerDesc := fmt.Sprintf("Dependencies for %s Lambda function", b.functionName)

	builder := NewLayerBuilder(layerName, layerDesc, b.runtime, b.sourcePath)
	return builder.Build()
}

// buildFunction builds the Lambda function package
func (b *Builder) buildFunction() (string, error) {
	outputPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s-function.zip", b.functionName))

	// For compiled languages, use Podman builder
	if strings.HasPrefix(b.runtime.Name, "go") || strings.HasPrefix(b.runtime.Name, "java") {
		builder := NewPodmanBuilder(b.runtime, b.sourcePath, outputPath)
		builder.SetDebug(true)
		if err := builder.BuildFunction(); err != nil {
			return "", err
		}
		return outputPath, nil
	}

	// For interpreted languages, create simple ZIP
	return b.createSimpleFunctionZip(outputPath)
}

// createSimpleFunctionZip creates a ZIP file for interpreted languages
func (b *Builder) createSimpleFunctionZip(outputPath string) (string, error) {
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	writer := zip.NewWriter(zipFile)
	defer writer.Close()

	// Walk through source files
	err = filepath.Walk(b.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-source files
		if info.IsDir() {
			return nil
		}

		// Skip common non-source files and directories
		relPath, err := filepath.Rel(b.sourcePath, path)
		if err != nil {
			return err
		}

		// Skip dependency directories and build artifacts
		for _, skip := range []string{"node_modules", "venv", "__pycache__", ".git", "dist", "build"} {
			if strings.Contains(relPath, skip) {
				return nil
			}
		}

		// Skip dependency files (they go in the layer)
		for _, depFile := range b.runtime.DependencyFiles {
			if filepath.Base(path) == depFile {
				return nil
			}
		}

		// Create zip entry
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		// Copy file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return "", fmt.Errorf("failed to create function zip: %w", err)
	}

	return outputPath, nil
}

// GetBuildInfo returns information about what will be built
func (b *Builder) GetBuildInfo() string {
	info := fmt.Sprintf("Lambda Function Build Info:\n")
	info += fmt.Sprintf("  Function Name: %s\n", b.functionName)
	info += fmt.Sprintf("  Runtime: %s\n", b.runtime.Name)
	info += fmt.Sprintf("  Handler: %s\n", b.handler)
	info += fmt.Sprintf("  Memory: %d MB\n", b.memory)
	info += fmt.Sprintf("  Timeout: %d seconds\n", b.timeout)

	if b.hasDependencies() {
		info += fmt.Sprintf("  Dependencies: Yes (will create layer)\n")
		for _, depFile := range b.runtime.DependencyFiles {
			if _, err := os.Stat(filepath.Join(b.sourcePath, depFile)); err == nil {
				info += fmt.Sprintf("    - %s\n", depFile)
			}
		}
	} else {
		info += fmt.Sprintf("  Dependencies: No\n")
	}

	if len(b.environment) > 0 {
		info += fmt.Sprintf("  Environment Variables:\n")
		for k, v := range b.environment {
			info += fmt.Sprintf("    - %s: %s\n", k, v)
		}
	}

	if len(b.tags) > 0 {
		info += fmt.Sprintf("  Tags:\n")
		for k, v := range b.tags {
			info += fmt.Sprintf("    - %s: %s\n", k, v)
		}
	}

	return info
}
