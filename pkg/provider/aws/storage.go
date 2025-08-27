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

// DeleteBucket deletes a bucket
func (s *StorageService) DeleteBucket(ctx context.Context, name string) error {
	client, err := s.provider.CreateClient("s3")
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s", name)
	resp, err := client.Request("DELETE", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("DeleteBucket failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
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