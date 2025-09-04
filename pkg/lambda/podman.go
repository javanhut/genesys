package lambda

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PodmanBuilder handles container-based Lambda builds
type PodmanBuilder struct {
	runtime    *Runtime
	sourcePath string
	outputPath string
	debug      bool
}

// NewPodmanBuilder creates a new Podman-based builder
func NewPodmanBuilder(runtime *Runtime, sourcePath, outputPath string) *PodmanBuilder {
	return &PodmanBuilder{
		runtime:    runtime,
		sourcePath: sourcePath,
		outputPath: outputPath,
		debug:      false,
	}
}

// SetDebug enables debug output
func (b *PodmanBuilder) SetDebug(debug bool) {
	b.debug = debug
}

// EnableDebug is a convenience method to enable debug output
func (b *PodmanBuilder) EnableDebug() {
	b.debug = true
}

// BuildLayer builds a Lambda layer using Podman
func (b *PodmanBuilder) BuildLayer() error {
	// Create temporary build directory
	buildDir, err := os.MkdirTemp("", "lambda-layer-*")
	if err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}
	defer os.RemoveAll(buildDir)

	// Create layer directory structure
	layerDir := filepath.Join(buildDir, "layer")
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		return fmt.Errorf("failed to create layer directory: %w", err)
	}

	// Build based on runtime
	switch {
	case strings.HasPrefix(b.runtime.Name, "python"):
		err = b.buildPythonLayer(layerDir)
	case strings.HasPrefix(b.runtime.Name, "nodejs"):
		err = b.buildNodeLayer(layerDir)
	case strings.HasPrefix(b.runtime.Name, "go"), strings.HasPrefix(b.runtime.Name, "provided.al2"):
		err = b.buildGoLayer(layerDir)
	case strings.HasPrefix(b.runtime.Name, "java"):
		err = b.buildJavaLayer(layerDir)
	default:
		return fmt.Errorf("unsupported runtime for layer build: %s", b.runtime.Name)
	}

	if err != nil {
		return fmt.Errorf("failed to build %s layer: %w", b.runtime.Name, err)
	}

	// Create ZIP file
	return b.createLayerZip(layerDir)
}

// BuildFunction builds a Lambda function deployment package
func (b *PodmanBuilder) BuildFunction() error {
	// Create temporary build directory
	buildDir, err := os.MkdirTemp("", "lambda-function-*")
	if err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}
	defer os.RemoveAll(buildDir)

	// Copy source files
	if err := b.copySourceFiles(buildDir); err != nil {
		return fmt.Errorf("failed to copy source files: %w", err)
	}

	// Build based on runtime
	switch {
	case strings.HasPrefix(b.runtime.Name, "go"), strings.HasPrefix(b.runtime.Name, "provided.al2"):
		// Go needs compilation
		if err := b.buildGoFunction(buildDir); err != nil {
			return fmt.Errorf("failed to build Go function: %w", err)
		}
	case strings.HasPrefix(b.runtime.Name, "java"):
		// Java needs compilation
		if err := b.buildJavaFunction(buildDir); err != nil {
			return fmt.Errorf("failed to build Java function: %w", err)
		}
	}

	// Create ZIP file
	return b.createFunctionZip(buildDir)
}

func (b *PodmanBuilder) buildPythonLayer(layerDir string) error {
	// Find requirements file
	reqFile := b.findDependencyFile([]string{"requirements.txt", "Pipfile", "poetry.lock"})
	if reqFile == "" {
		return fmt.Errorf("no Python dependency file found")
	}

	// Create container command
	pythonPath := filepath.Join(layerDir, "python")
	if err := os.MkdirAll(pythonPath, 0755); err != nil {
		return fmt.Errorf("failed to create python directory: %w", err)
	}

	// Build command based on dependency manager
	var cmd *exec.Cmd
	switch filepath.Base(reqFile) {
	case "requirements.txt":
		cmd = exec.Command("podman", "run", "--rm",
			"--entrypoint", "/bin/sh",
			"-v", fmt.Sprintf("%s:/src:ro", b.sourcePath),
			"-v", fmt.Sprintf("%s:/opt/python", pythonPath),
			b.runtime.BuildImage,
			"-c", fmt.Sprintf("pip install -r /src/requirements.txt -t /opt/python"),
		)
	case "Pipfile":
		// For Pipfile, we need pipenv
		cmd = exec.Command("podman", "run", "--rm",
			"--entrypoint", "/bin/sh",
			"-v", fmt.Sprintf("%s:/src:ro", b.sourcePath),
			"-v", fmt.Sprintf("%s:/opt/python", pythonPath),
			b.runtime.BuildImage,
			"-c", "pip install pipenv && cd /src && pipenv install --system --deploy",
		)
	case "poetry.lock":
		// For poetry.lock, we need poetry
		cmd = exec.Command("podman", "run", "--rm",
			"--entrypoint", "/bin/sh",
			"-v", fmt.Sprintf("%s:/src:ro", b.sourcePath),
			"-v", fmt.Sprintf("%s:/opt/python", pythonPath),
			b.runtime.BuildImage,
			"-c", "pip install poetry && cd /src && poetry export -f requirements.txt | pip install -r /dev/stdin -t /opt/python",
		)
	}

	return b.runCommand(cmd)
}

func (b *PodmanBuilder) buildNodeLayer(layerDir string) error {
	// Find package.json
	packageFile := filepath.Join(b.sourcePath, "package.json")
	if _, err := os.Stat(packageFile); err != nil {
		return fmt.Errorf("package.json not found")
	}

	// Create node_modules directory
	nodeModulesPath := filepath.Join(layerDir, "nodejs", "node_modules")
	if err := os.MkdirAll(nodeModulesPath, 0755); err != nil {
		return fmt.Errorf("failed to create node_modules directory: %w", err)
	}

	// Determine package manager
	var cmd *exec.Cmd
	if _, err := os.Stat(filepath.Join(b.sourcePath, "yarn.lock")); err == nil {
		// Use Yarn
		cmd = exec.Command("podman", "run", "--rm",
			"--entrypoint", "/bin/sh",
			"-v", fmt.Sprintf("%s:/src:ro", b.sourcePath),
			"-v", fmt.Sprintf("%s:/opt/nodejs", filepath.Join(layerDir, "nodejs")),
			"-w", "/src",
			b.runtime.BuildImage,
			"-c", "yarn install --production --modules-folder /opt/nodejs/node_modules",
		)
	} else if _, err := os.Stat(filepath.Join(b.sourcePath, "pnpm-lock.yaml")); err == nil {
		// Use pnpm
		cmd = exec.Command("podman", "run", "--rm",
			"--entrypoint", "/bin/sh",
			"-v", fmt.Sprintf("%s:/src:ro", b.sourcePath),
			"-v", fmt.Sprintf("%s:/opt/nodejs", filepath.Join(layerDir, "nodejs")),
			"-w", "/src",
			b.runtime.BuildImage,
			"-c", "npm install -g pnpm && pnpm install --prod --shamefully-hoist --prefix /opt/nodejs",
		)
	} else {
		// Use npm
		cmd = exec.Command("podman", "run", "--rm",
			"--entrypoint", "/bin/sh",
			"-v", fmt.Sprintf("%s:/src:ro", b.sourcePath),
			"-v", fmt.Sprintf("%s:/opt/nodejs", filepath.Join(layerDir, "nodejs")),
			"-w", "/src",
			b.runtime.BuildImage,
			"-c", "npm install --production --prefix /opt/nodejs",
		)
	}

	return b.runCommand(cmd)
}

func (b *PodmanBuilder) buildGoLayer(layerDir string) error {
	// Go typically doesn't use layers for dependencies
	// Dependencies are compiled into the binary
	return nil
}

func (b *PodmanBuilder) buildJavaLayer(layerDir string) error {
	// Create lib directory for Java dependencies
	libPath := filepath.Join(layerDir, "java", "lib")
	if err := os.MkdirAll(libPath, 0755); err != nil {
		return fmt.Errorf("failed to create lib directory: %w", err)
	}

	// Build based on build tool
	var cmd *exec.Cmd
	if _, err := os.Stat(filepath.Join(b.sourcePath, "pom.xml")); err == nil {
		// Maven
		cmd = exec.Command("podman", "run", "--rm",
			"--entrypoint", "/bin/sh",
			"-v", fmt.Sprintf("%s:/src:ro", b.sourcePath),
			"-v", fmt.Sprintf("%s:/opt/java/lib", libPath),
			"-w", "/src",
			b.runtime.BuildImage,
			"-c", "mvn dependency:copy-dependencies -DoutputDirectory=/opt/java/lib",
		)
	} else if _, err := os.Stat(filepath.Join(b.sourcePath, "build.gradle")); err == nil {
		// Gradle
		cmd = exec.Command("podman", "run", "--rm",
			"--entrypoint", "/bin/sh",
			"-v", fmt.Sprintf("%s:/src:ro", b.sourcePath),
			"-v", fmt.Sprintf("%s:/opt/java/lib", libPath),
			"-w", "/src",
			b.runtime.BuildImage,
			"-c", "gradle copyDependencies -Pdestination=/opt/java/lib",
		)
	} else {
		return fmt.Errorf("no Java build file found (pom.xml or build.gradle)")
	}

	return b.runCommand(cmd)
}

func (b *PodmanBuilder) buildGoFunction(buildDir string) error {
	// Determine architecture based on runtime
	goarch := "amd64"
	if b.runtime.Architecture == "arm64" {
		goarch = "arm64"
	}

	// Build Go binary
	cmd := exec.Command("podman", "run", "--rm",
		"--entrypoint", "/bin/sh",
		"-v", fmt.Sprintf("%s:/src:ro", b.sourcePath),
		"-v", fmt.Sprintf("%s:/build", buildDir),
		"-w", "/src",
		"-e", "GOOS=linux",
		"-e", fmt.Sprintf("GOARCH=%s", goarch),
		"-e", "CGO_ENABLED=0",
		b.runtime.BuildImage,
		"-c", "go build -o /build/bootstrap .",
	)

	return b.runCommand(cmd)
}

func (b *PodmanBuilder) buildJavaFunction(buildDir string) error {
	// Build based on build tool
	var cmd *exec.Cmd
	if _, err := os.Stat(filepath.Join(b.sourcePath, "pom.xml")); err == nil {
		// Maven
		cmd = exec.Command("podman", "run", "--rm",
			"--entrypoint", "/bin/sh",
			"-v", fmt.Sprintf("%s:/src:ro", b.sourcePath),
			"-v", fmt.Sprintf("%s:/build", buildDir),
			"-w", "/src",
			b.runtime.BuildImage,
			"-c", "mvn clean package -DoutputDirectory=/build",
		)
	} else if _, err := os.Stat(filepath.Join(b.sourcePath, "build.gradle")); err == nil {
		// Gradle
		cmd = exec.Command("podman", "run", "--rm",
			"--entrypoint", "/bin/sh",
			"-v", fmt.Sprintf("%s:/src:ro", b.sourcePath),
			"-v", fmt.Sprintf("%s:/build", buildDir),
			"-w", "/src",
			b.runtime.BuildImage,
			"-c", "gradle build -Pdestination=/build",
		)
	}

	return b.runCommand(cmd)
}

func (b *PodmanBuilder) runCommand(cmd *exec.Cmd) error {
	if b.debug {
		fmt.Printf("Running: %s\n", strings.Join(cmd.Args, " "))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	if b.debug {
		fmt.Printf("Output: %s\n", stdout.String())
	}

	return nil
}

func (b *PodmanBuilder) findDependencyFile(files []string) string {
	for _, file := range files {
		path := filepath.Join(b.sourcePath, file)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func (b *PodmanBuilder) copySourceFiles(destDir string) error {
	return filepath.Walk(b.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and directories
		if strings.HasPrefix(filepath.Base(path), ".") && path != b.sourcePath {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common non-source directories
		if info.IsDir() {
			switch filepath.Base(path) {
			case "node_modules", "venv", "__pycache__", "target", "build", "dist":
				return filepath.SkipDir
			}
		}

		// Calculate destination path
		relPath, err := filepath.Rel(b.sourcePath, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, relPath)

		// Create directory or copy file
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, content, info.Mode())
	})
}

func (b *PodmanBuilder) createLayerZip(layerDir string) error {
	cmd := exec.Command("zip", "-r", b.outputPath, ".")
	cmd.Dir = layerDir
	return b.runCommand(cmd)
}

func (b *PodmanBuilder) createFunctionZip(buildDir string) error {
	cmd := exec.Command("zip", "-r", b.outputPath, ".")
	cmd.Dir = buildDir
	return b.runCommand(cmd)
}

// CheckPodmanAvailable checks if Podman is installed and available
func CheckPodmanAvailable() error {
	cmd := exec.Command("podman", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("podman not found: %w. Please install podman to use Lambda builder", err)
	}
	return nil
}
