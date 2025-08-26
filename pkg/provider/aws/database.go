package aws

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/javanhut/genesys/pkg/provider"
)

// DatabaseService implements AWS RDS operations using direct API calls
type DatabaseService struct {
	provider *AWSProvider
}

// NewDatabaseService creates a new database service
func NewDatabaseService(p *AWSProvider) *DatabaseService {
	return &DatabaseService{
		provider: p,
	}
}

// RDS API response structures
type CreateDBInstanceResponse struct {
	XMLName    xml.Name `xml:"CreateDBInstanceResponse"`
	DBInstance RDSInstance `xml:"CreateDBInstanceResult>DBInstance"`
}

type DescribeDBInstancesResponse struct {
	XMLName     xml.Name `xml:"DescribeDBInstancesResponse"`
	DBInstances struct {
		Items []RDSInstance `xml:"DBInstance"`
	} `xml:"DescribeDBInstancesResult>DBInstances"`
}

type RDSInstance struct {
	DBInstanceIdentifier string `xml:"DBInstanceIdentifier"`
	DBInstanceClass      string `xml:"DBInstanceClass"`
	Engine               string `xml:"Engine"`
	EngineVersion        string `xml:"EngineVersion"`
	DBInstanceStatus     string `xml:"DBInstanceStatus"`
	Endpoint             struct {
		Address string `xml:"Address"`
		Port    int    `xml:"Port"`
	} `xml:"Endpoint"`
	MultiAZ             bool   `xml:"MultiAZ"`
	AllocatedStorage    int    `xml:"AllocatedStorage"`
	InstanceCreateTime  string `xml:"InstanceCreateTime"`
	BackupRetentionPeriod int  `xml:"BackupRetentionPeriod"`
	PreferredBackupWindow string `xml:"PreferredBackupWindow"`
}

// CreateDatabase creates a new RDS instance
func (d *DatabaseService) CreateDatabase(ctx context.Context, config *provider.DatabaseConfig) (*provider.Database, error) {
	client, err := d.provider.CreateClient("rds")
	if err != nil {
		return nil, fmt.Errorf("failed to create RDS client: %w", err)
	}

	// Map database size to instance class
	dbInstanceClass := d.mapDatabaseSize(string(config.Size))
	
	// Build parameters
	params := map[string]string{
		"Action":               "CreateDBInstance",
		"Version":              "2014-10-31",
		"DBInstanceIdentifier": config.Name,
		"DBInstanceClass":      dbInstanceClass,
		"Engine":               config.Engine,
		"EngineVersion":        config.Version,
		"AllocatedStorage":     fmt.Sprintf("%d", config.Storage),
		"MultiAZ":              fmt.Sprintf("%t", config.MultiAZ),
		"MasterUsername":       "admin", // Default username
		"MasterUserPassword":   "TempPassword123!", // Should be configurable
	}

	// Add backup configuration
	if config.BackupConfig != nil {
		params["BackupRetentionPeriod"] = fmt.Sprintf("%d", config.BackupConfig.RetentionDays)
		params["PreferredBackupWindow"] = config.BackupConfig.Window
	}

	// Add tags
	tagIndex := 1
	for key, value := range config.Tags {
		params[fmt.Sprintf("Tags.member.%d.Key", tagIndex)] = key
		params[fmt.Sprintf("Tags.member.%d.Value", tagIndex)] = value
		tagIndex++
	}

	// Make the request
	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("CreateDBInstance failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var createResp CreateDBInstanceResponse
	if err := xml.Unmarshal(body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to provider database
	return d.convertToProviderDatabase(createResp.DBInstance, config.Tags), nil
}

// GetDatabase retrieves a database by identifier
func (d *DatabaseService) GetDatabase(ctx context.Context, id string) (*provider.Database, error) {
	client, err := d.provider.CreateClient("rds")
	if err != nil {
		return nil, fmt.Errorf("failed to create RDS client: %w", err)
	}

	params := map[string]string{
		"Action":               "DescribeDBInstances",
		"Version":              "2014-10-31",
		"DBInstanceIdentifier": id,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("DescribeDBInstances failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var descResp DescribeDBInstancesResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	for _, instance := range descResp.DBInstances.Items {
		if instance.DBInstanceIdentifier == id {
			return d.convertToProviderDatabase(instance, nil), nil
		}
	}

	return nil, fmt.Errorf("database %s not found", id)
}

// UpdateDatabase updates a database configuration
func (d *DatabaseService) UpdateDatabase(ctx context.Context, id string, config *provider.DatabaseConfig) error {
	client, err := d.provider.CreateClient("rds")
	if err != nil {
		return fmt.Errorf("failed to create RDS client: %w", err)
	}

	params := map[string]string{
		"Action":               "ModifyDBInstance",
		"Version":              "2014-10-31",
		"DBInstanceIdentifier": id,
		"ApplyImmediately":     "true",
	}

	// Only update size if provided
	if config.Size != "" {
		params["DBInstanceClass"] = d.mapDatabaseSize(string(config.Size))
	}

	// Only update storage if provided
	if config.Storage > 0 {
		params["AllocatedStorage"] = fmt.Sprintf("%d", config.Storage)
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return fmt.Errorf("failed to modify database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return fmt.Errorf("ModifyDBInstance failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteDatabase deletes a database instance
func (d *DatabaseService) DeleteDatabase(ctx context.Context, id string) error {
	client, err := d.provider.CreateClient("rds")
	if err != nil {
		return fmt.Errorf("failed to create RDS client: %w", err)
	}

	params := map[string]string{
		"Action":               "DeleteDBInstance",
		"Version":              "2014-10-31",
		"DBInstanceIdentifier": id,
		"SkipFinalSnapshot":    "true", // Skip final snapshot for simplicity
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return fmt.Errorf("DeleteDBInstance failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DiscoverDatabases discovers existing database instances
func (d *DatabaseService) DiscoverDatabases(ctx context.Context) ([]*provider.Database, error) {
	client, err := d.provider.CreateClient("rds")
	if err != nil {
		return nil, fmt.Errorf("failed to create RDS client: %w", err)
	}

	params := map[string]string{
		"Action":  "DescribeDBInstances",
		"Version": "2014-10-31",
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe databases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("DescribeDBInstances failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var descResp DescribeDBInstancesResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var databases []*provider.Database
	for _, instance := range descResp.DBInstances.Items {
		databases = append(databases, d.convertToProviderDatabase(instance, nil))
	}

	return databases, nil
}

// AdoptDatabase adopts an existing database into management
func (d *DatabaseService) AdoptDatabase(ctx context.Context, id string) (*provider.Database, error) {
	return d.GetDatabase(ctx, id)
}

// Helper methods

func (d *DatabaseService) mapDatabaseSize(size string) string {
	switch size {
	case "small":
		return "db.t3.micro"
	case "medium":
		return "db.t3.small"
	case "large":
		return "db.t3.medium"
	case "xlarge":
		return "db.t3.large"
	default:
		return "db.t3.small"
	}
}

func (d *DatabaseService) reverseMapDatabaseSize(instanceClass string) string {
	switch instanceClass {
	case "db.t3.micro":
		return "small"
	case "db.t3.small":
		return "medium"
	case "db.t3.medium":
		return "large"
	case "db.t3.large":
		return "xlarge"
	default:
		return "medium"
	}
}

func (d *DatabaseService) convertToProviderDatabase(rdsInstance RDSInstance, tags map[string]string) *provider.Database {
	createdAt := time.Now()
	if rdsInstance.InstanceCreateTime != "" {
		if t, err := time.Parse(time.RFC3339, rdsInstance.InstanceCreateTime); err == nil {
			createdAt = t
		}
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	return &provider.Database{
		ID:        rdsInstance.DBInstanceIdentifier,
		Name:      rdsInstance.DBInstanceIdentifier,
		Engine:    rdsInstance.Engine,
		Version:   rdsInstance.EngineVersion,
		Size:      provider.DatabaseSize(d.reverseMapDatabaseSize(rdsInstance.DBInstanceClass)),
		Storage:   rdsInstance.AllocatedStorage,
		MultiAZ:   rdsInstance.MultiAZ,
		Endpoint:  rdsInstance.Endpoint.Address,
		Port:      rdsInstance.Endpoint.Port,
		Tags:      tags,
		CreatedAt: createdAt,
		ProviderData: map[string]interface{}{
			"Status":                  rdsInstance.DBInstanceStatus,
			"BackupRetentionPeriod":   rdsInstance.BackupRetentionPeriod,
			"PreferredBackupWindow":   rdsInstance.PreferredBackupWindow,
		},
	}
}