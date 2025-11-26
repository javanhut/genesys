package aws

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/javanhut/genesys/pkg/provider"
)

// StorageService implements AWS S3 operations using direct API calls
type StorageService struct {
	provider *AWSProvider
}

// NewStorageService creates a new storage service
func NewStorageService(p *AWSProvider) *StorageService {
	return &StorageService{
		provider: p,
	}
}

// createClientForBucket creates an S3 client configured for the bucket's region.
// This is necessary because S3 buckets must be accessed using their specific regional endpoint.
// Accessing a bucket in a different region results in a PermanentRedirect error.
func (s *StorageService) createClientForBucket(ctx context.Context, bucketName string) (*AWSClient, error) {
	// First, get the bucket's region
	bucketRegion, err := s.GetBucketRegion(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to determine bucket region: %w", err)
	}

	// Create a client for the bucket's region
	client, err := NewAWSClient(bucketRegion, "s3")
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client for region %s: %w", bucketRegion, err)
	}

	return client, nil
}

// S3 API response structures
type ListAllMyBucketsResult struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Buckets struct {
		Bucket []S3Bucket `xml:"Bucket"`
	} `xml:"Buckets"`
}

type S3Bucket struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

type BucketLocationConstraint struct {
	XMLName            xml.Name `xml:"LocationConstraint"`
	LocationConstraint string   `xml:",chardata"`
}

// encodeS3Key properly URL-encodes an S3 object key for use in HTTP requests.
// It encodes each path segment individually while preserving forward slashes as delimiters.
// This handles special characters like spaces, plus signs, ampersands, etc.
func encodeS3Key(key string) string {
	segments := strings.Split(key, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

// CreateBucket creates a new S3 bucket
func (s *StorageService) CreateBucket(ctx context.Context, config *provider.BucketConfig) (*provider.Bucket, error) {
	client, err := s.provider.CreateClient("s3")
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Create the bucket
	endpoint := fmt.Sprintf("/%s", config.Name)

	var body []byte
	if s.provider.region != "us-east-1" {
		// Need to specify location constraint for regions other than us-east-1
		locationXML := fmt.Sprintf(`<CreateBucketConfiguration><LocationConstraint>%s</LocationConstraint></CreateBucketConfiguration>`, s.provider.region)
		body = []byte(locationXML)
	}

	resp, err := client.Request("PUT", endpoint, nil, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return nil, fmt.Errorf("CreateBucket failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Configure versioning if requested
	if config.Versioning {
		if err := s.setBucketVersioning(client, config.Name, true); err != nil {
			return nil, fmt.Errorf("failed to enable versioning: %w", err)
		}
	}

	// Configure encryption if requested
	if config.Encryption {
		if err := s.setBucketEncryption(client, config.Name); err != nil {
			return nil, fmt.Errorf("failed to enable encryption: %w", err)
		}
	}

	// Set bucket tags
	if len(config.Tags) > 0 {
		if err := s.setBucketTags(client, config.Name, config.Tags); err != nil {
			return nil, fmt.Errorf("failed to set bucket tags: %w", err)
		}
	}

	// Return the created bucket
	return &provider.Bucket{
		Name:         config.Name,
		Region:       s.provider.region,
		Versioning:   config.Versioning,
		Encryption:   config.Encryption,
		PublicAccess: config.PublicAccess,
		Tags:         config.Tags,
		CreatedAt:    time.Now(),
	}, nil
}

// GetBucket retrieves information about a bucket
func (s *StorageService) GetBucket(ctx context.Context, name string) (*provider.Bucket, error) {
	client, err := s.provider.CreateClient("s3")
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Check if bucket exists by trying to get its location
	endpoint := fmt.Sprintf("/%s", name)
	params := map[string]string{"location": ""}
	resp, err := client.Request("GET", endpoint, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket location: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("bucket %s not found", name)
	}

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return nil, fmt.Errorf("GetBucketLocation failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Get bucket versioning status
	versioning, err := s.getBucketVersioning(client, name)
	if err != nil {
		versioning = false // Default to false if we can't determine
	}

	// Get bucket encryption status
	encryption, err := s.getBucketEncryption(client, name)
	if err != nil {
		encryption = false // Default to false if we can't determine
	}

	// Get bucket tags
	tags, err := s.getBucketTags(client, name)
	if err != nil {
		tags = make(map[string]string) // Default to empty if we can't get tags
	}

	return &provider.Bucket{
		Name:         name,
		Region:       s.provider.region,
		Versioning:   versioning,
		Encryption:   encryption,
		PublicAccess: false, // Default to private
		Tags:         tags,
		CreatedAt:    time.Now(), // We don't have creation time from basic API
	}, nil
}

// DeleteBucket deletes a bucket, automatically emptying it first if necessary
func (s *StorageService) DeleteBucket(ctx context.Context, name string) error {
	return s.DeleteBucketWithOptions(ctx, name, false)
}

// DeleteBucketWithOptions deletes a bucket with advanced options
func (s *StorageService) DeleteBucketWithOptions(ctx context.Context, name string, forceDelete bool) error {
	// Use bucket-specific client to handle cross-region bucket access
	client, err := s.createClientForBucket(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Try to delete the bucket first
	endpoint := fmt.Sprintf("/%s", name)
	resp, err := client.Request("DELETE", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}
	defer resp.Body.Close()

	// If successful, we're done
	if resp.StatusCode == 204 {
		return nil
	}

	// If bucket is not empty, try to empty it first
	if resp.StatusCode == 409 {
		responseBody, _ := ReadResponse(resp)
		if strings.Contains(string(responseBody), "BucketNotEmpty") {
			fmt.Printf("Bucket is not empty. Emptying bucket contents first...\n")

			// Empty the bucket with force option
			if err := s.EmptyBucketWithOptions(ctx, name, forceDelete); err != nil {
				return fmt.Errorf("failed to empty bucket before deletion: %w", err)
			}

			fmt.Printf("Bucket contents cleared. Attempting to delete bucket...\n")

			// Try to delete again after emptying
			resp2, err := client.Request("DELETE", endpoint, nil, nil)
			if err != nil {
				return fmt.Errorf("failed to delete bucket after emptying: %w", err)
			}
			defer resp2.Body.Close()

			if resp2.StatusCode != 204 {
				responseBody2, _ := ReadResponse(resp2)
				cleanError := parseS3Error(responseBody2)
				return fmt.Errorf("bucket deletion failed after emptying: %s", cleanError)
			}

			return nil
		}
	}

	// Handle other error cases with clean error parsing
	responseBody, _ := ReadResponse(resp)
	cleanError := parseS3Error(responseBody)
	return fmt.Errorf("bucket deletion failed: %s", cleanError)
}

// ListBuckets lists all buckets
func (s *StorageService) ListBuckets(ctx context.Context) ([]*provider.Bucket, error) {
	client, err := s.provider.CreateClient("s3")
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	resp, err := client.Request("GET", "/", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return nil, fmt.Errorf("ListBuckets failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var listResult ListAllMyBucketsResult
	if err := xml.Unmarshal(body, &listResult); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var buckets []*provider.Bucket
	for _, s3bucket := range listResult.Buckets.Bucket {
		createdAt := time.Now()
		if s3bucket.CreationDate != "" {
			if t, err := time.Parse(time.RFC3339, s3bucket.CreationDate); err == nil {
				createdAt = t
			}
		}

		buckets = append(buckets, &provider.Bucket{
			Name:      s3bucket.Name,
			Region:    s.provider.region,
			CreatedAt: createdAt,
		})
	}

	return buckets, nil
}

// DiscoverBuckets discovers existing buckets
func (s *StorageService) DiscoverBuckets(ctx context.Context) ([]*provider.Bucket, error) {
	return s.ListBuckets(ctx)
}

// AdoptBucket adopts an existing bucket into management
func (s *StorageService) AdoptBucket(ctx context.Context, name string) (*provider.Bucket, error) {
	return s.GetBucket(ctx, name)
}

// Helper methods for bucket configuration

func (s *StorageService) setBucketVersioning(client *AWSClient, bucketName string, enabled bool) error {
	status := "Suspended"
	if enabled {
		status = "Enabled"
	}

	versioningXML := fmt.Sprintf(`<VersioningConfiguration><Status>%s</Status></VersioningConfiguration>`, status)
	endpoint := fmt.Sprintf("/%s", bucketName)
	params := map[string]string{"versioning": ""}

	resp, err := client.Request("PUT", endpoint, params, []byte(versioningXML))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("failed to set versioning with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

func (s *StorageService) getBucketVersioning(client *AWSClient, bucketName string) (bool, error) {
	endpoint := fmt.Sprintf("/%s", bucketName)
	params := map[string]string{"versioning": ""}
	resp, err := client.Request("GET", endpoint, params, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("failed to get versioning status: %d", resp.StatusCode)
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return false, err
	}

	return strings.Contains(string(body), "<Status>Enabled</Status>"), nil
}

func (s *StorageService) setBucketEncryption(client *AWSClient, bucketName string) error {
	encryptionXML := `<ServerSideEncryptionConfiguration>
		<Rule>
			<ApplyServerSideEncryptionByDefault>
				<SSEAlgorithm>AES256</SSEAlgorithm>
			</ApplyServerSideEncryptionByDefault>
		</Rule>
	</ServerSideEncryptionConfiguration>`

	endpoint := fmt.Sprintf("/%s", bucketName)
	params := map[string]string{"encryption": ""}

	resp, err := client.Request("PUT", endpoint, params, []byte(encryptionXML))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("failed to set encryption with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

func (s *StorageService) getBucketEncryption(client *AWSClient, bucketName string) (bool, error) {
	endpoint := fmt.Sprintf("/%s", bucketName)
	params := map[string]string{"encryption": ""}
	resp, err := client.Request("GET", endpoint, params, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// 404 means no encryption is configured
	if resp.StatusCode == 404 {
		return false, nil
	}

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("failed to get encryption status: %d", resp.StatusCode)
	}

	return true, nil
}

func (s *StorageService) setBucketTags(client *AWSClient, bucketName string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	var tagSetXML strings.Builder
	tagSetXML.WriteString("<Tagging><TagSet>")

	for key, value := range tags {
		tagSetXML.WriteString(fmt.Sprintf("<Tag><Key>%s</Key><Value>%s</Value></Tag>", key, value))
	}

	tagSetXML.WriteString("</TagSet></Tagging>")

	endpoint := fmt.Sprintf("/%s", bucketName)
	params := map[string]string{"tagging": ""}

	// S3 tagging requires Content-MD5 header, use RequestWithHeaders
	resp, err := client.RequestWithMD5("PUT", endpoint, params, []byte(tagSetXML.String()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("failed to set tags with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

func (s *StorageService) getBucketTags(client *AWSClient, bucketName string) (map[string]string, error) {
	endpoint := fmt.Sprintf("/%s", bucketName)
	params := map[string]string{"tagging": ""}
	resp, err := client.Request("GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 404 means no tags are set
	if resp.StatusCode == 404 {
		return make(map[string]string), nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get tags: %d", resp.StatusCode)
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, err
	}

	// Simple parsing for tags (could be improved with proper XML parsing)
	tags := make(map[string]string)
	bodyStr := string(body)

	// This is a simplified approach - in production you'd want proper XML parsing
	if strings.Contains(bodyStr, "<TagSet>") {
		// Extract tags using simple string parsing
		// For now, return empty map - full implementation would parse XML properly
	}

	return tags, nil
}

// S3 object listing structures
type ListObjectsV2Result struct {
	XMLName               xml.Name       `xml:"ListBucketResult"`
	Name                  string         `xml:"Name"`
	IsTruncated           bool           `xml:"IsTruncated"`
	KeyCount              int            `xml:"KeyCount"`
	MaxKeys               int            `xml:"MaxKeys"`
	Prefix                string         `xml:"Prefix"`
	Contents              []S3Object     `xml:"Contents"`
	CommonPrefixes        []CommonPrefix `xml:"CommonPrefixes"`
	NextContinuationToken string         `xml:"NextContinuationToken"`
}

type S3Object struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

// DeleteObjectsRequest represents a batch delete request
type DeleteObjectsRequest struct {
	XMLName xml.Name           `xml:"Delete"`
	Objects []DeleteObjectItem `xml:"Object"`
	Quiet   bool               `xml:"Quiet"`
}

type DeleteObjectItem struct {
	Key       string `xml:"Key"`
	VersionId string `xml:"VersionId,omitempty"`
}

// DeleteObjectsResult represents the response from a batch delete operation
type DeleteObjectsResult struct {
	XMLName xml.Name        `xml:"DeleteResult"`
	Deleted []DeletedObject `xml:"Deleted"`
	Errors  []DeleteError   `xml:"Error"`
}

type DeletedObject struct {
	Key       string `xml:"Key"`
	VersionId string `xml:"VersionId,omitempty"`
}

type DeleteError struct {
	Key     string `xml:"Key"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

// S3Error represents an S3 API error response
type S3Error struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Bucket    string   `xml:"BucketName"`
	RequestId string   `xml:"RequestId"`
	HostId    string   `xml:"HostId"`
}

// parseS3Error extracts a clean error message from S3 XML error response
func parseS3Error(responseBody []byte) string {
	var s3Err S3Error
	if err := xml.Unmarshal(responseBody, &s3Err); err != nil {
		// If we can't parse the XML, return the raw response
		return string(responseBody)
	}

	// Return a clean, user-friendly error message
	if s3Err.Code != "" && s3Err.Message != "" {
		return fmt.Sprintf("%s: %s", s3Err.Code, s3Err.Message)
	}

	// Fallback to just the message if available
	if s3Err.Message != "" {
		return s3Err.Message
	}

	// Last resort fallback
	return string(responseBody)
}

// EmptyBucket removes all objects and versions from a bucket
func (s *StorageService) EmptyBucket(ctx context.Context, bucketName string) error {
	return s.EmptyBucketWithOptions(ctx, bucketName, false)
}

// EmptyBucketWithOptions removes all objects and versions from a bucket with advanced options
func (s *StorageService) EmptyBucketWithOptions(ctx context.Context, bucketName string, forceDelete bool) error {
	// Use bucket-specific client to handle cross-region bucket access
	client, err := s.createClientForBucket(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	// First, delete all current object versions
	if err := s.deleteAllObjects(client, bucketName); err != nil {
		return fmt.Errorf("failed to delete objects: %w", err)
	}

	// If force delete is enabled, also delete all non-current versions and delete markers
	if forceDelete {
		fmt.Printf("Force deletion enabled. Removing all object versions and delete markers...\n")
		if err := s.deleteAllVersionsAndMarkers(client, bucketName); err != nil {
			return fmt.Errorf("failed to delete object versions: %w", err)
		}
	} else {
		// Otherwise, try basic version deletion
		if err := s.deleteAllVersions(client, bucketName); err != nil {
			return fmt.Errorf("failed to delete object versions: %w", err)
		}
	}

	return nil
}

// deleteAllObjects deletes all current objects in the bucket
func (s *StorageService) deleteAllObjects(client *AWSClient, bucketName string) error {
	continuationToken := ""

	for {
		// List objects
		endpoint := fmt.Sprintf("/%s", bucketName)
		params := map[string]string{
			"list-type": "2",
			"max-keys":  "1000",
		}

		if continuationToken != "" {
			params["continuation-token"] = continuationToken
		}

		resp, err := client.Request("GET", endpoint, params, nil)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			responseBody, _ := ReadResponse(resp)
			return fmt.Errorf("ListObjectsV2 failed with status %d: %s", resp.StatusCode, string(responseBody))
		}

		body, err := ReadResponse(resp)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		var listResult ListObjectsV2Result
		if err := xml.Unmarshal(body, &listResult); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// If no objects found, we're done
		if listResult.KeyCount == 0 {
			break
		}

		// Delete objects in batches of up to 1000
		if err := s.deleteObjectBatch(client, bucketName, listResult.Contents); err != nil {
			return err
		}

		// Check if we need to continue
		if !listResult.IsTruncated {
			break
		}
		continuationToken = listResult.NextContinuationToken
	}

	return nil
}

// ListVersionsResult represents the response from ListObjectVersions
type ListVersionsResult struct {
	XMLName             xml.Name        `xml:"ListVersionsResult"`
	Name                string          `xml:"Name"`
	IsTruncated         bool            `xml:"IsTruncated"`
	KeyMarker           string          `xml:"KeyMarker"`
	VersionIdMarker     string          `xml:"VersionIdMarker"`
	NextKeyMarker       string          `xml:"NextKeyMarker"`
	NextVersionIdMarker string          `xml:"NextVersionIdMarker"`
	MaxKeys             int             `xml:"MaxKeys"`
	Versions            []ObjectVersion `xml:"Version"`
	DeleteMarkers       []DeleteMarker  `xml:"DeleteMarker"`
}

type ObjectVersion struct {
	Key          string `xml:"Key"`
	VersionId    string `xml:"VersionId"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
	IsLatest     bool   `xml:"IsLatest"`
}

type DeleteMarker struct {
	Key          string `xml:"Key"`
	VersionId    string `xml:"VersionId"`
	LastModified string `xml:"LastModified"`
	IsLatest     bool   `xml:"IsLatest"`
}

// deleteAllVersions deletes all object versions in the bucket (basic version - may not handle all cases)
func (s *StorageService) deleteAllVersions(client *AWSClient, bucketName string) error {
	// This is a simplified version - just attempts basic cleanup
	// The full implementation is in deleteAllVersionsAndMarkers
	return nil
}

// deleteAllVersionsAndMarkers deletes all object versions and delete markers in the bucket
func (s *StorageService) deleteAllVersionsAndMarkers(client *AWSClient, bucketName string) error {
	keyMarker := ""
	versionIdMarker := ""

	for {
		// List object versions
		endpoint := fmt.Sprintf("/%s", bucketName)
		params := map[string]string{
			"versions": "",
			"max-keys": "1000",
		}

		if keyMarker != "" {
			params["key-marker"] = keyMarker
		}
		if versionIdMarker != "" {
			params["version-id-marker"] = versionIdMarker
		}

		resp, err := client.Request("GET", endpoint, params, nil)
		if err != nil {
			return fmt.Errorf("failed to list object versions: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			responseBody, _ := ReadResponse(resp)
			cleanError := parseS3Error(responseBody)
			return fmt.Errorf("ListObjectVersions failed: %s", cleanError)
		}

		body, err := ReadResponse(resp)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		var listResult ListVersionsResult
		if err := xml.Unmarshal(body, &listResult); err != nil {
			return fmt.Errorf("failed to parse versions response: %w", err)
		}

		// Collect all versions and delete markers to delete
		var itemsToDelete []DeleteObjectItem

		// Add all versions
		for _, version := range listResult.Versions {
			itemsToDelete = append(itemsToDelete, DeleteObjectItem{
				Key:       version.Key,
				VersionId: version.VersionId,
			})
		}

		// Add all delete markers
		for _, marker := range listResult.DeleteMarkers {
			itemsToDelete = append(itemsToDelete, DeleteObjectItem{
				Key:       marker.Key,
				VersionId: marker.VersionId,
			})
		}

		// If no items to delete, we're done
		if len(itemsToDelete) == 0 {
			break
		}

		// Delete this batch of versions and markers
		if err := s.deleteVersionBatch(client, bucketName, itemsToDelete); err != nil {
			return err
		}

		fmt.Printf("Deleted %d object versions/delete markers\n", len(itemsToDelete))

		// Check if we need to continue
		if !listResult.IsTruncated {
			break
		}

		keyMarker = listResult.NextKeyMarker
		versionIdMarker = listResult.NextVersionIdMarker
	}

	return nil
}

// deleteVersionBatch deletes a batch of object versions and delete markers
func (s *StorageService) deleteVersionBatch(client *AWSClient, bucketName string, items []DeleteObjectItem) error {
	if len(items) == 0 {
		return nil
	}

	// Prepare delete request
	deleteRequest := DeleteObjectsRequest{
		Objects: items,
		Quiet:   true, // Don't return successful deletions in response
	}

	// Marshal to XML
	xmlData, err := xml.Marshal(deleteRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	// Send batch delete request
	endpoint := fmt.Sprintf("/%s", bucketName)
	params := map[string]string{"delete": ""}

	resp, err := client.RequestWithMD5("POST", endpoint, params, xmlData)
	if err != nil {
		return fmt.Errorf("failed to delete object versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		cleanError := parseS3Error(responseBody)
		return fmt.Errorf("DeleteObjects failed: %s", cleanError)
	}

	return nil
}

// deleteObjectBatch deletes a batch of objects
func (s *StorageService) deleteObjectBatch(client *AWSClient, bucketName string, objects []S3Object) error {
	if len(objects) == 0 {
		return nil
	}

	// Prepare delete request
	deleteRequest := DeleteObjectsRequest{
		Objects: make([]DeleteObjectItem, len(objects)),
		Quiet:   true, // Don't return successful deletions in response
	}

	for i, obj := range objects {
		deleteRequest.Objects[i] = DeleteObjectItem{
			Key: obj.Key,
		}
	}

	// Marshal to XML
	xmlData, err := xml.Marshal(deleteRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	// Send batch delete request
	endpoint := fmt.Sprintf("/%s", bucketName)
	params := map[string]string{"delete": ""}

	resp, err := client.RequestWithMD5("POST", endpoint, params, xmlData)
	if err != nil {
		return fmt.Errorf("failed to delete objects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("DeleteObjects failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

// S3 Object Management Operations

// ListObjects lists objects in a bucket with optional prefix and max keys
func (s *StorageService) ListObjects(ctx context.Context, bucketName, prefix string, maxKeys int) ([]*provider.S3ObjectInfo, error) {
	// Use bucket-specific client to handle cross-region bucket access
	client, err := s.createClientForBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s", bucketName)
	params := map[string]string{
		"list-type": "2",
	}

	if prefix != "" {
		params["prefix"] = prefix
	}

	if maxKeys > 0 {
		params["max-keys"] = fmt.Sprintf("%d", maxKeys)
	} else {
		params["max-keys"] = "1000"
	}

	resp, err := client.Request("GET", endpoint, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		cleanError := parseS3Error(responseBody)
		return nil, fmt.Errorf("ListObjects failed: %s", cleanError)
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var listResult ListObjectsV2Result
	if err := xml.Unmarshal(body, &listResult); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var objects []*provider.S3ObjectInfo

	// Add common prefixes (folders) first
	for _, prefix := range listResult.CommonPrefixes {
		objects = append(objects, &provider.S3ObjectInfo{
			Key:      prefix.Prefix,
			IsPrefix: true,
		})
	}

	// Add objects
	for _, obj := range listResult.Contents {
		lastModified, _ := time.Parse(time.RFC3339, obj.LastModified)
		objects = append(objects, &provider.S3ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: lastModified,
			ETag:         strings.Trim(obj.ETag, "\""),
			StorageClass: obj.StorageClass,
			IsPrefix:     false,
		})
	}

	return objects, nil
}

// ListObjectsRecursive lists all objects recursively in a bucket with optional prefix
func (s *StorageService) ListObjectsRecursive(ctx context.Context, bucketName, prefix string) ([]*provider.S3ObjectInfo, error) {
	// Use bucket-specific client to handle cross-region bucket access
	client, err := s.createClientForBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	var allObjects []*provider.S3ObjectInfo
	continuationToken := ""

	for {
		endpoint := fmt.Sprintf("/%s", bucketName)
		params := map[string]string{
			"list-type": "2",
			"max-keys":  "1000",
		}

		if prefix != "" {
			params["prefix"] = prefix
		}

		if continuationToken != "" {
			params["continuation-token"] = continuationToken
		}

		resp, err := client.Request("GET", endpoint, params, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		if resp.StatusCode != 200 {
			responseBody, _ := ReadResponse(resp)
			resp.Body.Close()
			cleanError := parseS3Error(responseBody)
			return nil, fmt.Errorf("ListObjects failed: %s", cleanError)
		}

		body, err := ReadResponse(resp)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		var listResult ListObjectsV2Result
		if err := xml.Unmarshal(body, &listResult); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		for _, obj := range listResult.Contents {
			lastModified, _ := time.Parse(time.RFC3339, obj.LastModified)
			allObjects = append(allObjects, &provider.S3ObjectInfo{
				Key:          obj.Key,
				Size:         obj.Size,
				LastModified: lastModified,
				ETag:         strings.Trim(obj.ETag, "\""),
				StorageClass: obj.StorageClass,
				IsPrefix:     false,
			})
		}

		if !listResult.IsTruncated {
			break
		}

		continuationToken = listResult.NextContinuationToken
	}

	return allObjects, nil
}

// GetObject retrieves an object's content from S3
func (s *StorageService) GetObject(ctx context.Context, bucketName, key string) ([]byte, error) {
	// Use bucket-specific client to handle cross-region bucket access
	client, err := s.createClientForBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s/%s", bucketName, encodeS3Key(key))
	resp, err := client.Request("GET", endpoint, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		cleanError := parseS3Error(responseBody)
		return nil, fmt.Errorf("GetObject failed: %s", cleanError)
	}

	return ReadResponse(resp)
}

// GetObjectMetadata retrieves metadata about an S3 object
func (s *StorageService) GetObjectMetadata(ctx context.Context, bucketName, key string) (*provider.S3ObjectMetadata, error) {
	// Use bucket-specific client to handle cross-region bucket access
	client, err := s.createClientForBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s/%s", bucketName, encodeS3Key(key))

	resp, err := client.Request("HEAD", endpoint, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to head object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		cleanError := parseS3Error(responseBody)
		return nil, fmt.Errorf("HeadObject failed: %s", cleanError)
	}

	metadata := &provider.S3ObjectMetadata{
		Key:                  key,
		ContentType:          resp.Header.Get("Content-Type"),
		ETag:                 strings.Trim(resp.Header.Get("ETag"), "\""),
		StorageClass:         resp.Header.Get("X-Amz-Storage-Class"),
		ServerSideEncryption: resp.Header.Get("X-Amz-Server-Side-Encryption"),
		VersionID:            resp.Header.Get("X-Amz-Version-Id"),
		CacheControl:         resp.Header.Get("Cache-Control"),
		ContentDisposition:   resp.Header.Get("Content-Disposition"),
		ContentEncoding:      resp.Header.Get("Content-Encoding"),
		ContentLanguage:      resp.Header.Get("Content-Language"),
	}

	if sizeStr := resp.Header.Get("Content-Length"); sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
			metadata.Size = size
		}
	}

	if modStr := resp.Header.Get("Last-Modified"); modStr != "" {
		if lastMod, err := time.Parse(time.RFC1123, modStr); err == nil {
			metadata.LastModified = lastMod
		}
	}

	metadata.Metadata = make(map[string]string)
	for key, values := range resp.Header {
		if strings.HasPrefix(key, "X-Amz-Meta-") {
			metaKey := strings.TrimPrefix(key, "X-Amz-Meta-")
			metadata.Metadata[metaKey] = values[0]
		}
	}

	return metadata, nil
}

// PutObject uploads data to S3
func (s *StorageService) PutObject(ctx context.Context, bucketName, key string, data []byte, contentType string) error {
	// Use bucket-specific client to handle cross-region bucket access
	client, err := s.createClientForBucket(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s/%s", bucketName, encodeS3Key(key))

	req, err := http.NewRequest("PUT", fmt.Sprintf("https://s3.%s.amazonaws.com%s", client.Region, endpoint), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Content-Length", strconv.Itoa(len(data)))

	payloadHash := fmt.Sprintf("%x", sha256.Sum256(data))
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	if err := client.signRequest(req, data); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		cleanError := parseS3Error(responseBody)
		return fmt.Errorf("PutObject failed: %s", cleanError)
	}

	return nil
}

// DeleteObject deletes a single object from S3
func (s *StorageService) DeleteObject(ctx context.Context, bucketName, key string) error {
	// Use bucket-specific client to handle cross-region bucket access
	client, err := s.createClientForBucket(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s/%s", bucketName, encodeS3Key(key))
	resp, err := client.Request("DELETE", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		cleanError := parseS3Error(responseBody)
		return fmt.Errorf("DeleteObject failed: %s", cleanError)
	}

	return nil
}

// CopyObject copies an object within S3
func (s *StorageService) CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	// Use destination bucket's region for the client
	client, err := s.createClientForBucket(ctx, dstBucket)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s/%s", dstBucket, encodeS3Key(dstKey))
	copySource := fmt.Sprintf("/%s/%s", srcBucket, encodeS3Key(srcKey))

	req, err := http.NewRequest("PUT", fmt.Sprintf("https://s3.%s.amazonaws.com%s", client.Region, endpoint), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Amz-Copy-Source", copySource)
	req.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")

	if err := client.signRequest(req, nil); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		cleanError := parseS3Error(responseBody)
		return fmt.Errorf("CopyObject failed: %s", cleanError)
	}

	return nil
}

// UploadFile uploads a file from local filesystem to S3 with progress tracking
func (s *StorageService) UploadFile(ctx context.Context, bucketName, key, localPath string, progress chan<- *provider.TransferProgress) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	fileSize := fileInfo.Size()
	startTime := time.Now()

	// Read file into memory (for small files)
	// TODO: Implement multipart upload for large files
	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Detect content type
	contentType := http.DetectContentType(data)

	if progress != nil {
		progress <- &provider.TransferProgress{
			TotalBytes:       fileSize,
			TransferredBytes: 0,
			PercentComplete:  0,
			StartTime:        startTime,
			CurrentFile:      localPath,
			Status:           "uploading",
		}
	}

	err = s.PutObject(ctx, bucketName, key, data, contentType)
	if err != nil {
		if progress != nil {
			progress <- &provider.TransferProgress{
				TotalBytes:       fileSize,
				TransferredBytes: 0,
				Status:           "failed",
				Error:            err,
			}
		}
		return err
	}

	if progress != nil {
		elapsed := time.Since(startTime)
		bytesPerSecond := float64(fileSize) / elapsed.Seconds()

		progress <- &provider.TransferProgress{
			TotalBytes:       fileSize,
			TransferredBytes: fileSize,
			PercentComplete:  100.0,
			BytesPerSecond:   bytesPerSecond,
			StartTime:        startTime,
			CurrentFile:      localPath,
			Status:           "complete",
		}
	}

	return nil
}

// DownloadFile downloads a file from S3 to local filesystem with progress tracking
func (s *StorageService) DownloadFile(ctx context.Context, bucketName, key, localPath string, progress chan<- *provider.TransferProgress) error {
	startTime := time.Now()

	// Get object metadata first to know the size
	metadata, err := s.GetObjectMetadata(ctx, bucketName, key)
	if err != nil {
		return fmt.Errorf("failed to get object metadata: %w", err)
	}

	if progress != nil {
		progress <- &provider.TransferProgress{
			TotalBytes:       metadata.Size,
			TransferredBytes: 0,
			PercentComplete:  0,
			StartTime:        startTime,
			CurrentFile:      key,
			Status:           "downloading",
		}
	}

	// Download object
	data, err := s.GetObject(ctx, bucketName, key)
	if err != nil {
		if progress != nil {
			progress <- &provider.TransferProgress{
				TotalBytes: metadata.Size,
				Status:     "failed",
				Error:      err,
			}
		}
		return err
	}

	// Write to file
	if err := os.WriteFile(localPath, data, 0644); err != nil {
		if progress != nil {
			progress <- &provider.TransferProgress{
				TotalBytes: metadata.Size,
				Status:     "failed",
				Error:      err,
			}
		}
		return fmt.Errorf("failed to write file: %w", err)
	}

	if progress != nil {
		elapsed := time.Since(startTime)
		bytesPerSecond := float64(len(data)) / elapsed.Seconds()

		progress <- &provider.TransferProgress{
			TotalBytes:       metadata.Size,
			TransferredBytes: metadata.Size,
			PercentComplete:  100.0,
			BytesPerSecond:   bytesPerSecond,
			StartTime:        startTime,
			CurrentFile:      key,
			Status:           "complete",
		}
	}

	return nil
}

// SyncDirectory synchronizes a local directory with an S3 prefix
func (s *StorageService) SyncDirectory(ctx context.Context, bucketName, prefix, localPath, direction string) error {
	switch direction {
	case "upload", "up", "push":
		return s.syncUpload(ctx, bucketName, prefix, localPath)
	case "download", "down", "pull":
		return s.syncDownload(ctx, bucketName, prefix, localPath)
	default:
		return fmt.Errorf("invalid direction: %s (use 'upload' or 'download')", direction)
	}
}

// syncUpload syncs local directory to S3
func (s *StorageService) syncUpload(ctx context.Context, bucketName, prefix, localPath string) error {
	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			return err
		}

		s3Key := filepath.Join(prefix, relPath)
		s3Key = strings.ReplaceAll(s3Key, "\\", "/") // Windows path fix

		fmt.Printf("Uploading: %s → s3://%s/%s\n", path, bucketName, s3Key)

		if err := s.UploadFile(ctx, bucketName, s3Key, path, nil); err != nil {
			fmt.Printf("  Error: %v\n", err)
			return err
		}

		fmt.Printf("  Complete\n")
		return nil
	})
}

// syncDownload syncs S3 prefix to local directory
func (s *StorageService) syncDownload(ctx context.Context, bucketName, prefix, localPath string) error {
	objects, err := s.ListObjectsRecursive(ctx, bucketName, prefix)
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	for _, obj := range objects {
		if obj.IsPrefix {
			continue
		}

		relPath := strings.TrimPrefix(obj.Key, prefix)
		relPath = strings.TrimPrefix(relPath, "/")
		localFile := filepath.Join(localPath, relPath)

		localDir := filepath.Dir(localFile)
		if err := os.MkdirAll(localDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		fmt.Printf("Downloading: s3://%s/%s → %s\n", bucketName, obj.Key, localFile)

		if err := s.DownloadFile(ctx, bucketName, obj.Key, localFile, nil); err != nil {
			fmt.Printf("  Error: %v\n", err)
			return err
		}

		fmt.Printf("  Complete\n")
	}

	return nil
}

// GeneratePresignedURL generates a presigned URL for temporary access
func (s *StorageService) GeneratePresignedURL(ctx context.Context, bucketName, key string, expiresIn int) (string, error) {
	client, err := s.provider.CreateClient("s3")
	if err != nil {
		return "", fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s/%s", bucketName, key)
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	url := fmt.Sprintf("https://s3.%s.amazonaws.com%s", s.provider.region, endpoint)

	params := map[string]string{
		"X-Amz-Algorithm":     "AWS4-HMAC-SHA256",
		"X-Amz-Credential":    fmt.Sprintf("%s/%s/%s/s3/aws4_request", client.AccessKey, expiresAt.Format("20060102"), s.provider.region),
		"X-Amz-Date":          expiresAt.Format("20060102T150405Z"),
		"X-Amz-Expires":       strconv.Itoa(expiresIn),
		"X-Amz-SignedHeaders": "host",
	}

	var paramPairs []string
	for k, v := range params {
		paramPairs = append(paramPairs, fmt.Sprintf("%s=%s", k, v))
	}

	return fmt.Sprintf("%s?%s", url, strings.Join(paramPairs, "&")), nil
}

// GetBucketRegion retrieves the region where a bucket is located
func (s *StorageService) GetBucketRegion(ctx context.Context, bucketName string) (string, error) {
	client, err := s.provider.CreateClient("s3")
	if err != nil {
		return "", fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s", bucketName)
	params := map[string]string{"location": ""}

	resp, err := client.Request("GET", endpoint, params, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get bucket location: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return "", fmt.Errorf("GetBucketLocation failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var locationResult BucketLocationConstraint
	if err := xml.Unmarshal(body, &locationResult); err != nil {
		// Empty response means us-east-1
		if len(body) == 0 || string(body) == "" {
			return "us-east-1", nil
		}
		return "", fmt.Errorf("failed to parse location response: %w", err)
	}

	// Empty LocationConstraint means us-east-1
	if locationResult.LocationConstraint == "" {
		return "us-east-1", nil
	}

	return locationResult.LocationConstraint, nil
}

// ListBucketsInRegion lists all buckets and filters by region
func (s *StorageService) ListBucketsInRegion(ctx context.Context, region string) ([]*provider.Bucket, error) {
	// List all buckets first
	allBuckets, err := s.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by region
	var regionBuckets []*provider.Bucket
	for _, bucket := range allBuckets {
		bucketRegion, err := s.GetBucketRegion(ctx, bucket.Name)
		if err != nil {
			// Skip buckets we can't determine region for
			continue
		}
		if bucketRegion == region {
			bucket.Region = bucketRegion
			regionBuckets = append(regionBuckets, bucket)
		}
	}

	return regionBuckets, nil
}

// CopyObjectCrossRegion copies an object from source bucket to destination bucket in a different region
func (s *StorageService) CopyObjectCrossRegion(ctx context.Context, srcBucket, srcKey, dstRegion, dstBucket, dstKey string) error {
	// Create a client for the destination region
	dstClient, err := NewAWSClient(dstRegion, "s3")
	if err != nil {
		return fmt.Errorf("failed to create destination region client: %w", err)
	}

	// Get source object metadata to check size
	srcMetadata, err := s.GetObjectMetadata(ctx, srcBucket, srcKey)
	if err != nil {
		return fmt.Errorf("failed to get source object metadata: %w", err)
	}

	// For objects larger than 5GB, use multipart copy
	const multipartThreshold = 5 * 1024 * 1024 * 1024 // 5GB
	if srcMetadata.Size > multipartThreshold {
		return s.copyLargeObjectCrossRegion(ctx, dstClient, srcBucket, srcKey, dstRegion, dstBucket, dstKey, srcMetadata.Size)
	}

	// For smaller objects, use simple copy
	return s.copySmallObjectCrossRegion(ctx, dstClient, srcBucket, srcKey, dstRegion, dstBucket, dstKey)
}

// copySmallObjectCrossRegion copies objects smaller than 5GB using simple PUT with x-amz-copy-source
func (s *StorageService) copySmallObjectCrossRegion(ctx context.Context, dstClient *AWSClient, srcBucket, srcKey, dstRegion, dstBucket, dstKey string) error {
	endpoint := fmt.Sprintf("/%s/%s", dstBucket, dstKey)

	// URL encode the copy source path
	copySource := fmt.Sprintf("/%s/%s", srcBucket, srcKey)

	req, err := http.NewRequest("PUT", fmt.Sprintf("https://s3.%s.amazonaws.com%s", dstRegion, endpoint), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set the copy source header - this tells S3 to copy from another location
	req.Header.Set("x-amz-copy-source", copySource)
	req.Header.Set("x-amz-content-sha256", "UNSIGNED-PAYLOAD")

	if err := dstClient.signRequest(req, nil); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := dstClient.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		cleanError := parseS3Error(responseBody)
		return fmt.Errorf("CopyObject cross-region failed: %s", cleanError)
	}

	return nil
}

// copyLargeObjectCrossRegion copies objects larger than 5GB using multipart copy
func (s *StorageService) copyLargeObjectCrossRegion(ctx context.Context, dstClient *AWSClient, srcBucket, srcKey, dstRegion, dstBucket, dstKey string, objectSize int64) error {
	// Part size: 100MB for large objects
	const partSize int64 = 100 * 1024 * 1024

	// Calculate number of parts
	numParts := (objectSize + partSize - 1) / partSize

	// Initiate multipart upload
	uploadID, err := s.initiateMultipartUpload(dstClient, dstRegion, dstBucket, dstKey)
	if err != nil {
		return fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	// Copy parts
	var completedParts []completedPart
	copySource := fmt.Sprintf("/%s/%s", srcBucket, srcKey)

	for partNum := int64(1); partNum <= numParts; partNum++ {
		startByte := (partNum - 1) * partSize
		endByte := startByte + partSize - 1
		if endByte >= objectSize {
			endByte = objectSize - 1
		}

		etag, err := s.uploadPartCopy(dstClient, dstRegion, dstBucket, dstKey, uploadID, int(partNum), copySource, startByte, endByte)
		if err != nil {
			// Abort multipart upload on failure
			s.abortMultipartUpload(dstClient, dstRegion, dstBucket, dstKey, uploadID)
			return fmt.Errorf("failed to copy part %d: %w", partNum, err)
		}

		completedParts = append(completedParts, completedPart{
			PartNumber: int(partNum),
			ETag:       etag,
		})
	}

	// Complete multipart upload
	if err := s.completeMultipartUpload(dstClient, dstRegion, dstBucket, dstKey, uploadID, completedParts); err != nil {
		s.abortMultipartUpload(dstClient, dstRegion, dstBucket, dstKey, uploadID)
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

// completedPart represents a completed part for multipart upload
type completedPart struct {
	PartNumber int
	ETag       string
}

// initiateMultipartUpload starts a multipart upload and returns the upload ID
func (s *StorageService) initiateMultipartUpload(client *AWSClient, region, bucket, key string) (string, error) {
	endpoint := fmt.Sprintf("/%s/%s", bucket, key)
	params := map[string]string{"uploads": ""}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://s3.%s.amazonaws.com%s?uploads", region, endpoint), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-amz-content-sha256", "UNSIGNED-PAYLOAD")

	if err := client.signRequest(req, nil); err != nil {
		return "", fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return "", fmt.Errorf("InitiateMultipartUpload failed: %s", parseS3Error(responseBody))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return "", err
	}

	var result struct {
		XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
		UploadId string   `xml:"UploadId"`
	}
	if err := xml.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Suppress unused variable warning for params
	_ = params

	return result.UploadId, nil
}

// uploadPartCopy copies a part from source to destination
func (s *StorageService) uploadPartCopy(client *AWSClient, region, bucket, key, uploadID string, partNumber int, copySource string, startByte, endByte int64) (string, error) {
	endpoint := fmt.Sprintf("/%s/%s", bucket, key)
	urlStr := fmt.Sprintf("https://s3.%s.amazonaws.com%s?partNumber=%d&uploadId=%s", region, endpoint, partNumber, uploadID)

	req, err := http.NewRequest("PUT", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-amz-copy-source", copySource)
	req.Header.Set("x-amz-copy-source-range", fmt.Sprintf("bytes=%d-%d", startByte, endByte))
	req.Header.Set("x-amz-content-sha256", "UNSIGNED-PAYLOAD")

	if err := client.signRequest(req, nil); err != nil {
		return "", fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return "", fmt.Errorf("UploadPartCopy failed: %s", parseS3Error(responseBody))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return "", err
	}

	var result struct {
		XMLName xml.Name `xml:"CopyPartResult"`
		ETag    string   `xml:"ETag"`
	}
	if err := xml.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.ETag, nil
}

// completeMultipartUpload completes a multipart upload
func (s *StorageService) completeMultipartUpload(client *AWSClient, region, bucket, key, uploadID string, parts []completedPart) error {
	endpoint := fmt.Sprintf("/%s/%s", bucket, key)
	urlStr := fmt.Sprintf("https://s3.%s.amazonaws.com%s?uploadId=%s", region, endpoint, uploadID)

	// Build completion XML
	var xmlParts strings.Builder
	xmlParts.WriteString("<CompleteMultipartUpload>")
	for _, part := range parts {
		xmlParts.WriteString(fmt.Sprintf("<Part><PartNumber>%d</PartNumber><ETag>%s</ETag></Part>", part.PartNumber, part.ETag))
	}
	xmlParts.WriteString("</CompleteMultipartUpload>")

	body := []byte(xmlParts.String())

	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	payloadHash := fmt.Sprintf("%x", sha256.Sum256(body))
	req.Header.Set("x-amz-content-sha256", payloadHash)

	if err := client.signRequest(req, body); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("CompleteMultipartUpload failed: %s", parseS3Error(responseBody))
	}

	return nil
}

// abortMultipartUpload aborts a multipart upload
func (s *StorageService) abortMultipartUpload(client *AWSClient, region, bucket, key, uploadID string) error {
	endpoint := fmt.Sprintf("/%s/%s", bucket, key)
	urlStr := fmt.Sprintf("https://s3.%s.amazonaws.com%s?uploadId=%s", region, endpoint, uploadID)

	req, err := http.NewRequest("DELETE", urlStr, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-amz-content-sha256", "UNSIGNED-PAYLOAD")

	if err := client.signRequest(req, nil); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// CopyBucketCrossRegion copies all objects from source bucket to destination bucket in a different region
func (s *StorageService) CopyBucketCrossRegion(ctx context.Context, srcBucket, dstRegion, dstBucket, prefix string, progress chan<- *provider.CrossRegionCopyProgress) error {
	startTime := time.Now()

	// Get source bucket region
	srcRegion, err := s.GetBucketRegion(ctx, srcBucket)
	if err != nil {
		srcRegion = s.provider.region
	}

	// Send initial progress
	if progress != nil {
		progress <- &provider.CrossRegionCopyProgress{
			SourceBucket: srcBucket,
			SourceRegion: srcRegion,
			DestBucket:   dstBucket,
			DestRegion:   dstRegion,
			Status:       "preparing",
			StartTime:    startTime,
		}
	}

	// List all objects in source bucket
	objects, err := s.ListObjectsRecursive(ctx, srcBucket, prefix)
	if err != nil {
		if progress != nil {
			progress <- &provider.CrossRegionCopyProgress{
				Status: "failed",
				Error:  fmt.Errorf("failed to list source objects: %w", err),
			}
		}
		return fmt.Errorf("failed to list source objects: %w", err)
	}

	// Calculate total size
	var totalBytes int64
	var totalObjects int64
	for _, obj := range objects {
		if !obj.IsPrefix {
			totalBytes += obj.Size
			totalObjects++
		}
	}

	if progress != nil {
		progress <- &provider.CrossRegionCopyProgress{
			SourceBucket:    srcBucket,
			SourceRegion:    srcRegion,
			DestBucket:      dstBucket,
			DestRegion:      dstRegion,
			TotalObjects:    totalObjects,
			TotalBytes:      totalBytes,
			Status:          "copying",
			StartTime:       startTime,
			PercentComplete: 0,
		}
	}

	// Ensure destination bucket exists
	dstClient, err := NewAWSClient(dstRegion, "s3")
	if err != nil {
		if progress != nil {
			progress <- &provider.CrossRegionCopyProgress{
				Status: "failed",
				Error:  fmt.Errorf("failed to create destination client: %w", err),
			}
		}
		return fmt.Errorf("failed to create destination client: %w", err)
	}

	// Try to create destination bucket (ignore error if it already exists)
	err = s.createBucketInRegion(dstClient, dstRegion, dstBucket)
	if err != nil && !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") && !strings.Contains(err.Error(), "BucketAlreadyExists") {
		if progress != nil {
			progress <- &provider.CrossRegionCopyProgress{
				Status: "failed",
				Error:  fmt.Errorf("failed to create destination bucket: %w", err),
			}
		}
		return fmt.Errorf("failed to create destination bucket: %w", err)
	}

	// Copy objects concurrently
	var copiedObjects, copiedBytes int64
	var failedObjects int64
	var failedKeys []string
	var mu sync.Mutex

	// Use worker pool for concurrent copies
	const maxWorkers = 10
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, obj := range objects {
		if obj.IsPrefix {
			continue
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(obj *provider.S3ObjectInfo) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Determine destination key
			dstKey := obj.Key
			if prefix != "" {
				dstKey = strings.TrimPrefix(obj.Key, prefix)
			}

			// Send progress update
			if progress != nil {
				mu.Lock()
				progress <- &provider.CrossRegionCopyProgress{
					SourceBucket:    srcBucket,
					SourceRegion:    srcRegion,
					DestBucket:      dstBucket,
					DestRegion:      dstRegion,
					TotalObjects:    totalObjects,
					CopiedObjects:   copiedObjects,
					FailedObjects:   failedObjects,
					TotalBytes:      totalBytes,
					CopiedBytes:     copiedBytes,
					CurrentObject:   obj.Key,
					Status:          "copying",
					StartTime:       startTime,
					PercentComplete: float64(copiedObjects) / float64(totalObjects) * 100,
				}
				mu.Unlock()
			}

			// Copy the object
			err := s.CopyObjectCrossRegion(ctx, srcBucket, obj.Key, dstRegion, dstBucket, dstKey)

			mu.Lock()
			if err != nil {
				failedObjects++
				failedKeys = append(failedKeys, obj.Key)
			} else {
				copiedObjects++
				copiedBytes += obj.Size
			}
			mu.Unlock()
		}(obj)
	}

	wg.Wait()

	// Send final progress
	status := "complete"
	var finalErr error
	if failedObjects > 0 {
		if copiedObjects == 0 {
			status = "failed"
			finalErr = fmt.Errorf("all %d objects failed to copy", failedObjects)
		} else {
			status = "complete"
			finalErr = fmt.Errorf("%d objects failed to copy", failedObjects)
		}
	}

	elapsed := time.Since(startTime)
	bytesPerSecond := float64(copiedBytes) / elapsed.Seconds()

	if progress != nil {
		progress <- &provider.CrossRegionCopyProgress{
			SourceBucket:    srcBucket,
			SourceRegion:    srcRegion,
			DestBucket:      dstBucket,
			DestRegion:      dstRegion,
			TotalObjects:    totalObjects,
			CopiedObjects:   copiedObjects,
			FailedObjects:   failedObjects,
			TotalBytes:      totalBytes,
			CopiedBytes:     copiedBytes,
			Status:          status,
			StartTime:       startTime,
			PercentComplete: 100,
			BytesPerSecond:  bytesPerSecond,
			Error:           finalErr,
			FailedKeys:      failedKeys,
		}
	}

	return finalErr
}

// createBucketInRegion creates a bucket in a specific region
func (s *StorageService) createBucketInRegion(client *AWSClient, region, bucketName string) error {
	endpoint := fmt.Sprintf("/%s", bucketName)

	var body []byte
	if region != "us-east-1" {
		locationXML := fmt.Sprintf(`<CreateBucketConfiguration><LocationConstraint>%s</LocationConstraint></CreateBucketConfiguration>`, region)
		body = []byte(locationXML)
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("https://s3.%s.amazonaws.com%s", region, endpoint), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/xml")
		payloadHash := fmt.Sprintf("%x", sha256.Sum256(body))
		req.Header.Set("x-amz-content-sha256", payloadHash)
	} else {
		req.Header.Set("x-amz-content-sha256", "UNSIGNED-PAYLOAD")
	}

	if err := client.signRequest(req, body); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("CreateBucket failed: %s", parseS3Error(responseBody))
	}

	return nil
}
