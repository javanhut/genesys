package aws

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AWSClient provides direct AWS API access without the heavy SDK
type AWSClient struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
	Region       string
	Service      string
	HTTPClient   *http.Client
}

// ProviderCredentials represents stored credentials (matching config package)
type ProviderCredentials struct {
	Provider      string            `json:"provider"`
	Region        string            `json:"region"`
	Credentials   map[string]string `json:"credentials"`
	UseLocal      bool              `json:"use_local"`
	DefaultConfig bool              `json:"default_config"`
	ExpiresAt     *time.Time        `json:"expires_at,omitempty"`
	LastRefreshed time.Time         `json:"last_refreshed"`
}

// NewAWSClient creates a new AWS client for direct API calls
func NewAWSClient(region, service string) (*AWSClient, error) {
	// First try environment variables
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	// If not in environment, try loading from config file
	if accessKey == "" || secretKey == "" {
		creds, err := loadAWSCredentialsFromConfig()
		if err == nil && creds != nil {
			// Check if credentials are expired and need refresh
			if creds.ExpiresAt != nil && time.Now().After(*creds.ExpiresAt) {
				// Try to refresh credentials
				refreshedCreds, refreshErr := refreshAWSCredentials(creds)
				if refreshErr == nil {
					creds = refreshedCreds
				} else {
					fmt.Printf("Warning: Credentials expired and refresh failed: %v\n", refreshErr)
				}
			}

			if creds.UseLocal {
				// If using local credentials, still need to read from environment
				// but we know they should exist
			} else {
				// Use credentials from config file
				if ak, ok := creds.Credentials["access_key_id"]; ok {
					accessKey = ak
				}
				if sk, ok := creds.Credentials["secret_access_key"]; ok {
					secretKey = sk
				}
				if st, ok := creds.Credentials["session_token"]; ok {
					sessionToken = st
				}
			}
			// Use region from config if not specified
			if region == "" && creds.Region != "" {
				region = creds.Region
			}
		}
	}

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("AWS credentials not found in environment variables or configuration file")
	}

	return &AWSClient{
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		SessionToken: sessionToken,
		Region:       region,
		Service:      service,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// loadAWSCredentialsFromConfig loads AWS credentials from ~/.genesys/aws.json
func loadAWSCredentialsFromConfig() (*ProviderCredentials, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(homeDir, ".genesys", "aws.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var creds ProviderCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

// refreshAWSCredentials attempts to refresh AWS credentials
func refreshAWSCredentials(creds *ProviderCredentials) (*ProviderCredentials, error) {
	// For now, just try to reload from the credentials file
	// In the future, this could integrate with AWS STS for automatic token refresh

	// Try to read from AWS credentials file if using local credentials
	if creds.UseLocal {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		credFile := filepath.Join(homeDir, ".aws", "credentials")
		if _, err := os.Stat(credFile); err == nil {
			// Parse AWS credentials file
			refreshedCreds, err := parseAWSCredentialsFile(credFile, creds.Region)
			if err != nil {
				return nil, fmt.Errorf("failed to parse AWS credentials file: %w", err)
			}

			// Update timestamps
			refreshedCreds.LastRefreshed = time.Now()

			// Save updated credentials
			if err := saveAWSCredentialsToConfig(refreshedCreds); err != nil {
				return nil, fmt.Errorf("failed to save refreshed credentials: %w", err)
			}

			return refreshedCreds, nil
		}
	}

	return nil, fmt.Errorf("credential refresh not supported for this configuration")
}

// parseAWSCredentialsFile parses the AWS credentials file
func parseAWSCredentialsFile(credFile, region string) (*ProviderCredentials, error) {
	data, err := os.ReadFile(credFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	credentials := make(map[string]string)
	var expiresAt *time.Time

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "[") || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "aws_access_key_id":
			credentials["access_key_id"] = value
		case "aws_secret_access_key":
			credentials["secret_access_key"] = value
		case "aws_session_token", "aws_security_token":
			credentials["session_token"] = value
		case "x_security_token_expires":
			if expTime, err := time.Parse(time.RFC3339, value); err == nil {
				expiresAt = &expTime
			}
		}
	}

	return &ProviderCredentials{
		Provider:      "aws",
		Region:        region,
		Credentials:   credentials,
		UseLocal:      true,
		DefaultConfig: true,
		ExpiresAt:     expiresAt,
		LastRefreshed: time.Now(),
	}, nil
}

// saveAWSCredentialsToConfig saves credentials to the config file
func saveAWSCredentialsToConfig(creds *ProviderCredentials) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".genesys")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "aws.json")
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

// ValidateAWSCredentials validates AWS credentials by making a test API call
func ValidateAWSCredentials(accessKey, secretKey, sessionToken, region string) error {
	client := &AWSClient{
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		SessionToken: sessionToken,
		Region:       region,
		Service:      "sts",
		HTTPClient:   &http.Client{Timeout: 10 * time.Second},
	}

	// Make a GetCallerIdentity call to validate credentials
	params := map[string]string{
		"Action":  "GetCallerIdentity",
		"Version": "2011-06-15",
	}

	resp, err := client.Request("POST", "", params, nil)
	if err != nil {
		return fmt.Errorf("credential validation failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("credential validation failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

// RefreshAndValidateCredentials refreshes and validates stored credentials
func RefreshAndValidateCredentials() error {
	creds, err := loadAWSCredentialsFromConfig()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	if creds == nil {
		return fmt.Errorf("no credentials found")
	}

	// Check if credentials need refresh
	if creds.ExpiresAt != nil && time.Now().After(*creds.ExpiresAt) {
		refreshedCreds, err := refreshAWSCredentials(creds)
		if err != nil {
			return fmt.Errorf("failed to refresh credentials: %w", err)
		}
		creds = refreshedCreds
	}

	// Validate credentials
	var accessKey, secretKey, sessionToken string
	if ak, ok := creds.Credentials["access_key_id"]; ok {
		accessKey = ak
	}
	if sk, ok := creds.Credentials["secret_access_key"]; ok {
		secretKey = sk
	}
	if st, ok := creds.Credentials["session_token"]; ok {
		sessionToken = st
	}

	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("incomplete credentials")
	}

	return ValidateAWSCredentials(accessKey, secretKey, sessionToken, creds.Region)
}

// isGlobalService returns true if the AWS service is global and doesn't use regional endpoints
func isGlobalService(service string) bool {
	globalServices := map[string]bool{
		"iam":             true,
		"sts":             true,
		"cloudfront":      true,
		"waf":             true,
		"route53":         true,
		"route53resolver": false, // Regional
		"shield":          true,
		"support":         true,
		"trustedadvisor":  true,
	}
	return globalServices[service]
}

// getSigningRegion returns the region to use for AWS signing - us-east-1 for global services
func (c *AWSClient) getSigningRegion() string {
	if isGlobalService(c.Service) {
		return "us-east-1"
	}
	return c.Region
}

// RequestWithMD5 makes an authenticated AWS API request with Content-MD5 header
func (c *AWSClient) RequestWithMD5(method, endpoint string, params map[string]string, body []byte) (*http.Response, error) {
	return c.requestInternal(method, endpoint, params, body, true)
}

// Request makes an authenticated AWS API request
func (c *AWSClient) Request(method, endpoint string, params map[string]string, body []byte) (*http.Response, error) {
	return c.requestInternal(method, endpoint, params, body, false)
}

// requestInternal is the internal request method
func (c *AWSClient) requestInternal(method, endpoint string, params map[string]string, body []byte, includeMD5 bool) (*http.Response, error) {
	// Build URL - handle global services that don't use regional endpoints
	var baseURL string
	if isGlobalService(c.Service) {
		baseURL = fmt.Sprintf("https://%s.amazonaws.com", c.Service)
	} else {
		baseURL = fmt.Sprintf("https://%s.%s.amazonaws.com", c.Service, c.Region)
	}
	if endpoint != "" {
		// Don't add extra slash if endpoint already starts with one
		if !strings.HasPrefix(endpoint, "/") {
			baseURL += "/"
		}
		baseURL += endpoint
	}

	// Handle parameters based on service and method
	var requestBody []byte
	if len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}

		// For IAM/STS/EC2 POST requests, put parameters in the body
		// EC2 Query API expects form-encoded parameters in the POST body
		if (c.Service == "iam" || c.Service == "sts" || c.Service == "ec2") && method == "POST" {
			requestBody = []byte(values.Encode())
		} else {
			baseURL += "?" + values.Encode()
			requestBody = body
		}
	} else {
		requestBody = body
	}

	// Create request
	req, err := http.NewRequest(method, baseURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Calculate payload hash for x-amz-content-sha256
	payloadHash := fmt.Sprintf("%x", sha256.Sum256(requestBody))

	// Set headers
	if c.Service == "s3" {
		// S3 requires specific content type handling
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/xml")
		}
		req.Header.Set("x-amz-content-sha256", payloadHash)

		// Add Content-MD5 header if requested (required for some S3 operations like tagging)
		if includeMD5 && len(body) > 0 {
			md5sum := md5.Sum(body)
			req.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(md5sum[:]))
		}
	} else if c.Service == "iam" || c.Service == "sts" || c.Service == "ec2" {
		// IAM, STS, and EC2 use form encoding for POST requests (Query API)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	} else {
		req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	}

	if c.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", c.SessionToken)
	}

	// Sign the request
	if err := c.signRequest(req, requestBody); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Make the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// signRequest signs the request using AWS Signature Version 4
func (c *AWSClient) signRequest(req *http.Request, body []byte) error {
	now := time.Now().UTC()
	datestamp := now.Format("20060102")
	timestamp := now.Format("20060102T150405Z")

	// Update the X-Amz-Date header with the current timestamp
	req.Header.Set("X-Amz-Date", timestamp)

	// Create canonical request
	canonicalURI := req.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalQuerystring := req.URL.RawQuery

	// Create canonical headers - need to be sorted alphabetically
	headerMap := make(map[string]string)
	var headerNames []string

	// Add all headers
	for name, values := range req.Header {
		lowerName := strings.ToLower(name)
		if lowerName == "host" || strings.HasPrefix(lowerName, "x-amz-") || lowerName == "content-type" || lowerName == "content-md5" {
			headerMap[lowerName] = strings.TrimSpace(strings.Join(values, ","))
			headerNames = append(headerNames, lowerName)
		}
	}

	// Add host header (required)
	if req.URL.Host != "" {
		headerMap["host"] = req.URL.Host
	} else {
		headerMap["host"] = req.Host
	}
	if !contains(headerNames, "host") {
		headerNames = append(headerNames, "host")
	}

	// Sort header names
	sort.Strings(headerNames)

	// Build canonical headers string
	canonicalHeaders := ""
	for _, name := range headerNames {
		canonicalHeaders += name + ":" + headerMap[name] + "\n"
	}

	signedHeaders := strings.Join(headerNames, ";")

	// Create payload hash
	payloadHash := fmt.Sprintf("%x", sha256.Sum256(body))

	// Create canonical request
	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQuerystring,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// Create string to sign
	algorithm := "AWS4-HMAC-SHA256"
	signingRegion := c.getSigningRegion()
	credentialScope := datestamp + "/" + signingRegion + "/" + c.Service + "/aws4_request"
	stringToSign := strings.Join([]string{
		algorithm,
		timestamp,
		credentialScope,
		fmt.Sprintf("%x", sha256.Sum256([]byte(canonicalRequest))),
	}, "\n")

	// Calculate signature
	signature := c.calculateSignature(stringToSign, datestamp, signingRegion)

	// Create authorization header
	authorizationHeader := algorithm + " " +
		"Credential=" + c.AccessKey + "/" + credentialScope + ", " +
		"SignedHeaders=" + signedHeaders + ", " +
		"Signature=" + signature

	req.Header.Set("Authorization", authorizationHeader)

	return nil
}

// calculateSignature calculates the AWS4-HMAC-SHA256 signature
func (c *AWSClient) calculateSignature(stringToSign, datestamp, signingRegion string) string {
	kDate := hmacSHA256([]byte("AWS4"+c.SecretKey), datestamp)
	kRegion := hmacSHA256(kDate, signingRegion)
	kService := hmacSHA256(kRegion, c.Service)
	kSigning := hmacSHA256(kService, "aws4_request")
	signature := hmacSHA256(kSigning, stringToSign)
	return fmt.Sprintf("%x", signature)
}

// hmacSHA256 computes HMAC-SHA256
func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// contains checks if a string slice contains a value
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// ReadResponse reads and returns the response body
func ReadResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
