package intent

import (
	"fmt"
	"sort"
	"strings"
)

// Intent represents a parsed user intent
type Intent struct {
	Type       IntentType        `json:"type"`
	Name       string            `json:"name,omitempty"`
	Action     Action            `json:"action"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Modifiers  []string          `json:"modifiers,omitempty"`
}

// IntentType represents the type of intent
type IntentType string

const (
	IntentBucket     IntentType = "bucket"
	IntentNetwork    IntentType = "network"
	IntentFunction   IntentType = "function"
	IntentStaticSite IntentType = "static-site"
	IntentDatabase   IntentType = "database"
	IntentAPI        IntentType = "api"
	IntentWebapp     IntentType = "webapp"
)

// Action represents what to do with the resource
type Action string

const (
	ActionCreate Action = "create"
	ActionAdopt  Action = "adopt"
	ActionModify Action = "modify"
	ActionDelete Action = "delete"
	ActionShow   Action = "show"
)

// Parser handles intent parsing
type Parser struct {
	aliases map[string]IntentType
}

// NewParser creates a new intent parser
func NewParser() *Parser {
	return &Parser{
		aliases: map[string]IntentType{
			"bucket":      IntentBucket,
			"storage":     IntentBucket,
			"s3":          IntentBucket,
			"network":     IntentNetwork,
			"vpc":         IntentNetwork,
			"net":         IntentNetwork,
			"function":    IntentFunction,
			"lambda":      IntentFunction,
			"fn":          IntentFunction,
			"static-site": IntentStaticSite,
			"website":     IntentStaticSite,
			"site":        IntentStaticSite,
			"database":    IntentDatabase,
			"db":          IntentDatabase,
			"postgres":    IntentDatabase,
			"mysql":       IntentDatabase,
			"api":         IntentAPI,
			"webapp":      IntentWebapp,
			"app":         IntentWebapp,
		},
	}
}

// Parse parses command line arguments into an intent
func (p *Parser) Parse(args []string) (*Intent, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no intent specified")
	}

	intent := &Intent{
		Parameters: make(map[string]string),
		Modifiers:  make([]string, 0),
	}

	// First argument is the intent type
	intentType, ok := p.aliases[strings.ToLower(args[0])]
	if !ok {
		return nil, fmt.Errorf("unknown intent type: %s", args[0])
	}
	intent.Type = intentType

	// Default action is create
	intent.Action = ActionCreate

	// Parse remaining arguments
	for i := 1; i < len(args); i++ {
		arg := args[i]

		// Check if it's a key=value pair
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				intent.Parameters[parts[0]] = parts[1]
			}
		} else if strings.HasPrefix(arg, "--") {
			// Long flag
			key := strings.TrimPrefix(arg, "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				// Has value
				intent.Parameters[key] = args[i+1]
				i++ // Skip next arg
			} else {
				// Boolean flag
				intent.Parameters[key] = "true"
			}
		} else if strings.HasPrefix(arg, "-") {
			// Single dash flags
			key := strings.TrimPrefix(arg, "-")
			intent.Modifiers = append(intent.Modifiers, key)
		} else {
			// First non-flag argument is usually the name
			if intent.Name == "" {
				intent.Name = arg
			} else {
				// Additional arguments as modifiers
				intent.Modifiers = append(intent.Modifiers, arg)
			}
		}
	}

	// Apply intent-specific parsing
	if err := p.parseIntentSpecific(intent); err != nil {
		return nil, err
	}

	return intent, nil
}

// parseIntentSpecific applies intent-specific parsing logic
func (p *Parser) parseIntentSpecific(intent *Intent) error {
	switch intent.Type {
	case IntentBucket:
		return p.parseBucketIntent(intent)
	case IntentNetwork:
		return p.parseNetworkIntent(intent)
	case IntentFunction:
		return p.parseFunctionIntent(intent)
	case IntentStaticSite:
		return p.parseStaticSiteIntent(intent)
	case IntentDatabase:
		return p.parseDatabaseIntent(intent)
	case IntentAPI:
		return p.parseAPIIntent(intent)
	case IntentWebapp:
		return p.parseWebappIntent(intent)
	}
	return nil
}

// parseBucketIntent handles bucket-specific parsing
func (p *Parser) parseBucketIntent(intent *Intent) error {
	// Set defaults for bucket (only if not already set by user)
	if _, exists := intent.Parameters["versioning"]; !exists {
		intent.Parameters["versioning"] = "true"
	}
	if _, exists := intent.Parameters["encryption"]; !exists {
		intent.Parameters["encryption"] = "true"
	}
	if _, exists := intent.Parameters["public"]; !exists {
		intent.Parameters["public"] = "false"
	}

	// Validate bucket name if provided
	if intent.Name != "" {
		if !isValidBucketName(intent.Name) {
			return fmt.Errorf("invalid bucket name: %s", intent.Name)
		}
	}

	return nil
}

// parseNetworkIntent handles network-specific parsing
func (p *Parser) parseNetworkIntent(intent *Intent) error {
	// Set default CIDR if not provided
	if _, exists := intent.Parameters["cidr"]; !exists {
		intent.Parameters["cidr"] = "10.0.0.0/16"
	}

	// Default to creating public and private subnets
	if _, exists := intent.Parameters["subnets"]; !exists {
		intent.Parameters["subnets"] = "public,private"
	}

	return nil
}

// parseFunctionIntent handles function-specific parsing
func (p *Parser) parseFunctionIntent(intent *Intent) error {
	// Set default runtime
	if _, exists := intent.Parameters["runtime"]; !exists {
		intent.Parameters["runtime"] = "python3.11"
	}

	// Set default memory
	if _, exists := intent.Parameters["memory"]; !exists {
		intent.Parameters["memory"] = "256"
	}

	// Set default timeout
	if _, exists := intent.Parameters["timeout"]; !exists {
		intent.Parameters["timeout"] = "60"
	}

	// Default handler
	if _, exists := intent.Parameters["handler"]; !exists {
		intent.Parameters["handler"] = "main.handler"
	}

	return nil
}

// parseStaticSiteIntent handles static site-specific parsing
func (p *Parser) parseStaticSiteIntent(intent *Intent) error {
	// Enable CDN by default
	if _, exists := intent.Parameters["cdn"]; !exists {
		intent.Parameters["cdn"] = "true"
	}

	// Enable HTTPS by default
	if _, exists := intent.Parameters["https"]; !exists {
		intent.Parameters["https"] = "true"
	}

	// Default index document
	if _, exists := intent.Parameters["index"]; !exists {
		intent.Parameters["index"] = "index.html"
	}

	return nil
}

// parseDatabaseIntent handles database-specific parsing
func (p *Parser) parseDatabaseIntent(intent *Intent) error {
	// Set default engine
	if _, exists := intent.Parameters["engine"]; !exists {
		intent.Parameters["engine"] = "postgres"
	}

	// Set default size
	if _, exists := intent.Parameters["size"]; !exists {
		intent.Parameters["size"] = "small"
	}

	// Set default storage
	if _, exists := intent.Parameters["storage"]; !exists {
		intent.Parameters["storage"] = "20"
	}

	// Enable backups by default
	if _, exists := intent.Parameters["backup"]; !exists {
		intent.Parameters["backup"] = "true"
	}

	return nil
}

// parseAPIIntent handles API-specific parsing
func (p *Parser) parseAPIIntent(intent *Intent) error {
	// Default to HTTP API
	if _, exists := intent.Parameters["type"]; !exists {
		intent.Parameters["type"] = "http"
	}

	// Default runtime for API functions
	if _, exists := intent.Parameters["runtime"]; !exists {
		intent.Parameters["runtime"] = "python3.11"
	}

	return nil
}

// parseWebappIntent handles webapp-specific parsing
func (p *Parser) parseWebappIntent(intent *Intent) error {
	// Default instance type
	if _, exists := intent.Parameters["type"]; !exists {
		intent.Parameters["type"] = "medium"
	}

	// Default to auto-scaling
	if _, exists := intent.Parameters["scaling"]; !exists {
		intent.Parameters["scaling"] = "auto"
	}

	// Enable load balancer by default
	if _, exists := intent.Parameters["lb"]; !exists {
		intent.Parameters["lb"] = "true"
	}

	return nil
}

// isValidBucketName validates bucket naming rules
func isValidBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}

	// Basic validation - can be extended
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '.') {
			return false
		}
	}

	return true
}

// ToHumanReadable returns a human-readable description of the intent
func (i *Intent) ToHumanReadable() string {
	var parts []string

	switch i.Action {
	case ActionCreate:
		parts = append(parts, fmt.Sprintf("Create %s", i.Type))
	case ActionAdopt:
		parts = append(parts, fmt.Sprintf("Adopt existing %s", i.Type))
	case ActionModify:
		parts = append(parts, fmt.Sprintf("Modify %s", i.Type))
	case ActionDelete:
		parts = append(parts, fmt.Sprintf("Delete %s", i.Type))
	}

	if i.Name != "" {
		parts = append(parts, fmt.Sprintf("named '%s'", i.Name))
	}

	// Add key parameters (sorted for consistent output)
	var params []string
	var keys []string
	for key := range i.Parameters {
		keys = append(keys, key)
	}

	// Sort keys for consistent ordering
	sort.Strings(keys)

	for _, key := range keys {
		value := i.Parameters[key]
		if value != "true" && value != "" {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		} else if value == "true" {
			params = append(params, key)
		}
	}

	if len(params) > 0 {
		parts = append(parts, fmt.Sprintf("with %s", strings.Join(params, ", ")))
	}

	return strings.Join(parts, " ")
}

// Examples of valid intents:
// bucket my-bucket
// bucket my-bucket --versioning true --encryption true
// network vpc-prod --cidr 10.0.0.0/16
// function api-handler --runtime python3.11 --memory 512
// static-site --domain example.com --cdn true
// database prod-db --engine postgres --size large
// api my-api --runtime nodejs18
// webapp my-app --type large --scaling auto
