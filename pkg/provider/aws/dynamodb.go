package aws

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/javanhut/genesys/pkg/provider"
)

// DynamoDBService implements provider.DynamoDBService for AWS DynamoDB
type DynamoDBService struct {
	provider *AWSProvider
}

// NewDynamoDBService creates a new DynamoDB service
func NewDynamoDBService(p *AWSProvider) *DynamoDBService {
	return &DynamoDBService{provider: p}
}

// makeRequest makes a DynamoDB API request
func (d *DynamoDBService) makeRequest(ctx context.Context, action string, payload interface{}) ([]byte, error) {
	client, err := d.provider.CreateClient("dynamodb")
	if err != nil {
		return nil, fmt.Errorf("failed to create DynamoDB client: %w", err)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("https://dynamodb.%s.amazonaws.com/", d.provider.region)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// DynamoDB uses JSON API
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810."+action)

	// Calculate payload hash
	payloadHash := fmt.Sprintf("%x", sha256.Sum256(body))
	req.Header.Set("x-amz-content-sha256", payloadHash)

	// Add session token if present
	if client.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", client.SessionToken)
	}

	// Sign the request
	if err := client.signRequest(req, body); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp dynamoDBError
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Message != "" {
			return nil, fmt.Errorf("%s: %s", errResp.Type, errResp.Message)
		}
		return nil, fmt.Errorf("DynamoDB %s failed with status %d: %s", action, resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// DynamoDB error response
type dynamoDBError struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
}

// ListTables returns all DynamoDB tables
func (d *DynamoDBService) ListTables(ctx context.Context) ([]*provider.DynamoDBTable, error) {
	var tables []*provider.DynamoDBTable
	var lastTableName string

	for {
		req := map[string]interface{}{}
		if lastTableName != "" {
			req["ExclusiveStartTableName"] = lastTableName
		}

		resp, err := d.makeRequest(ctx, "ListTables", req)
		if err != nil {
			return nil, err
		}

		var result listTablesResponse
		if err := json.Unmarshal(resp, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		// Get details for each table
		for _, tableName := range result.TableNames {
			table, err := d.DescribeTable(ctx, tableName)
			if err != nil {
				// Log error but continue
				continue
			}
			tables = append(tables, table)
		}

		if result.LastEvaluatedTableName == "" {
			break
		}
		lastTableName = result.LastEvaluatedTableName
	}

	return tables, nil
}

type listTablesResponse struct {
	TableNames             []string `json:"TableNames"`
	LastEvaluatedTableName string   `json:"LastEvaluatedTableName"`
}

// DescribeTable returns detailed information about a table
func (d *DynamoDBService) DescribeTable(ctx context.Context, tableName string) (*provider.DynamoDBTable, error) {
	req := map[string]interface{}{
		"TableName": tableName,
	}

	resp, err := d.makeRequest(ctx, "DescribeTable", req)
	if err != nil {
		return nil, err
	}

	var result describeTableResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	table := d.convertTableDescription(&result.Table)
	table.Region = d.provider.region
	return table, nil
}

type describeTableResponse struct {
	Table tableDescription `json:"Table"`
}

type tableDescription struct {
	TableName              string                  `json:"TableName"`
	TableStatus            string                  `json:"TableStatus"`
	TableArn               string                  `json:"TableArn"`
	ItemCount              int64                   `json:"ItemCount"`
	TableSizeBytes         int64                   `json:"TableSizeBytes"`
	CreationDateTime       float64                 `json:"CreationDateTime"`
	KeySchema              []keySchemaElement      `json:"KeySchema"`
	AttributeDefinitions   []attributeDefinition   `json:"AttributeDefinitions"`
	ProvisionedThroughput  *provisionedThroughput  `json:"ProvisionedThroughput"`
	BillingModeSummary     *billingModeSummary     `json:"BillingModeSummary"`
	GlobalSecondaryIndexes []gsiDescription        `json:"GlobalSecondaryIndexes"`
	LocalSecondaryIndexes  []lsiDescription        `json:"LocalSecondaryIndexes"`
	StreamSpecification    *streamSpecification    `json:"StreamSpecification"`
}

type keySchemaElement struct {
	AttributeName string `json:"AttributeName"`
	KeyType       string `json:"KeyType"`
}

type attributeDefinition struct {
	AttributeName string `json:"AttributeName"`
	AttributeType string `json:"AttributeType"`
}

type provisionedThroughput struct {
	ReadCapacityUnits  int64 `json:"ReadCapacityUnits"`
	WriteCapacityUnits int64 `json:"WriteCapacityUnits"`
}

type billingModeSummary struct {
	BillingMode string `json:"BillingMode"`
}

type gsiDescription struct {
	IndexName             string                 `json:"IndexName"`
	KeySchema             []keySchemaElement     `json:"KeySchema"`
	Projection            *projection            `json:"Projection"`
	ProvisionedThroughput *provisionedThroughput `json:"ProvisionedThroughput"`
	IndexStatus           string                 `json:"IndexStatus"`
	ItemCount             int64                  `json:"ItemCount"`
	IndexSizeBytes        int64                  `json:"IndexSizeBytes"`
}

type lsiDescription struct {
	IndexName      string             `json:"IndexName"`
	KeySchema      []keySchemaElement `json:"KeySchema"`
	Projection     *projection        `json:"Projection"`
	ItemCount      int64              `json:"ItemCount"`
	IndexSizeBytes int64              `json:"IndexSizeBytes"`
}

type projection struct {
	ProjectionType   string   `json:"ProjectionType"`
	NonKeyAttributes []string `json:"NonKeyAttributes"`
}

type streamSpecification struct {
	StreamEnabled  bool   `json:"StreamEnabled"`
	StreamViewType string `json:"StreamViewType"`
}

func (d *DynamoDBService) convertTableDescription(desc *tableDescription) *provider.DynamoDBTable {
	table := &provider.DynamoDBTable{
		Name:           desc.TableName,
		Status:         desc.TableStatus,
		ARN:            desc.TableArn,
		ItemCount:      desc.ItemCount,
		TableSizeBytes: desc.TableSizeBytes,
		CreatedAt:      time.Unix(int64(desc.CreationDateTime), 0),
	}

	// Convert key schema
	for _, ks := range desc.KeySchema {
		table.KeySchema = append(table.KeySchema, provider.DynamoDBKeySchemaElement{
			AttributeName: ks.AttributeName,
			KeyType:       provider.DynamoDBKeyType(ks.KeyType),
		})
	}

	// Convert attribute definitions
	for _, ad := range desc.AttributeDefinitions {
		table.AttributeDefinitions = append(table.AttributeDefinitions, provider.DynamoDBAttributeDefinition{
			AttributeName: ad.AttributeName,
			AttributeType: provider.DynamoDBAttributeType(ad.AttributeType),
		})
	}

	// Set billing mode
	if desc.BillingModeSummary != nil {
		table.BillingMode = provider.DynamoDBBillingMode(desc.BillingModeSummary.BillingMode)
	} else if desc.ProvisionedThroughput != nil {
		table.BillingMode = provider.BillingModeProvisioned
	}

	// Convert provisioned throughput
	if desc.ProvisionedThroughput != nil {
		table.ProvisionedThroughput = &provider.DynamoDBProvisionedThroughput{
			ReadCapacityUnits:  desc.ProvisionedThroughput.ReadCapacityUnits,
			WriteCapacityUnits: desc.ProvisionedThroughput.WriteCapacityUnits,
		}
	}

	// Convert GSIs
	for _, gsi := range desc.GlobalSecondaryIndexes {
		index := provider.DynamoDBGlobalSecondaryIndex{
			IndexName:      gsi.IndexName,
			IndexStatus:    gsi.IndexStatus,
			ItemCount:      gsi.ItemCount,
			IndexSizeBytes: gsi.IndexSizeBytes,
		}
		for _, ks := range gsi.KeySchema {
			index.KeySchema = append(index.KeySchema, provider.DynamoDBKeySchemaElement{
				AttributeName: ks.AttributeName,
				KeyType:       provider.DynamoDBKeyType(ks.KeyType),
			})
		}
		if gsi.Projection != nil {
			index.Projection = &provider.DynamoDBProjection{
				ProjectionType:   gsi.Projection.ProjectionType,
				NonKeyAttributes: gsi.Projection.NonKeyAttributes,
			}
		}
		if gsi.ProvisionedThroughput != nil {
			index.ProvisionedThroughput = &provider.DynamoDBProvisionedThroughput{
				ReadCapacityUnits:  gsi.ProvisionedThroughput.ReadCapacityUnits,
				WriteCapacityUnits: gsi.ProvisionedThroughput.WriteCapacityUnits,
			}
		}
		table.GlobalSecondaryIndexes = append(table.GlobalSecondaryIndexes, index)
	}

	// Convert LSIs
	for _, lsi := range desc.LocalSecondaryIndexes {
		index := provider.DynamoDBLocalSecondaryIndex{
			IndexName:      lsi.IndexName,
			ItemCount:      lsi.ItemCount,
			IndexSizeBytes: lsi.IndexSizeBytes,
		}
		for _, ks := range lsi.KeySchema {
			index.KeySchema = append(index.KeySchema, provider.DynamoDBKeySchemaElement{
				AttributeName: ks.AttributeName,
				KeyType:       provider.DynamoDBKeyType(ks.KeyType),
			})
		}
		if lsi.Projection != nil {
			index.Projection = &provider.DynamoDBProjection{
				ProjectionType:   lsi.Projection.ProjectionType,
				NonKeyAttributes: lsi.Projection.NonKeyAttributes,
			}
		}
		table.LocalSecondaryIndexes = append(table.LocalSecondaryIndexes, index)
	}

	// Stream settings
	if desc.StreamSpecification != nil {
		table.StreamEnabled = desc.StreamSpecification.StreamEnabled
		table.StreamViewType = desc.StreamSpecification.StreamViewType
	}

	return table
}

// CreateTable creates a new DynamoDB table
func (d *DynamoDBService) CreateTable(ctx context.Context, config *provider.DynamoDBTableConfig) (*provider.DynamoDBTable, error) {
	req := map[string]interface{}{
		"TableName": config.Name,
	}

	// Key schema
	keySchema := []map[string]string{
		{
			"AttributeName": config.HashKey.AttributeName,
			"KeyType":       string(config.HashKey.KeyType),
		},
	}
	if config.RangeKey != nil {
		keySchema = append(keySchema, map[string]string{
			"AttributeName": config.RangeKey.AttributeName,
			"KeyType":       string(config.RangeKey.KeyType),
		})
	}
	req["KeySchema"] = keySchema

	// Attribute definitions
	var attrDefs []map[string]string
	for _, ad := range config.AttributeDefinitions {
		attrDefs = append(attrDefs, map[string]string{
			"AttributeName": ad.AttributeName,
			"AttributeType": string(ad.AttributeType),
		})
	}
	req["AttributeDefinitions"] = attrDefs

	// Billing mode
	if config.BillingMode == provider.BillingModeOnDemand {
		req["BillingMode"] = "PAY_PER_REQUEST"
	} else {
		req["BillingMode"] = "PROVISIONED"
		req["ProvisionedThroughput"] = map[string]int64{
			"ReadCapacityUnits":  config.ReadCapacityUnits,
			"WriteCapacityUnits": config.WriteCapacityUnits,
		}
	}

	// Stream specification
	if config.EnableStreams {
		streamViewType := config.StreamViewType
		if streamViewType == "" {
			streamViewType = "NEW_AND_OLD_IMAGES"
		}
		req["StreamSpecification"] = map[string]interface{}{
			"StreamEnabled":  true,
			"StreamViewType": streamViewType,
		}
	}

	// Tags
	if len(config.Tags) > 0 {
		var tags []map[string]string
		for k, v := range config.Tags {
			tags = append(tags, map[string]string{
				"Key":   k,
				"Value": v,
			})
		}
		req["Tags"] = tags
	}

	resp, err := d.makeRequest(ctx, "CreateTable", req)
	if err != nil {
		return nil, err
	}

	var result describeTableResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	table := d.convertTableDescription(&result.Table)
	table.Region = d.provider.region

	// Enable TTL if requested
	if config.EnableTTL && config.TTLAttributeName != "" {
		if err := d.UpdateTTL(ctx, config.Name, true, config.TTLAttributeName); err != nil {
			// Log but don't fail - TTL can be enabled later
		}
	}

	return table, nil
}

// DeleteTable deletes a DynamoDB table
func (d *DynamoDBService) DeleteTable(ctx context.Context, tableName string) error {
	req := map[string]interface{}{
		"TableName": tableName,
	}

	_, err := d.makeRequest(ctx, "DeleteTable", req)
	return err
}

// UpdateTable updates a DynamoDB table
func (d *DynamoDBService) UpdateTable(ctx context.Context, tableName string, config *provider.DynamoDBTableConfig) error {
	req := map[string]interface{}{
		"TableName": tableName,
	}

	// Update billing mode and throughput
	if config.BillingMode == provider.BillingModeOnDemand {
		req["BillingMode"] = "PAY_PER_REQUEST"
	} else if config.BillingMode == provider.BillingModeProvisioned {
		req["BillingMode"] = "PROVISIONED"
		req["ProvisionedThroughput"] = map[string]int64{
			"ReadCapacityUnits":  config.ReadCapacityUnits,
			"WriteCapacityUnits": config.WriteCapacityUnits,
		}
	}

	_, err := d.makeRequest(ctx, "UpdateTable", req)
	return err
}

// ScanTable scans items from a table
func (d *DynamoDBService) ScanTable(ctx context.Context, tableName string, limit int64, exclusiveStartKey map[string]interface{}) (*provider.DynamoDBScanResult, error) {
	req := map[string]interface{}{
		"TableName": tableName,
	}

	if limit > 0 {
		req["Limit"] = limit
	}

	if len(exclusiveStartKey) > 0 {
		req["ExclusiveStartKey"] = exclusiveStartKey
	}

	resp, err := d.makeRequest(ctx, "Scan", req)
	if err != nil {
		return nil, err
	}

	var result scanResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	scanResult := &provider.DynamoDBScanResult{
		Count:            result.Count,
		ScannedCount:     result.ScannedCount,
		LastEvaluatedKey: result.LastEvaluatedKey,
	}

	for _, item := range result.Items {
		scanResult.Items = append(scanResult.Items, provider.DynamoDBItem{
			Attributes: d.convertDynamoDBItem(item),
		})
	}

	return scanResult, nil
}

type scanResponse struct {
	Items            []map[string]interface{} `json:"Items"`
	Count            int64                    `json:"Count"`
	ScannedCount     int64                    `json:"ScannedCount"`
	LastEvaluatedKey map[string]interface{}   `json:"LastEvaluatedKey"`
}

// convertDynamoDBItem converts DynamoDB item format to simple map
func (d *DynamoDBService) convertDynamoDBItem(item map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range item {
		result[key] = d.convertDynamoDBValue(value)
	}
	return result
}

// convertDynamoDBValue converts a DynamoDB value to a Go value
func (d *DynamoDBService) convertDynamoDBValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	valueMap, ok := value.(map[string]interface{})
	if !ok {
		return value
	}

	// DynamoDB uses typed values like {"S": "string"}, {"N": "123"}
	for typeKey, typeValue := range valueMap {
		switch typeKey {
		case "S":
			return typeValue
		case "N":
			if numStr, ok := typeValue.(string); ok {
				// Try to parse as int first, then float
				if i, err := strconv.ParseInt(numStr, 10, 64); err == nil {
					return i
				}
				if f, err := strconv.ParseFloat(numStr, 64); err == nil {
					return f
				}
				return numStr
			}
			return typeValue
		case "B":
			return typeValue
		case "BOOL":
			return typeValue
		case "NULL":
			return nil
		case "L":
			if list, ok := typeValue.([]interface{}); ok {
				var result []interface{}
				for _, item := range list {
					result = append(result, d.convertDynamoDBValue(item))
				}
				return result
			}
		case "M":
			if m, ok := typeValue.(map[string]interface{}); ok {
				return d.convertDynamoDBItem(m)
			}
		case "SS":
			return typeValue
		case "NS":
			return typeValue
		case "BS":
			return typeValue
		}
	}

	return value
}

// convertGoValueToDynamoDB converts a Go value to DynamoDB format
func (d *DynamoDBService) convertGoValueToDynamoDB(value interface{}) map[string]interface{} {
	if value == nil {
		return map[string]interface{}{"NULL": true}
	}

	switch v := value.(type) {
	case string:
		return map[string]interface{}{"S": v}
	case int, int32, int64, float32, float64:
		return map[string]interface{}{"N": fmt.Sprintf("%v", v)}
	case bool:
		return map[string]interface{}{"BOOL": v}
	case []byte:
		return map[string]interface{}{"B": v}
	case []interface{}:
		var list []interface{}
		for _, item := range v {
			list = append(list, d.convertGoValueToDynamoDB(item))
		}
		return map[string]interface{}{"L": list}
	case map[string]interface{}:
		m := make(map[string]interface{})
		for key, val := range v {
			m[key] = d.convertGoValueToDynamoDB(val)
		}
		return map[string]interface{}{"M": m}
	default:
		return map[string]interface{}{"S": fmt.Sprintf("%v", v)}
	}
}

// GetItem retrieves a single item from a table
func (d *DynamoDBService) GetItem(ctx context.Context, tableName string, key map[string]interface{}) (*provider.DynamoDBItem, error) {
	// Convert key to DynamoDB format
	dynamoKey := make(map[string]interface{})
	for k, v := range key {
		dynamoKey[k] = d.convertGoValueToDynamoDB(v)
	}

	req := map[string]interface{}{
		"TableName": tableName,
		"Key":       dynamoKey,
	}

	resp, err := d.makeRequest(ctx, "GetItem", req)
	if err != nil {
		return nil, err
	}

	var result getItemResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("item not found")
	}

	return &provider.DynamoDBItem{
		Attributes: d.convertDynamoDBItem(result.Item),
	}, nil
}

type getItemResponse struct {
	Item map[string]interface{} `json:"Item"`
}

// PutItem inserts or updates an item in a table
func (d *DynamoDBService) PutItem(ctx context.Context, tableName string, item map[string]interface{}) error {
	// Convert item to DynamoDB format
	dynamoItem := make(map[string]interface{})
	for k, v := range item {
		dynamoItem[k] = d.convertGoValueToDynamoDB(v)
	}

	req := map[string]interface{}{
		"TableName": tableName,
		"Item":      dynamoItem,
	}

	_, err := d.makeRequest(ctx, "PutItem", req)
	return err
}

// DeleteItem removes an item from a table
func (d *DynamoDBService) DeleteItem(ctx context.Context, tableName string, key map[string]interface{}) error {
	// Convert key to DynamoDB format
	dynamoKey := make(map[string]interface{})
	for k, v := range key {
		dynamoKey[k] = d.convertGoValueToDynamoDB(v)
	}

	req := map[string]interface{}{
		"TableName": tableName,
		"Key":       dynamoKey,
	}

	_, err := d.makeRequest(ctx, "DeleteItem", req)
	return err
}

// UpdateTTL enables or disables TTL on a table
func (d *DynamoDBService) UpdateTTL(ctx context.Context, tableName string, enabled bool, attributeName string) error {
	req := map[string]interface{}{
		"TableName": tableName,
		"TimeToLiveSpecification": map[string]interface{}{
			"Enabled":       enabled,
			"AttributeName": attributeName,
		},
	}

	_, err := d.makeRequest(ctx, "UpdateTimeToLive", req)
	return err
}

// DescribeTTL returns the TTL settings for a table
func (d *DynamoDBService) DescribeTTL(ctx context.Context, tableName string) (bool, string, error) {
	req := map[string]interface{}{
		"TableName": tableName,
	}

	resp, err := d.makeRequest(ctx, "DescribeTimeToLive", req)
	if err != nil {
		return false, "", err
	}

	var result describeTTLResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return false, "", fmt.Errorf("failed to parse response: %w", err)
	}

	enabled := result.TimeToLiveDescription.TimeToLiveStatus == "ENABLED"
	return enabled, result.TimeToLiveDescription.AttributeName, nil
}

type describeTTLResponse struct {
	TimeToLiveDescription struct {
		TimeToLiveStatus string `json:"TimeToLiveStatus"`
		AttributeName    string `json:"AttributeName"`
	} `json:"TimeToLiveDescription"`
}
