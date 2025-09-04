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
	// Build URL
	baseURL := fmt.Sprintf("https://%s.%s.amazonaws.com", c.Service, c.Region)
	if endpoint != "" {
		baseURL += "/" + strings.TrimPrefix(endpoint, "/")
	}

	// Add query parameters
	if len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}
		baseURL += "?" + values.Encode()
	}

	// Create request
	req, err := http.NewRequest(method, baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Calculate payload hash for x-amz-content-sha256
	payloadHash := fmt.Sprintf("%x", sha256.Sum256(body))

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
	} else {
		req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	}

	if c.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", c.SessionToken)
	}

	// Sign the request
	if err := c.signRequest(req, body); err != nil {
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
	credentialScope := datestamp + "/" + c.Region + "/" + c.Service + "/aws4_request"
	stringToSign := strings.Join([]string{
		algorithm,
		timestamp,
		credentialScope,
		fmt.Sprintf("%x", sha256.Sum256([]byte(canonicalRequest))),
	}, "\n")

	// Calculate signature
	signature := c.calculateSignature(stringToSign, datestamp)

	// Create authorization header
	authorizationHeader := algorithm + " " +
		"Credential=" + c.AccessKey + "/" + credentialScope + ", " +
		"SignedHeaders=" + signedHeaders + ", " +
		"Signature=" + signature

	req.Header.Set("Authorization", authorizationHeader)

	return nil
}

// calculateSignature calculates the AWS4-HMAC-SHA256 signature
func (c *AWSClient) calculateSignature(stringToSign, datestamp string) string {
	kDate := hmacSHA256([]byte("AWS4"+c.SecretKey), datestamp)
	kRegion := hmacSHA256(kDate, c.Region)
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