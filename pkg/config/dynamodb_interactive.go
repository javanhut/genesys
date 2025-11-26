package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/BurntSushi/toml"
	"github.com/javanhut/genesys/pkg/provider"
)

// DynamoDBTableResource represents a DynamoDB table configuration
type DynamoDBTableResource struct {
	Name               string            `toml:"name"`
	BillingMode        string            `toml:"billing_mode"`
	HashKeyName        string            `toml:"hash_key_name"`
	HashKeyType        string            `toml:"hash_key_type"`
	RangeKeyName       string            `toml:"range_key_name,omitempty"`
	RangeKeyType       string            `toml:"range_key_type,omitempty"`
	ReadCapacityUnits  int64             `toml:"read_capacity_units,omitempty"`
	WriteCapacityUnits int64             `toml:"write_capacity_units,omitempty"`
	EnableStreams      bool              `toml:"enable_streams"`
	StreamViewType     string            `toml:"stream_view_type,omitempty"`
	EnableTTL          bool              `toml:"enable_ttl"`
	TTLAttributeName   string            `toml:"ttl_attribute_name,omitempty"`
	Tags               map[string]string `toml:"tags,omitempty"`
}

// DynamoDBConfig represents a DynamoDB table configuration file
type DynamoDBConfig struct {
	Provider string `toml:"provider"`
	Region   string `toml:"region"`

	Resources struct {
		DynamoDB []DynamoDBTableResource `toml:"dynamodb"`
	} `toml:"resources"`
}

// InteractiveDynamoDBConfig manages interactive DynamoDB table configuration
type InteractiveDynamoDBConfig struct {
	configDir string
}

// NewInteractiveDynamoDBConfig creates a new interactive DynamoDB configuration manager
func NewInteractiveDynamoDBConfig() (*InteractiveDynamoDBConfig, error) {
	ic, err := NewInteractiveConfig()
	if err != nil {
		return nil, err
	}

	return &InteractiveDynamoDBConfig{
		configDir: ic.configDir,
	}, nil
}

// CreateTableConfig creates an interactive DynamoDB table configuration
func (idc *InteractiveDynamoDBConfig) CreateTableConfig() (*provider.DynamoDBTableConfig, string, error) {
	fmt.Println("\nDynamoDB Table Configuration Wizard")
	fmt.Println("====================================")
	fmt.Println("Let's create a DynamoDB table configuration!")
	fmt.Println()

	// Get table name
	tableName, err := idc.getTableName()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get table name: %w", err)
	}

	// Get billing mode
	billingMode, err := idc.getBillingMode()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get billing mode: %w", err)
	}

	config := &provider.DynamoDBTableConfig{
		Name:        tableName,
		BillingMode: billingMode,
		Tags:        make(map[string]string),
	}

	// Get provisioned throughput if needed
	if billingMode == provider.BillingModeProvisioned {
		rcu, wcu, err := idc.getProvisionedCapacity()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get provisioned capacity: %w", err)
		}
		config.ReadCapacityUnits = rcu
		config.WriteCapacityUnits = wcu
	}

	// Get partition key (hash key)
	hashKeyInfo, err := idc.getKeySchema("Partition key (Hash key)", true)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get partition key: %w", err)
	}
	config.HashKey = hashKeyInfo.KeySchema
	config.AttributeDefinitions = append(config.AttributeDefinitions, hashKeyInfo.AttrDefinition)

	// Get sort key (range key) - optional
	var useRangeKey bool
	rangeKeyPrompt := &survey.Confirm{
		Message: "Do you want to add a sort key (range key)?",
		Default: false,
		Help:    "Sort keys allow you to query related items together",
	}
	if err := survey.AskOne(rangeKeyPrompt, &useRangeKey); err != nil {
		return nil, "", err
	}

	if useRangeKey {
		rangeKeyInfo, err := idc.getKeySchema("Sort key (Range key)", false)
		if err != nil {
			return nil, "", fmt.Errorf("failed to get sort key: %w", err)
		}
		config.RangeKey = &rangeKeyInfo.KeySchema
		config.AttributeDefinitions = append(config.AttributeDefinitions, rangeKeyInfo.AttrDefinition)
	}

	// Get stream settings
	enableStreams, streamViewType, err := idc.getStreamSettings()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get stream settings: %w", err)
	}
	config.EnableStreams = enableStreams
	config.StreamViewType = streamViewType

	// Get TTL settings
	enableTTL, ttlAttribute, err := idc.getTTLSettings()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get TTL settings: %w", err)
	}
	config.EnableTTL = enableTTL
	config.TTLAttributeName = ttlAttribute

	// Get tags
	tags, err := idc.getTags()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get tags: %w", err)
	}
	config.Tags = tags

	return config, tableName, nil
}

func (idc *InteractiveDynamoDBConfig) getTableName() (string, error) {
	var tableName string
	prompt := &survey.Input{
		Message: "Table name:",
		Help:    "Enter the DynamoDB table name (3-255 characters, alphanumeric and underscores)",
	}

	err := survey.AskOne(prompt, &tableName, survey.WithValidator(func(ans interface{}) error {
		str := ans.(string)
		if len(str) < 3 || len(str) > 255 {
			return fmt.Errorf("table name must be 3-255 characters")
		}
		for _, c := range str {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.') {
				return fmt.Errorf("table name can only contain alphanumeric characters, underscores, hyphens, and dots")
			}
		}
		return nil
	}))

	return tableName, err
}

func (idc *InteractiveDynamoDBConfig) getBillingMode() (provider.DynamoDBBillingMode, error) {
	var billingModeStr string
	prompt := &survey.Select{
		Message: "Billing mode:",
		Options: []string{
			"On-Demand (PAY_PER_REQUEST)",
			"Provisioned",
		},
		Default: "On-Demand (PAY_PER_REQUEST)",
		Help:    "On-demand: pay per request, no capacity planning. Provisioned: set fixed read/write capacity units.",
	}

	if err := survey.AskOne(prompt, &billingModeStr); err != nil {
		return "", err
	}

	if strings.HasPrefix(billingModeStr, "On-Demand") {
		return provider.BillingModeOnDemand, nil
	}
	return provider.BillingModeProvisioned, nil
}

func (idc *InteractiveDynamoDBConfig) getProvisionedCapacity() (int64, int64, error) {
	var rcuStr, wcuStr string

	rcuPrompt := &survey.Input{
		Message: "Read Capacity Units (RCU):",
		Default: "5",
		Help:    "Number of read capacity units (1 RCU = 1 strongly consistent read/sec for items up to 4KB)",
	}
	if err := survey.AskOne(rcuPrompt, &rcuStr); err != nil {
		return 0, 0, err
	}

	wcuPrompt := &survey.Input{
		Message: "Write Capacity Units (WCU):",
		Default: "5",
		Help:    "Number of write capacity units (1 WCU = 1 write/sec for items up to 1KB)",
	}
	if err := survey.AskOne(wcuPrompt, &wcuStr); err != nil {
		return 0, 0, err
	}

	rcu, err := strconv.ParseInt(rcuStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid RCU value: %w", err)
	}

	wcu, err := strconv.ParseInt(wcuStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid WCU value: %w", err)
	}

	return rcu, wcu, nil
}

// keyInfo holds both key schema and attribute definition info
type keyInfo struct {
	KeySchema       provider.DynamoDBKeySchemaElement
	AttrDefinition  provider.DynamoDBAttributeDefinition
}

func (idc *InteractiveDynamoDBConfig) getKeySchema(keyName string, isHashKey bool) (keyInfo, error) {
	var attrName, attrType string

	namePrompt := &survey.Input{
		Message: fmt.Sprintf("%s attribute name:", keyName),
		Help:    "Name of the attribute to use as the key",
	}
	if err := survey.AskOne(namePrompt, &attrName, survey.WithValidator(survey.Required)); err != nil {
		return keyInfo{}, err
	}

	typePrompt := &survey.Select{
		Message: fmt.Sprintf("%s attribute type:", keyName),
		Options: []string{
			"String (S)",
			"Number (N)",
			"Binary (B)",
		},
		Default: "String (S)",
	}
	if err := survey.AskOne(typePrompt, &attrType); err != nil {
		return keyInfo{}, err
	}

	// Extract attribute type code (S/N/B)
	attrTypeCode := provider.AttributeTypeString
	if strings.HasPrefix(attrType, "Number") {
		attrTypeCode = provider.AttributeTypeNumber
	} else if strings.HasPrefix(attrType, "Binary") {
		attrTypeCode = provider.AttributeTypeBinary
	}

	// Determine key type (HASH or RANGE)
	keyType := provider.KeyTypeHash
	if !isHashKey {
		keyType = provider.KeyTypeRange
	}

	return keyInfo{
		KeySchema: provider.DynamoDBKeySchemaElement{
			AttributeName: attrName,
			KeyType:       keyType,
		},
		AttrDefinition: provider.DynamoDBAttributeDefinition{
			AttributeName: attrName,
			AttributeType: attrTypeCode,
		},
	}, nil
}

func (idc *InteractiveDynamoDBConfig) getStreamSettings() (bool, string, error) {
	var enableStreams bool
	prompt := &survey.Confirm{
		Message: "Enable DynamoDB Streams?",
		Default: false,
		Help:    "Streams capture item-level changes for triggers and replication",
	}
	if err := survey.AskOne(prompt, &enableStreams); err != nil {
		return false, "", err
	}

	if !enableStreams {
		return false, "", nil
	}

	var streamViewType string
	viewPrompt := &survey.Select{
		Message: "Stream view type:",
		Options: []string{
			"NEW_AND_OLD_IMAGES",
			"NEW_IMAGE",
			"OLD_IMAGE",
			"KEYS_ONLY",
		},
		Default: "NEW_AND_OLD_IMAGES",
		Help:    "What data to include in stream records",
	}
	if err := survey.AskOne(viewPrompt, &streamViewType); err != nil {
		return false, "", err
	}

	return true, streamViewType, nil
}

func (idc *InteractiveDynamoDBConfig) getTTLSettings() (bool, string, error) {
	var enableTTL bool
	prompt := &survey.Confirm{
		Message: "Enable Time To Live (TTL)?",
		Default: false,
		Help:    "TTL automatically deletes expired items based on a timestamp attribute",
	}
	if err := survey.AskOne(prompt, &enableTTL); err != nil {
		return false, "", err
	}

	if !enableTTL {
		return false, "", nil
	}

	var ttlAttribute string
	attrPrompt := &survey.Input{
		Message: "TTL attribute name:",
		Default: "ttl",
		Help:    "Name of the attribute containing the expiration timestamp (Unix epoch)",
	}
	if err := survey.AskOne(attrPrompt, &ttlAttribute, survey.WithValidator(survey.Required)); err != nil {
		return false, "", err
	}

	return true, ttlAttribute, nil
}

func (idc *InteractiveDynamoDBConfig) getTags() (map[string]string, error) {
	tags := make(map[string]string)

	var addTags bool
	prompt := &survey.Confirm{
		Message: "Add tags to the table?",
		Default: false,
	}
	if err := survey.AskOne(prompt, &addTags); err != nil {
		return tags, err
	}

	if !addTags {
		return tags, nil
	}

	for {
		var key, value string

		keyPrompt := &survey.Input{
			Message: "Tag key (empty to finish):",
		}
		if err := survey.AskOne(keyPrompt, &key); err != nil {
			return tags, err
		}

		if key == "" {
			break
		}

		valuePrompt := &survey.Input{
			Message: fmt.Sprintf("Tag value for '%s':", key),
		}
		if err := survey.AskOne(valuePrompt, &value); err != nil {
			return tags, err
		}

		tags[key] = value
	}

	return tags, nil
}

// SaveConfig saves the DynamoDB configuration to a TOML file
func (idc *InteractiveDynamoDBConfig) SaveConfig(config *provider.DynamoDBTableConfig, tableName, region string) (string, error) {
	// Create the dynamodb resources directory
	dynamodbDir := filepath.Join(idc.configDir, "resources", "dynamodb")
	if err := os.MkdirAll(dynamodbDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create dynamodb directory: %w", err)
	}

	// Build the config structure
	tomlConfig := DynamoDBConfig{
		Provider: "aws",
		Region:   region,
	}

	resource := DynamoDBTableResource{
		Name:             config.Name,
		BillingMode:      string(config.BillingMode),
		HashKeyName:      config.HashKey.AttributeName,
		HashKeyType:      string(config.HashKey.KeyType),
		EnableStreams:    config.EnableStreams,
		StreamViewType:   config.StreamViewType,
		EnableTTL:        config.EnableTTL,
		TTLAttributeName: config.TTLAttributeName,
		Tags:             config.Tags,
	}

	if config.BillingMode == provider.BillingModeProvisioned {
		resource.ReadCapacityUnits = config.ReadCapacityUnits
		resource.WriteCapacityUnits = config.WriteCapacityUnits
	}

	if config.RangeKey != nil {
		resource.RangeKeyName = config.RangeKey.AttributeName
		resource.RangeKeyType = string(config.RangeKey.KeyType)
	}

	// Get the attribute type from AttributeDefinitions
	for _, attr := range config.AttributeDefinitions {
		if attr.AttributeName == config.HashKey.AttributeName {
			resource.HashKeyType = string(attr.AttributeType)
		}
		if config.RangeKey != nil && attr.AttributeName == config.RangeKey.AttributeName {
			resource.RangeKeyType = string(attr.AttributeType)
		}
	}

	tomlConfig.Resources.DynamoDB = []DynamoDBTableResource{resource}

	// Write to file
	filePath := filepath.Join(dynamodbDir, fmt.Sprintf("%s.toml", tableName))
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(tomlConfig); err != nil {
		return "", fmt.Errorf("failed to encode config: %w", err)
	}

	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return filePath, nil
}

// GetRegion prompts for AWS region selection
func (idc *InteractiveDynamoDBConfig) GetRegion() (string, error) {
	commonRegions := []string{
		"us-east-1",
		"us-east-2",
		"us-west-1",
		"us-west-2",
		"eu-west-1",
		"eu-central-1",
		"ap-southeast-1",
		"ap-northeast-1",
	}

	var region string
	prompt := &survey.Select{
		Message: "AWS Region:",
		Options: commonRegions,
		Default: "us-east-1",
	}

	if err := survey.AskOne(prompt, &region); err != nil {
		return "", err
	}

	return region, nil
}
