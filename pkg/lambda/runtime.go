package lambda

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Runtime represents a Lambda runtime configuration
type Runtime struct {
	Name            string
	Version         string
	Handler         string
	BuildImage      string
	LayerPath       string
	FileExtensions  []string
	DependencyFiles []string
	Architecture    string
	Description     string
}

// SupportedRuntimes contains all supported Lambda runtimes
var SupportedRuntimes = map[string]*Runtime{
	"python3.8": {
		Name:            "python3.8",
		Version:         "3.8",
		BuildImage:      "public.ecr.aws/lambda/python:3.8",
		LayerPath:       "/opt/python",
		FileExtensions:  []string{".py"},
		DependencyFiles: []string{"requirements.txt", "Pipfile", "poetry.lock", "pyproject.toml"},
		Architecture:    "x86_64",
		Description:     "Python 3.8 (x86_64)",
	},
	"python3.9": {
		Name:            "python3.9",
		Version:         "3.9",
		BuildImage:      "public.ecr.aws/lambda/python:3.9",
		LayerPath:       "/opt/python",
		FileExtensions:  []string{".py"},
		DependencyFiles: []string{"requirements.txt", "Pipfile", "poetry.lock", "pyproject.toml"},
		Architecture:    "x86_64",
		Description:     "Python 3.9 (x86_64)",
	},
	"python3.10": {
		Name:            "python3.10",
		Version:         "3.10",
		BuildImage:      "public.ecr.aws/lambda/python:3.10",
		LayerPath:       "/opt/python",
		FileExtensions:  []string{".py"},
		DependencyFiles: []string{"requirements.txt", "Pipfile", "poetry.lock", "pyproject.toml"},
		Architecture:    "x86_64",
		Description:     "Python 3.10 (x86_64)",
	},
	"python3.11": {
		Name:            "python3.11",
		Version:         "3.11",
		BuildImage:      "public.ecr.aws/lambda/python:3.11",
		LayerPath:       "/opt/python",
		FileExtensions:  []string{".py"},
		DependencyFiles: []string{"requirements.txt", "Pipfile", "poetry.lock", "pyproject.toml"},
		Architecture:    "x86_64",
		Description:     "Python 3.11 (x86_64)",
	},
	"python3.12": {
		Name:            "python3.12",
		Version:         "3.12",
		BuildImage:      "public.ecr.aws/lambda/python:3.12",
		LayerPath:       "/opt/python",
		FileExtensions:  []string{".py"},
		DependencyFiles: []string{"requirements.txt", "Pipfile", "poetry.lock", "pyproject.toml"},
		Architecture:    "x86_64",
		Description:     "Python 3.12 (x86_64)",
	},
	"python3.13": {
		Name:            "python3.13",
		Version:         "3.13",
		BuildImage:      "public.ecr.aws/lambda/python:3.13",
		LayerPath:       "/opt/python",
		FileExtensions:  []string{".py"},
		DependencyFiles: []string{"requirements.txt", "Pipfile", "poetry.lock", "pyproject.toml"},
		Architecture:    "x86_64",
		Description:     "Python 3.13 (x86_64)",
	},
	// ARM64 Python variants
	"python3.9-arm64": {
		Name:            "python3.9",
		Version:         "3.9",
		BuildImage:      "public.ecr.aws/lambda/python:3.9-arm64",
		LayerPath:       "/opt/python",
		FileExtensions:  []string{".py"},
		DependencyFiles: []string{"requirements.txt", "Pipfile", "poetry.lock", "pyproject.toml"},
		Architecture:    "arm64",
		Description:     "Python 3.9 (arm64)",
	},
	"python3.10-arm64": {
		Name:            "python3.10",
		Version:         "3.10",
		BuildImage:      "public.ecr.aws/lambda/python:3.10-arm64",
		LayerPath:       "/opt/python",
		FileExtensions:  []string{".py"},
		DependencyFiles: []string{"requirements.txt", "Pipfile", "poetry.lock", "pyproject.toml"},
		Architecture:    "arm64",
		Description:     "Python 3.10 (arm64)",
	},
	"python3.11-arm64": {
		Name:            "python3.11",
		Version:         "3.11",
		BuildImage:      "public.ecr.aws/lambda/python:3.11-arm64",
		LayerPath:       "/opt/python",
		FileExtensions:  []string{".py"},
		DependencyFiles: []string{"requirements.txt", "Pipfile", "poetry.lock", "pyproject.toml"},
		Architecture:    "arm64",
		Description:     "Python 3.11 (arm64)",
	},
	"python3.12-arm64": {
		Name:            "python3.12",
		Version:         "3.12",
		BuildImage:      "public.ecr.aws/lambda/python:3.12-arm64",
		LayerPath:       "/opt/python",
		FileExtensions:  []string{".py"},
		DependencyFiles: []string{"requirements.txt", "Pipfile", "poetry.lock", "pyproject.toml"},
		Architecture:    "arm64",
		Description:     "Python 3.12 (arm64)",
	},
	"python3.13-arm64": {
		Name:            "python3.13",
		Version:         "3.13",
		BuildImage:      "public.ecr.aws/lambda/python:3.13-arm64",
		LayerPath:       "/opt/python",
		FileExtensions:  []string{".py"},
		DependencyFiles: []string{"requirements.txt", "Pipfile", "poetry.lock", "pyproject.toml"},
		Architecture:    "arm64",
		Description:     "Python 3.13 (arm64)",
	},
	"nodejs18.x": {
		Name:            "nodejs18.x",
		Version:         "18",
		BuildImage:      "public.ecr.aws/lambda/nodejs:18",
		LayerPath:       "/opt/nodejs",
		FileExtensions:  []string{".js", ".mjs", ".ts"},
		DependencyFiles: []string{"package.json", "yarn.lock", "pnpm-lock.yaml"},
		Architecture:    "x86_64",
		Description:     "Node.js 18.x (x86_64)",
	},
	"nodejs20.x": {
		Name:            "nodejs20.x",
		Version:         "20",
		BuildImage:      "public.ecr.aws/lambda/nodejs:20",
		LayerPath:       "/opt/nodejs",
		FileExtensions:  []string{".js", ".mjs", ".ts"},
		DependencyFiles: []string{"package.json", "yarn.lock", "pnpm-lock.yaml"},
		Architecture:    "x86_64",
		Description:     "Node.js 20.x (x86_64)",
	},
	"nodejs18.x-arm64": {
		Name:            "nodejs18.x",
		Version:         "18",
		BuildImage:      "public.ecr.aws/lambda/nodejs:18-arm64",
		LayerPath:       "/opt/nodejs",
		FileExtensions:  []string{".js", ".mjs", ".ts"},
		DependencyFiles: []string{"package.json", "yarn.lock", "pnpm-lock.yaml"},
		Architecture:    "arm64",
		Description:     "Node.js 18.x (arm64)",
	},
	"nodejs20.x-arm64": {
		Name:            "nodejs20.x",
		Version:         "20",
		BuildImage:      "public.ecr.aws/lambda/nodejs:20-arm64",
		LayerPath:       "/opt/nodejs",
		FileExtensions:  []string{".js", ".mjs", ".ts"},
		DependencyFiles: []string{"package.json", "yarn.lock", "pnpm-lock.yaml"},
		Architecture:    "arm64",
		Description:     "Node.js 20.x (arm64)",
	},
	"provided.al2023": {
		Name:            "provided.al2023",
		Version:         "al2023",
		BuildImage:      "public.ecr.aws/lambda/provided:al2023",
		LayerPath:       "/opt",
		FileExtensions:  []string{".go"},
		DependencyFiles: []string{"go.mod", "go.sum"},
		Architecture:    "x86_64",
		Description:     "Go custom runtime on Amazon Linux 2023 (x86_64)",
	},
	"provided.al2023-arm64": {
		Name:            "provided.al2023",
		Version:         "al2023",
		BuildImage:      "public.ecr.aws/lambda/provided:al2023-arm64",
		LayerPath:       "/opt",
		FileExtensions:  []string{".go"},
		DependencyFiles: []string{"go.mod", "go.sum"},
		Architecture:    "arm64",
		Description:     "Go custom runtime on Amazon Linux 2023 (arm64)",
	},
	"provided.al2": {
		Name:            "provided.al2",
		Version:         "al2",
		BuildImage:      "public.ecr.aws/lambda/provided:al2",
		LayerPath:       "/opt",
		FileExtensions:  []string{".go"},
		DependencyFiles: []string{"go.mod", "go.sum"},
		Architecture:    "x86_64",
		Description:     "Go custom runtime on Amazon Linux 2 (x86_64)",
	},
	"provided.al2-arm64": {
		Name:            "provided.al2",
		Version:         "al2",
		BuildImage:      "public.ecr.aws/lambda/provided:al2-arm64",
		LayerPath:       "/opt",
		FileExtensions:  []string{".go"},
		DependencyFiles: []string{"go.mod", "go.sum"},
		Architecture:    "arm64",
		Description:     "Go custom runtime on Amazon Linux 2 (arm64)",
	},
	"java11": {
		Name:            "java11",
		Version:         "11",
		BuildImage:      "public.ecr.aws/lambda/java:11",
		LayerPath:       "/opt/java",
		FileExtensions:  []string{".java"},
		DependencyFiles: []string{"pom.xml", "build.gradle", "build.gradle.kts"},
		Architecture:    "x86_64",
		Description:     "Java 11 (x86_64)",
	},
	"java17": {
		Name:            "java17",
		Version:         "17",
		BuildImage:      "public.ecr.aws/lambda/java:17",
		LayerPath:       "/opt/java",
		FileExtensions:  []string{".java"},
		DependencyFiles: []string{"pom.xml", "build.gradle", "build.gradle.kts"},
		Architecture:    "x86_64",
		Description:     "Java 17 (x86_64)",
	},
}

// RuntimeDetector handles runtime detection from source code
type RuntimeDetector struct {
	sourcePath string
}

// NewRuntimeDetector creates a new runtime detector
func NewRuntimeDetector(sourcePath string) *RuntimeDetector {
	return &RuntimeDetector{
		sourcePath: sourcePath,
	}
}

// DetectRuntime analyzes the source directory to determine the runtime
func (d *RuntimeDetector) DetectRuntime() (*Runtime, error) {
	// Check for dependency files first (most reliable)
	for _, runtime := range SupportedRuntimes {
		for _, depFile := range runtime.DependencyFiles {
			if _, err := os.Stat(filepath.Join(d.sourcePath, depFile)); err == nil {
				return runtime, nil
			}
		}
	}

	// Fall back to file extension detection
	fileCount := make(map[string]int)
	err := filepath.Walk(d.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := filepath.Ext(path)
			fileCount[ext]++
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Find the most common supported file extension
	maxCount := 0
	var detectedRuntime *Runtime
	for _, runtime := range SupportedRuntimes {
		for _, ext := range runtime.FileExtensions {
			if count := fileCount[ext]; count > maxCount {
				maxCount = count
				detectedRuntime = runtime
			}
		}
	}

	if detectedRuntime == nil {
		return nil, fmt.Errorf("unable to detect runtime from source files")
	}

	return detectedRuntime, nil
}

// DetectHandler attempts to find the Lambda handler in the source code
func (d *RuntimeDetector) DetectHandler(runtime *Runtime) (string, error) {
	switch {
	case strings.HasPrefix(runtime.Name, "python"):
		return d.detectPythonHandler()
	case strings.HasPrefix(runtime.Name, "nodejs"):
		return d.detectNodeHandler()
	case strings.HasPrefix(runtime.Name, "go"), strings.HasPrefix(runtime.Name, "provided.al2"):
		return "bootstrap", nil
	case strings.HasPrefix(runtime.Name, "java"):
		return d.detectJavaHandler()
	default:
		return "", fmt.Errorf("handler detection not implemented for runtime %s", runtime.Name)
	}
}

func (d *RuntimeDetector) detectPythonHandler() (string, error) {
	// Look for common Lambda handler patterns
	commonHandlers := []string{
		"app.lambda_handler",
		"handler.lambda_handler",
		"main.lambda_handler",
		"index.handler",
		"lambda_function.lambda_handler",
	}

	for _, handler := range commonHandlers {
		parts := strings.Split(handler, ".")
		if len(parts) == 2 {
			filename := parts[0] + ".py"
			if _, err := os.Stat(filepath.Join(d.sourcePath, filename)); err == nil {
				// Check if the file contains the handler function
				content, err := os.ReadFile(filepath.Join(d.sourcePath, filename))
				if err == nil && strings.Contains(string(content), parts[1]) {
					return handler, nil
				}
			}
		}
	}

	return "app.lambda_handler", nil // Default
}

func (d *RuntimeDetector) detectNodeHandler() (string, error) {
	// Look for common Node.js handler patterns
	commonHandlers := []string{
		"index.handler",
		"app.handler",
		"handler.handler",
		"index.lambdaHandler",
		"app.lambdaHandler",
	}

	for _, handler := range commonHandlers {
		parts := strings.Split(handler, ".")
		if len(parts) == 2 {
			// Check both .js and .ts files
			for _, ext := range []string{".js", ".ts", ".mjs"} {
				filename := parts[0] + ext
				if _, err := os.Stat(filepath.Join(d.sourcePath, filename)); err == nil {
					return handler, nil
				}
			}
		}
	}

	return "index.handler", nil // Default
}

func (d *RuntimeDetector) detectJavaHandler() (string, error) {
	// For Java, we need to find the main handler class
	// This is a simplified detection - real implementation would parse Java files
	return "com.example.Handler::handleRequest", nil
}

// GetRuntimeByName returns a runtime configuration by name
func GetRuntimeByName(name string) (*Runtime, error) {
	runtime, exists := SupportedRuntimes[name]
	if !exists {
		return nil, fmt.Errorf("unsupported runtime: %s", name)
	}
	return runtime, nil
}

// GetRuntimeNames returns all supported runtime names
func GetRuntimeNames() []string {
	names := make([]string, 0, len(SupportedRuntimes))
	for name := range SupportedRuntimes {
		names = append(names, name)
	}
	return names
}

// GetRuntimeDescriptions returns runtime names with descriptions for UI selection
func GetRuntimeDescriptions() []string {
	descriptions := make([]string, 0, len(SupportedRuntimes))
	for _, runtime := range SupportedRuntimes {
		descriptions = append(descriptions, runtime.Description)
	}
	return descriptions
}

// GetRuntimeByDescription returns runtime by its description
func GetRuntimeByDescription(description string) (*Runtime, error) {
	for _, runtime := range SupportedRuntimes {
		if runtime.Description == description {
			return runtime, nil
		}
	}
	return nil, fmt.Errorf("runtime not found for description: %s", description)
}

// GetRuntimesByLanguage returns runtimes filtered by language
func GetRuntimesByLanguage(language string) []*Runtime {
	var runtimes []*Runtime
	for _, runtime := range SupportedRuntimes {
		switch language {
		case "python":
			if strings.HasPrefix(runtime.Name, "python") {
				runtimes = append(runtimes, runtime)
			}
		case "nodejs", "javascript":
			if strings.HasPrefix(runtime.Name, "nodejs") {
				runtimes = append(runtimes, runtime)
			}
		case "go":
			if strings.HasPrefix(runtime.Name, "go") || strings.HasPrefix(runtime.Name, "provided.al2") {
				runtimes = append(runtimes, runtime)
			}
		case "java":
			if strings.HasPrefix(runtime.Name, "java") {
				runtimes = append(runtimes, runtime)
			}
		}
	}
	return runtimes
}

// GetRuntimesByLanguageAndArch returns runtimes filtered by language and architecture
func GetRuntimesByLanguageAndArch(language, architecture string) []*Runtime {
	var runtimes []*Runtime
	for _, runtime := range SupportedRuntimes {
		// Check if runtime matches the language
		var matchesLanguage bool
		switch language {
		case "python":
			matchesLanguage = strings.HasPrefix(runtime.Name, "python")
		case "nodejs", "javascript":
			matchesLanguage = strings.HasPrefix(runtime.Name, "nodejs")
		case "go":
			matchesLanguage = strings.HasPrefix(runtime.Name, "go") || strings.HasPrefix(runtime.Name, "provided.al2")
		case "java":
			matchesLanguage = strings.HasPrefix(runtime.Name, "java")
		}
		
		// Check if runtime matches the architecture
		if matchesLanguage && runtime.Architecture == architecture {
			runtimes = append(runtimes, runtime)
		}
	}
	sortRuntimes(runtimes)
	return runtimes
}

// GetRuntimesByArch returns runtimes filtered by architecture
func GetRuntimesByArch(architecture string) []*Runtime {
	var runtimes []*Runtime
	for _, runtime := range SupportedRuntimes {
		if runtime.Architecture == architecture {
			runtimes = append(runtimes, runtime)
		}
	}
	sortRuntimes(runtimes)
	return runtimes
}

// sortRuntimes sorts runtimes by language and version
func sortRuntimes(runtimes []*Runtime) {
	sort.Slice(runtimes, func(i, j int) bool {
		rt1, rt2 := runtimes[i], runtimes[j]
		
		// Extract language from runtime name
		lang1 := getLanguageFromRuntime(rt1.Name)
		lang2 := getLanguageFromRuntime(rt2.Name)
		
		// First sort by language
		if lang1 != lang2 {
			return lang1 < lang2
		}
		
		// Then sort by version within the same language
		return compareVersions(rt1.Version, rt2.Version)
	})
}

// getLanguageFromRuntime extracts language from runtime name
func getLanguageFromRuntime(runtimeName string) string {
	switch {
	case strings.HasPrefix(runtimeName, "python"):
		return "python"
	case strings.HasPrefix(runtimeName, "nodejs"):
		return "nodejs"
	case strings.HasPrefix(runtimeName, "go"), strings.HasPrefix(runtimeName, "provided.al2"):
		return "go"
	case strings.HasPrefix(runtimeName, "java"):
		return "java"
	default:
		return runtimeName
	}
}

// compareVersions compares version strings (e.g., "3.8" vs "3.11")
func compareVersions(v1, v2 string) bool {
	// Handle special cases for AL2/AL2023
	if v1 == "al2023" && v2 != "al2023" {
		return false // AL2023 should come after other versions
	}
	if v2 == "al2023" && v1 != "al2023" {
		return true
	}
	if v1 == "al2" && v2 != "al2" && v2 != "al2023" {
		return false // AL2 should come after other versions but before AL2023
	}
	if v2 == "al2" && v1 != "al2" && v1 != "al2023" {
		return true
	}
	
	// Standard version comparison (lexicographic is fine for our versions)
	return v1 < v2
}
