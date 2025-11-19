package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/BurntSushi/toml"
	"github.com/javanhut/genesys/pkg/lambda"
	"github.com/javanhut/genesys/pkg/validation"
)

// InteractiveLambdaConfig handles interactive Lambda function configuration
type InteractiveLambdaConfig struct {
	config *Config
}

// NewInteractiveLambdaConfig creates a new interactive Lambda configuration
func NewInteractiveLambdaConfig() (*InteractiveLambdaConfig, error) {
	// For interactive mode, we don't need to load a specific config file
	cfg := &Config{}

	return &InteractiveLambdaConfig{
		config: cfg,
	}, nil
}

// LambdaFunctionConfig represents Lambda function configuration for TOML
type LambdaFunctionConfig struct {
	Metadata   LambdaMetadata   `toml:"metadata"`
	Build      LambdaBuild      `toml:"build"`
	Function   LambdaFunction   `toml:"function"`
	Deployment LambdaDeployment `toml:"deployment"`
	Triggers   []LambdaTrigger  `toml:"triggers,omitempty"`
	Layer      *LambdaLayer     `toml:"layer,omitempty"`
	IAM        *LambdaIAM       `toml:"iam,omitempty"`
}

// LambdaMetadata contains function metadata
type LambdaMetadata struct {
	Name        string `toml:"name"`
	Runtime     string `toml:"runtime"`
	Handler     string `toml:"handler"`
	Description string `toml:"description"`
}

// LambdaBuild contains build configuration
type LambdaBuild struct {
	SourcePath       string `toml:"source_path"`
	BuildMethod      string `toml:"build_method"`
	LayerAuto        bool   `toml:"layer_auto"`
	RequirementsFile string `toml:"requirements_file,omitempty"`
}

// LambdaFunction contains function configuration
type LambdaFunction struct {
	MemoryMB       int               `toml:"memory_mb"`
	TimeoutSeconds int               `toml:"timeout_seconds"`
	Environment    map[string]string `toml:"environment,omitempty"`
}

// LambdaDeployment contains deployment configuration
type LambdaDeployment struct {
	FunctionURL  bool   `toml:"function_url"`
	CORSEnabled  bool   `toml:"cors_enabled"`
	AuthType     string `toml:"auth_type"`
	Architecture string `toml:"architecture"`
}

// LambdaTrigger represents a function trigger
type LambdaTrigger struct {
	Type   string `toml:"type"`
	Path   string `toml:"path,omitempty"`
	Method string `toml:"method,omitempty"`
}

// LambdaLayer represents layer configuration
type LambdaLayer struct {
	Name               string   `toml:"name"`
	Description        string   `toml:"description"`
	CompatibleRuntimes []string `toml:"compatible_runtimes"`
}

// LambdaIAM represents IAM configuration for Lambda
type LambdaIAM struct {
	RoleName         string            `toml:"role_name"`
	RoleArn          string            `toml:"role_arn,omitempty"`
	RequiredPolicies []string          `toml:"required_policies"`
	CustomPolicies   []string          `toml:"custom_policies,omitempty"`
	AutoManage       bool              `toml:"auto_manage"`
	AutoCleanup      bool              `toml:"auto_cleanup"`
	ManagedBy        string            `toml:"managed_by,omitempty"`
	PolicyDetails    map[string]string `toml:"policy_details,omitempty"` // Policy ARN -> Description mapping
}

// UnifiedIAMConfig is an alias for LambdaIAM to support unified IAM role management
type UnifiedIAMConfig = LambdaIAM

// CreateLambdaConfig creates Lambda configuration interactively
func (ilc *InteractiveLambdaConfig) CreateLambdaConfig() (*LambdaFunctionConfig, string, error) {
	fmt.Println("Lambda Function Configuration")
	fmt.Println("=============================")
	fmt.Println()

	// Function name with auto-formatting
	functionName, err := ilc.getFunctionName()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get function name: %w", err)
	}

	// Source path with file browser option
	var sourcePath string
	var useFileBrowser bool

	// Ask if user wants to use file browser or type path
	browserPrompt := &survey.Confirm{
		Message: "Browse for source code directory?",
		Default: true,
		Help:    "Use interactive directory browser or type path manually",
	}
	survey.AskOne(browserPrompt, &useFileBrowser)

	if useFileBrowser {
		var err error
		sourcePath, err = selectDirectory(".")
		if err != nil {
			return nil, "", fmt.Errorf("failed to select directory: %w", err)
		}
	} else {
		sourcePrompt := &survey.Input{
			Message: "Source code path:",
			Default: ".",
			Help:    "Path to your Lambda function source code (supports ~ and relative paths)",
		}
		if err := survey.AskOne(sourcePrompt, &sourcePath); err != nil {
			return nil, "", err
		}
	}

	// Expand path
	if sourcePath == "." {
		sourcePath, _ = os.Getwd()
	} else if strings.HasPrefix(sourcePath, "~") {
		home, _ := os.UserHomeDir()
		sourcePath = filepath.Join(home, sourcePath[2:])
	}

	// Detect runtime
	detector := lambda.NewRuntimeDetector(sourcePath)
	detectedRuntime, err := detector.DetectRuntime()

	var runtime string
	var selectedRuntime *lambda.Runtime

	var selectedArchitecture string
	var language string

	if err == nil {
		// Runtime detected - show detected language
		language = strings.Split(detectedRuntime.Name, "3")[0] // Extract language (python, nodejs, etc.)
		if strings.Contains(detectedRuntime.Name, "nodejs") {
			language = "nodejs"
		}
		fmt.Printf("🔍 Detected language: %s\n", strings.Title(language))
	}

	// Step 1: ALWAYS Select Architecture FIRST
	// This must happen before ANY runtime selection to avoid confusion
	fmt.Println("\n🏗️  Step 1: Select Architecture")
	fmt.Println("Choose the processor architecture for your Lambda function:")
	fmt.Println("• x86_64: Standard architecture, widely compatible")
	fmt.Println("• arm64: AWS Graviton2, up to 34% cost savings")

	var selectedArchDescription string
	archPrompt := &survey.Select{
		Message: "Architecture:",
		Options: []string{
			"x86_64 (Intel/AMD)",
			"arm64 (AWS Graviton2)",
		},
		Default: "x86_64 (Intel/AMD)",
	}
	if err := survey.AskOne(archPrompt, &selectedArchDescription); err != nil {
		return nil, "", err
	}

	if strings.Contains(selectedArchDescription, "arm64") {
		selectedArchitecture = "arm64"
		fmt.Printf("✓ Selected architecture: arm64 (AWS Graviton2)\n")
	} else {
		selectedArchitecture = "x86_64"
		fmt.Printf("✓ Selected architecture: x86_64 (Intel/AMD)\n")
	}

	// Step 2: Select Runtime Version (ONLY for selected architecture)
	fmt.Printf("\n🚀 Step 2: Select Runtime Version\n")
	if language != "" {
		fmt.Printf("Showing %s versions available for %s:\n", strings.Title(language), selectedArchitecture)
		// Show runtimes for detected language and selected architecture ONLY
		availableRuntimes := lambda.GetRuntimesByLanguageAndArch(language, selectedArchitecture)

		if len(availableRuntimes) > 0 {
			// Create clean runtime options without architecture suffix
			options := make([]string, len(availableRuntimes))
			for i, rt := range availableRuntimes {
				// Extract clean version (e.g., "Python 3.11" instead of "Python 3.11 (x86_64)")
				cleanName := strings.Split(rt.Description, " (")[0]
				options[i] = cleanName
			}

			var selectedVersion string
			runtimePrompt := &survey.Select{
				Message: fmt.Sprintf("%s version:", strings.Title(language)),
				Options: options,
				Help:    fmt.Sprintf("Choose the %s version (for %s architecture)", language, selectedArchitecture),
			}
			if err := survey.AskOne(runtimePrompt, &selectedVersion); err != nil {
				return nil, "", err
			}

			// Find matching runtime
			for _, rt := range availableRuntimes {
				if strings.HasPrefix(rt.Description, selectedVersion) {
					selectedRuntime = rt
					runtime = selectedRuntime.Name
					break
				}
			}
		}
	}

	// If no runtime selected, show all runtimes for selected architecture
	if runtime == "" {
		availableRuntimes := lambda.GetRuntimesByArch(selectedArchitecture)
		options := make([]string, len(availableRuntimes))
		for i, rt := range availableRuntimes {
			// Extract clean version without architecture
			cleanName := strings.Split(rt.Description, " (")[0]
			options[i] = cleanName
		}

		var selectedVersion string
		runtimePrompt := &survey.Select{
			Message: "Select runtime:",
			Options: options,
			Help:    fmt.Sprintf("Choose the runtime for your function (%s)", selectedArchitecture),
		}
		if err := survey.AskOne(runtimePrompt, &selectedVersion); err != nil {
			return nil, "", err
		}

		// Find matching runtime
		for _, rt := range availableRuntimes {
			if strings.HasPrefix(rt.Description, selectedVersion) {
				selectedRuntime = rt
				runtime = selectedRuntime.Name
				break
			}
		}
	}

	// Set detected runtime for later use
	if selectedRuntime != nil {
		detectedRuntime = selectedRuntime
	} else if detectedRuntime != nil && detectedRuntime.Architecture != selectedArchitecture {
		// If the detected runtime has a different architecture, try to find a matching runtime with the selected architecture
		// but same language and version
		language := strings.Split(detectedRuntime.Name, "3")[0]
		if strings.Contains(detectedRuntime.Name, "nodejs") {
			language = "nodejs"
		}

		// Try to find a runtime with the same language and version but selected architecture
		availableRuntimes := lambda.GetRuntimesByLanguageAndArch(language, selectedArchitecture)
		for _, rt := range availableRuntimes {
			if strings.Contains(rt.Name, detectedRuntime.Version) {
				detectedRuntime = rt
				break
			}
		}
	}

	// Detect handler
	handler, err := detector.DetectHandler(detectedRuntime)
	if err != nil {
		// Ask for handler
		handlerPrompt := &survey.Input{
			Message: "Handler:",
			Default: "index.handler",
			Help:    "Function handler (e.g., app.lambda_handler for Python)",
		}
		if err := survey.AskOne(handlerPrompt, &handler); err != nil {
			return nil, "", err
		}
	} else {
		// Confirm detected handler
		handlerPrompt := &survey.Input{
			Message: "Handler:",
			Default: handler,
			Help:    "Function handler (detected from source)",
		}
		if err := survey.AskOne(handlerPrompt, &handler); err != nil {
			return nil, "", err
		}
	}

	// Memory configuration
	memoryOptions := []string{"128", "256", "512", "1024", "2048", "3072", "4096", "5120", "6144", "7168", "8192", "9216", "10240"}
	var memoryStr string
	memoryPrompt := &survey.Select{
		Message: "Memory (MB):",
		Options: memoryOptions,
		Default: "512",
		Help:    "Amount of memory allocated to the function",
	}
	if err := survey.AskOne(memoryPrompt, &memoryStr); err != nil {
		return nil, "", err
	}
	memory := 512
	fmt.Sscanf(memoryStr, "%d", &memory)

	// Timeout configuration
	var timeout int
	timeoutPrompt := &survey.Input{
		Message: "Timeout (seconds):",
		Default: "30",
		Help:    "Maximum execution time (1-900 seconds)",
	}
	if err := survey.AskOne(timeoutPrompt, &timeout); err != nil {
		return nil, "", err
	}

	// Environment variables
	envVars := make(map[string]string)
	addEnv := false
	envPrompt := &survey.Confirm{
		Message: "Add environment variables?",
		Default: false,
	}
	survey.AskOne(envPrompt, &addEnv)

	if addEnv {
		for {
			var envKey, envValue string
			keyPrompt := &survey.Input{
				Message: "Environment variable name (or press Enter to finish):",
			}
			if err := survey.AskOne(keyPrompt, &envKey); err != nil || envKey == "" {
				break
			}

			valuePrompt := &survey.Input{
				Message: fmt.Sprintf("Value for %s:", envKey),
			}
			if err := survey.AskOne(valuePrompt, &envValue); err != nil {
				break
			}

			envVars[envKey] = envValue
		}
	}

	// Function URL
	var functionURL bool
	urlPrompt := &survey.Confirm{
		Message: "Enable function URL?",
		Default: true,
		Help:    "Create a public HTTPS endpoint for your function",
	}
	survey.AskOne(urlPrompt, &functionURL)

	// Create configuration
	config := &LambdaFunctionConfig{
		Metadata: LambdaMetadata{
			Name:        functionName,
			Runtime:     runtime,
			Handler:     handler,
			Description: fmt.Sprintf("Lambda function %s created with Genesys", functionName),
		},
		Build: LambdaBuild{
			SourcePath:  sourcePath,
			BuildMethod: "podman",
			LayerAuto:   true,
		},
		Function: LambdaFunction{
			MemoryMB:       memory,
			TimeoutSeconds: timeout,
			Environment:    envVars,
		},
		Deployment: LambdaDeployment{
			FunctionURL:  functionURL,
			CORSEnabled:  functionURL,
			AuthType:     "AWS_IAM",
			Architecture: detectedRuntime.Architecture,
		},
	}

	// Check for dependencies and configure layer
	hasLayer := false
	for _, depFile := range detectedRuntime.DependencyFiles {
		if _, err := os.Stat(filepath.Join(sourcePath, depFile)); err == nil {
			hasLayer = true
			config.Build.RequirementsFile = depFile
			break
		}
	}

	if hasLayer {
		layerName := fmt.Sprintf("%s-deps", functionName)
		if detectedRuntime.Architecture == "arm64" {
			layerName += "-arm64"
		}
		config.Layer = &LambdaLayer{
			Name:               layerName,
			Description:        fmt.Sprintf("Dependencies for %s Lambda function (%s)", functionName, detectedRuntime.Architecture),
			CompatibleRuntimes: []string{runtime},
		}
	}

	// Add default API Gateway trigger if function URL is not enabled
	if !functionURL {
		addTrigger := false
		triggerPrompt := &survey.Confirm{
			Message: "Add API Gateway trigger?",
			Default: true,
		}
		survey.AskOne(triggerPrompt, &addTrigger)

		if addTrigger {
			config.Triggers = append(config.Triggers, LambdaTrigger{
				Type:   "api_gateway",
				Path:   "/{proxy+}",
				Method: "ANY",
			})
		}
	}

	// Configure IAM role and permissions
	iamConfig, err := ilc.configureIAMPermissions(functionName, sourcePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to configure IAM: %w", err)
	}
	config.IAM = iamConfig

	return config, functionName, nil
}

// SaveConfig saves the Lambda configuration to a TOML file
func (ilc *InteractiveLambdaConfig) SaveConfig(config *LambdaFunctionConfig, functionName string) (string, error) {
	// Create resources/lambda directory
	resourceDir := "resources/lambda"
	if err := os.MkdirAll(resourceDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", resourceDir, err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("lambda_%s_%s.toml", functionName, config.Metadata.Runtime)
	filePath := filepath.Join(resourceDir, filename)

	// Check if file exists and add timestamp if needed
	if _, err := os.Stat(filePath); err == nil {
		filename = fmt.Sprintf("lambda_%s_%s_%s.toml", functionName, config.Metadata.Runtime, timestamp)
		filePath = filepath.Join(resourceDir, filename)
	}

	// Marshal configuration to TOML
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return "", fmt.Errorf("failed to encode configuration: %w", err)
	}

	absPath, _ := filepath.Abs(filePath)
	return absPath, nil
}

// ValidateLambdaConfig validates Lambda configuration
func ValidateLambdaConfig(config *LambdaFunctionConfig) error {
	// Validate function name
	if config.Metadata.Name == "" {
		return fmt.Errorf("function name is required")
	}

	// Validate runtime
	if _, err := lambda.GetRuntimeByName(config.Metadata.Runtime); err != nil {
		return fmt.Errorf("invalid runtime: %s", config.Metadata.Runtime)
	}

	// Validate handler
	if config.Metadata.Handler == "" {
		return fmt.Errorf("handler is required")
	}

	// Validate memory
	if config.Function.MemoryMB < 128 || config.Function.MemoryMB > 10240 {
		return fmt.Errorf("memory must be between 128 and 10240 MB")
	}

	// Validate timeout
	if config.Function.TimeoutSeconds < 1 || config.Function.TimeoutSeconds > 900 {
		return fmt.Errorf("timeout must be between 1 and 900 seconds")
	}

	// Validate source path
	if _, err := os.Stat(config.Build.SourcePath); err != nil {
		return fmt.Errorf("source path does not exist: %s", config.Build.SourcePath)
	}

	return nil
}

// configureIAMPermissions collects IAM requirements during interactive config
func (ilc *InteractiveLambdaConfig) configureIAMPermissions(functionName, sourcePath string) (*LambdaIAM, error) {
	fmt.Printf("\n🔐 IAM Permissions Configuration\n")
	fmt.Printf("================================\n")

	// Enhanced service permissions with multiple policy options
	servicePermissions := []struct {
		name        string
		description string
		policies    []struct {
			arn         string
			description string
			level       string // "read", "write", "full"
		}
	}{
		{
			name:        "CloudWatch Logs",
			description: "Basic logging capabilities (recommended)",
			policies: []struct {
				arn         string
				description string
				level       string
			}{
				{
					arn:         "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
					description: "Basic CloudWatch Logs access",
					level:       "basic",
				},
			},
		},
		{
			name:        "DynamoDB",
			description: "NoSQL database access",
			policies: []struct {
				arn         string
				description string
				level       string
			}{
				{
					arn:         "arn:aws:iam::aws:policy/AmazonDynamoDBReadOnlyAccess",
					description: "Read-only access to DynamoDB",
					level:       "read",
				},
				{
					arn:         "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess",
					description: "Full access to DynamoDB",
					level:       "full",
				},
			},
		},
		{
			name:        "S3",
			description: "Object storage access",
			policies: []struct {
				arn         string
				description string
				level       string
			}{
				{
					arn:         "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
					description: "Read-only access to S3",
					level:       "read",
				},
				{
					arn:         "arn:aws:iam::aws:policy/AmazonS3FullAccess",
					description: "Full access to S3",
					level:       "full",
				},
			},
		},
		{
			name:        "SQS",
			description: "Message queue access",
			policies: []struct {
				arn         string
				description string
				level       string
			}{
				{
					arn:         "arn:aws:iam::aws:policy/AmazonSQSReadOnlyAccess",
					description: "Read-only access to SQS",
					level:       "read",
				},
				{
					arn:         "arn:aws:iam::aws:policy/AmazonSQSFullAccess",
					description: "Full access to SQS",
					level:       "full",
				},
			},
		},
		{
			name:        "SNS",
			description: "Notification service access",
			policies: []struct {
				arn         string
				description string
				level       string
			}{
				{
					arn:         "arn:aws:iam::aws:policy/AmazonSNSReadOnlyAccess",
					description: "Read-only access to SNS",
					level:       "read",
				},
				{
					arn:         "arn:aws:iam::aws:policy/AmazonSNSFullAccess",
					description: "Full access to SNS",
					level:       "full",
				},
			},
		},
		{
			name:        "Secrets Manager",
			description: "Secure secrets access",
			policies: []struct {
				arn         string
				description string
				level       string
			}{
				{
					arn:         "arn:aws:iam::aws:policy/SecretsManagerReadWrite",
					description: "Read/write access to Secrets Manager",
					level:       "full",
				},
			},
		},
		{
			name:        "VPC",
			description: "Virtual Private Cloud access",
			policies: []struct {
				arn         string
				description string
				level       string
			}{
				{
					arn:         "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole",
					description: "VPC access for Lambda functions",
					level:       "full",
				},
			},
		},
	}

	// Start with basic required permissions
	var selectedPolicies []string
	var policyDetails = make(map[string]string)

	// Always include basic execution role
	selectedPolicies = append(selectedPolicies, "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole")
	policyDetails["arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"] = "Basic CloudWatch Logs access"

	fmt.Println("\nSelect additional AWS services your function will access:")
	fmt.Println("(Basic CloudWatch Logs access is automatically included)")

	// Create options for service selection
	serviceOptions := make([]string, len(servicePermissions)-1) // -1 to exclude CloudWatch Logs (already included)
	for i, svc := range servicePermissions[1:] {                // Skip CloudWatch Logs
		serviceOptions[i] = fmt.Sprintf("%s - %s", svc.name, svc.description)
	}

	// Add option for no additional services
	serviceOptions = append(serviceOptions, "None - Only basic logging")

	multiSelectPrompt := &survey.MultiSelect{
		Message: "Select AWS services:",
		Options: serviceOptions,
		Help:    "Choose all services your Lambda function needs to access. You can select multiple.",
	}

	var selectedOptions []string
	if err := survey.AskOne(multiSelectPrompt, &selectedOptions); err != nil {
		return nil, err
	}

	// Process selected services and choose permission levels
	for _, selected := range selectedOptions {
		if selected == "None - Only basic logging" {
			continue
		}

		// Find the corresponding service
		for _, svc := range servicePermissions[1:] { // Skip CloudWatch Logs
			expectedOption := fmt.Sprintf("%s - %s", svc.name, svc.description)
			if selected == expectedOption {
				// If service has multiple policy options, ask user to choose
				if len(svc.policies) > 1 {
					fmt.Printf("\nChoose permission level for %s:\n", svc.name)

					policyOptions := make([]string, len(svc.policies))
					for j, policy := range svc.policies {
						policyOptions[j] = fmt.Sprintf("%s (%s)", policy.description, policy.level)
					}

					var selectedPolicy string
					policyPrompt := &survey.Select{
						Message: fmt.Sprintf("Permission level for %s:", svc.name),
						Options: policyOptions,
						Help:    "Choose the minimum permissions your function needs",
					}

					if err := survey.AskOne(policyPrompt, &selectedPolicy); err != nil {
						return nil, err
					}

					// Find selected policy
					for j, option := range policyOptions {
						if selectedPolicy == option {
							policy := svc.policies[j]
							selectedPolicies = append(selectedPolicies, policy.arn)
							policyDetails[policy.arn] = policy.description
							break
						}
					}
				} else {
					// Only one policy option
					policy := svc.policies[0]
					selectedPolicies = append(selectedPolicies, policy.arn)
					policyDetails[policy.arn] = policy.description
				}
				break
			}
		}
	}

	// Ask about custom policies
	addCustom := false
	customPrompt := &survey.Confirm{
		Message: "Add custom IAM policy ARNs?",
		Default: false,
		Help:    "Add your own custom IAM policies by ARN",
	}
	survey.AskOne(customPrompt, &addCustom)

	var customPolicies []string
	if addCustom {
		for {
			var customArn string
			arnPrompt := &survey.Input{
				Message: "Custom policy ARN (or press Enter to finish):",
				Help:    "Enter a custom IAM policy ARN (e.g., arn:aws:iam::123456789012:policy/MyCustomPolicy)",
			}
			if err := survey.AskOne(arnPrompt, &customArn); err != nil {
				return nil, err
			}

			if customArn == "" {
				break
			}

			// Basic ARN validation
			if !strings.HasPrefix(customArn, "arn:aws:iam::") {
				fmt.Println("⚠️  Invalid ARN format. Please enter a valid IAM policy ARN.")
				continue
			}

			customPolicies = append(customPolicies, customArn)
			policyDetails[customArn] = "Custom policy"
			fmt.Printf("✓ Added custom policy: %s\n", customArn)
		}
	}

	// Generate role name with validation
	defaultRoleName := fmt.Sprintf("genesys-lambda-%s", functionName)
	formattedRoleName, err := validation.ValidateAndFormatName("iam-role", defaultRoleName)
	if err != nil {
		return nil, fmt.Errorf("failed to format role name: %w", err)
	}

	// Ask if they want to customize the role name
	customizeRole := false
	customizePrompt := &survey.Confirm{
		Message: fmt.Sprintf("Use default role name '%s'?", formattedRoleName),
		Default: true,
		Help:    "The IAM role will be created automatically if it doesn't exist",
	}
	survey.AskOne(customizePrompt, &customizeRole)

	roleName := formattedRoleName
	if !customizeRole {
		var rawRoleName string
		roleNamePrompt := &survey.Input{
			Message: "Enter custom role name:",
			Default: formattedRoleName,
			Help:    "Name will be automatically formatted for AWS IAM compatibility",
		}
		if err := survey.AskOne(roleNamePrompt, &rawRoleName); err != nil {
			return nil, err
		}

		// Format the custom role name
		formattedCustomName, err := validation.ValidateAndFormatName("iam-role", rawRoleName)
		if err != nil {
			return nil, fmt.Errorf("invalid role name: %w", err)
		}

		// Show formatting if changed
		if formattedCustomName != rawRoleName {
			fmt.Printf("✓ Role name formatted for AWS IAM: %s → %s\n", rawRoleName, formattedCustomName)
		}

		roleName = formattedCustomName
	}

	fmt.Printf("\n✓ IAM configuration complete\n")
	fmt.Printf("  Role name: %s\n", roleName)
	fmt.Printf("  AWS managed policies: %d\n", len(selectedPolicies))
	fmt.Printf("  Custom policies: %d\n", len(customPolicies))

	return &LambdaIAM{
		RoleName:         roleName,
		RequiredPolicies: selectedPolicies,
		CustomPolicies:   customPolicies,
		AutoManage:       true,
		AutoCleanup:      true,
		PolicyDetails:    policyDetails,
	}, nil
}

// selectDirectory provides an interactive directory browser
func selectDirectory(startDir string) (string, error) {
	currentDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		// Read current directory contents
		entries, err := os.ReadDir(currentDir)
		if err != nil {
			return "", fmt.Errorf("failed to read directory: %w", err)
		}

		// Filter and sort directories
		var options []string
		var paths []string

		// Add parent directory option (unless we're at root)
		if currentDir != "/" && currentDir != filepath.VolumeName(currentDir)+"\\" {
			options = append(options, "📁 .. (parent directory)")
			paths = append(paths, filepath.Dir(currentDir))
		}

		// Add current directory option
		options = append(options, fmt.Sprintf("✅ . (select current: %s)", filepath.Base(currentDir)))
		paths = append(paths, currentDir)

		// Add subdirectories
		var dirs []string
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				dirs = append(dirs, entry.Name())
			}
		}
		sort.Strings(dirs)

		for _, dir := range dirs {
			options = append(options, fmt.Sprintf("📁 %s/", dir))
			paths = append(paths, filepath.Join(currentDir, dir))
		}

		if len(options) == 0 {
			options = append(options, fmt.Sprintf("✅ . (select current: %s)", filepath.Base(currentDir)))
			paths = append(paths, currentDir)
		}

		// Show directory browser
		var selectedIndex int
		prompt := &survey.Select{
			Message: fmt.Sprintf("Select directory (current: %s)", currentDir),
			Options: options,
			Help:    "Navigate with arrow keys, Enter to select, choose '..' to go up",
		}

		// Convert to string for survey (it expects string response)
		var selectedOption string
		if err := survey.AskOne(prompt, &selectedOption); err != nil {
			return "", err
		}

		// Find selected index
		for i, option := range options {
			if option == selectedOption {
				selectedIndex = i
				break
			}
		}

		selectedPath := paths[selectedIndex]

		// If user selected current directory, return it
		if selectedPath == currentDir && strings.HasPrefix(selectedOption, "✅") {
			return selectedPath, nil
		}

		// Otherwise navigate to selected directory
		currentDir = selectedPath
	}
}

// getFunctionName gets and validates the Lambda function name
func (ilc *InteractiveLambdaConfig) getFunctionName() (string, error) {
	var rawName string
	prompt := &survey.Input{
		Message: "Function name:",
		Help:    "Enter any name - it will be automatically formatted for AWS Lambda",
	}

	if err := survey.AskOne(prompt, &rawName, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	// Auto-format the name
	formattedName, err := validation.ValidateAndFormatName("lambda", rawName)
	if err != nil {
		return "", fmt.Errorf("invalid function name: %w", err)
	}

	// Show the user what will be used if it changed
	if formattedName != rawName {
		fmt.Printf("✓ Name formatted for AWS Lambda: %s → %s\n", rawName, formattedName)

		// Confirm with user
		confirm := true
		confirmPrompt := &survey.Confirm{
			Message: fmt.Sprintf("Use formatted name '%s'?", formattedName),
			Default: true,
			Help:    "AWS Lambda requires specific naming rules",
		}
		if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
			return "", err
		}

		if !confirm {
			fmt.Println("Please enter a different name...")
			return ilc.getFunctionName() // Ask again
		}
	}

	return formattedName, nil
}
