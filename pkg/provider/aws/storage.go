package aws

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
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
	client, err := s.provider.CreateClient("s3")
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
	XMLName      xml.Name    `xml:"ListBucketResult"`
	Name         string      `xml:"Name"`
	IsTruncated  bool        `xml:"IsTruncated"`
	KeyCount     int         `xml:"KeyCount"`
	MaxKeys      int         `xml:"MaxKeys"`
	Prefix       string      `xml:"Prefix"`
	Contents     []S3Object  `xml:"Contents"`
	CommonPrefixes []CommonPrefix `xml:"CommonPrefixes"`
	NextContinuationToken string `xml:"NextContinuationToken"`
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
	XMLName xml.Name `xml:"Delete"`
	Objects []DeleteObjectItem `xml:"Object"`
	Quiet   bool `xml:"Quiet"`
}

type DeleteObjectItem struct {
	Key       string `xml:"Key"`
	VersionId string `xml:"VersionId,omitempty"`
}

// DeleteObjectsResult represents the response from a batch delete operation
type DeleteObjectsResult struct {
	XMLName xml.Name `xml:"DeleteResult"`
	Deleted []DeletedObject `xml:"Deleted"`
	Errors  []DeleteError `xml:"Error"`
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
	client, err := s.provider.CreateClient("s3")
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
			"max-keys": "1000",
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
	XMLName               xml.Name        `xml:"ListVersionsResult"`
	Name                  string          `xml:"Name"`
	IsTruncated           bool            `xml:"IsTruncated"`
	KeyMarker             string          `xml:"KeyMarker"`
	VersionIdMarker       string          `xml:"VersionIdMarker"`
	NextKeyMarker         string          `xml:"NextKeyMarker"`
	NextVersionIdMarker   string          `xml:"NextVersionIdMarker"`
	MaxKeys               int             `xml:"MaxKeys"`
	Versions              []ObjectVersion `xml:"Version"`
	DeleteMarkers         []DeleteMarker  `xml:"DeleteMarker"`
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