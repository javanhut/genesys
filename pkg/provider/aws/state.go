package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/javanhut/genesys/pkg/provider"
)

// StateBackend implements S3-based state storage using direct API calls
type StateBackend struct {
	provider   *AWSProvider
	bucketName string
}

// NewStateBackend creates a new state backend
func NewStateBackend(p *AWSProvider) *StateBackend {
	return &StateBackend{
		provider:   p,
		bucketName: "genesys-state-" + p.region, // Default bucket name
	}
}

// Init initializes the state backend
func (s *StateBackend) Init(ctx context.Context) error {
	// Check if state bucket exists, create if not
	client, err := s.provider.CreateClient("s3")
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Try to get bucket location to see if it exists
	endpoint := fmt.Sprintf("/%s?location", s.bucketName)
	resp, err := client.Request("GET", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// Bucket doesn't exist, create it
		if err := s.createStateBucket(client); err != nil {
			return fmt.Errorf("failed to create state bucket: %w", err)
		}
	}

	return nil
}

// Lock locks the state for exclusive access
func (s *StateBackend) Lock(ctx context.Context, key string) error {
	// Simple implementation - in production you'd want DynamoDB-based locking
	client, err := s.provider.CreateClient("s3")
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	lockKey := key + ".lock"
	lockData := map[string]interface{}{
		"locked_at": time.Now().Format(time.RFC3339),
		"locked_by": "genesys",
	}

	data, err := json.Marshal(lockData)
	if err != nil {
		return fmt.Errorf("failed to marshal lock data: %w", err)
	}

	endpoint := fmt.Sprintf("/%s/%s", s.bucketName, lockKey)
	resp, err := client.Request("PUT", endpoint, nil, data)
	if err != nil {
		return fmt.Errorf("failed to create lock: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("failed to create lock with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

// Unlock releases the state lock
func (s *StateBackend) Unlock(ctx context.Context, key string) error {
	client, err := s.provider.CreateClient("s3")
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	lockKey := key + ".lock"
	endpoint := fmt.Sprintf("/%s/%s", s.bucketName, lockKey)

	resp, err := client.Request("DELETE", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete lock: %w", err)
	}
	defer resp.Body.Close()

	// 404 is OK - lock might not exist
	if resp.StatusCode != 204 && resp.StatusCode != 404 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("failed to delete lock with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

// Read reads state from storage
func (s *StateBackend) Read(ctx context.Context, key string) (*provider.State, error) {
	client, err := s.provider.CreateClient("s3")
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s/%s", s.bucketName, key)
	resp, err := client.Request("GET", endpoint, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// State doesn't exist, return empty state
		return &provider.State{
			Version:   1,
			Resources: make(map[string]interface{}),
			Outputs:   make(map[string]interface{}),
			UpdatedAt: time.Now(),
		}, nil
	}

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return nil, fmt.Errorf("failed to get state with status %d: %s", resp.StatusCode, string(responseBody))
	}

	responseBody, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var state provider.State
	if err := json.Unmarshal(responseBody, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// Write writes state to storage
func (s *StateBackend) Write(ctx context.Context, key string, state *provider.State) error {
	client, err := s.provider.CreateClient("s3")
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	state.UpdatedAt = time.Now()

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	endpoint := fmt.Sprintf("/%s/%s", s.bucketName, key)
	resp, err := client.Request("PUT", endpoint, nil, data)
	if err != nil {
		return fmt.Errorf("failed to put state: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("failed to put state with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

// createStateBucket creates the S3 bucket for state storage
func (s *StateBackend) createStateBucket(client *AWSClient) error {
	endpoint := fmt.Sprintf("/%s", s.bucketName)

	var body []byte
	if s.provider.region != "us-east-1" {
		// Need to specify location constraint for regions other than us-east-1
		locationXML := fmt.Sprintf(`<CreateBucketConfiguration><LocationConstraint>%s</LocationConstraint></CreateBucketConfiguration>`, s.provider.region)
		body = []byte(locationXML)
	}

	resp, err := client.Request("PUT", endpoint, nil, body)
	if err != nil {
		return fmt.Errorf("failed to create state bucket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("failed to create state bucket with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Enable versioning for state bucket
	versioningXML := `<VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>`
	versioningEndpoint := fmt.Sprintf("/%s?versioning", s.bucketName)

	versioningResp, err := client.Request("PUT", versioningEndpoint, nil, []byte(versioningXML))
	if err != nil {
		return fmt.Errorf("failed to enable versioning: %w", err)
	}
	defer versioningResp.Body.Close()

	if versioningResp.StatusCode != 200 {
		responseBody, _ := ReadResponse(versioningResp)
		return fmt.Errorf("failed to enable versioning with status %d: %s", versioningResp.StatusCode, string(responseBody))
	}

	return nil
}

// Refresh forces a refresh of the state from the remote storage
func (s *StateBackend) Refresh(ctx context.Context, key string) (*provider.State, error) {
	// This is essentially the same as Read, but we ensure no local caching
	return s.Read(ctx, key)
}

// ListStates lists all available state keys in the bucket
func (s *StateBackend) ListStates(ctx context.Context) ([]string, error) {
	client, err := s.provider.CreateClient("s3")
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s", s.bucketName)
	resp, err := client.Request("GET", endpoint, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return nil, fmt.Errorf("failed to list objects with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse S3 ListObjects response (simplified)
	// TODO: Implement proper XML parsing for S3 ListObjects response
	var keys []string
	return keys, nil
}

// ValidateState checks if the state is consistent and valid
func (s *StateBackend) ValidateState(ctx context.Context, key string) error {
	state, err := s.Read(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to read state for validation: %w", err)
	}

	if state == nil {
		return fmt.Errorf("state is nil")
	}

	if state.Version == 0 {
		return fmt.Errorf("state version is invalid")
	}

	return nil
}
