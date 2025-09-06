package aws

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

// NewAWSClient creates a new AWS client for direct API calls
func NewAWSClient(region, service string) (*AWSClient, error) {
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("AWS credentials not found in environment variables")
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

// Request makes an authenticated AWS API request
func (c *AWSClient) Request(method, endpoint string, params map[string]string, body []byte) (*http.Response, error) {
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

	// Set headers
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))

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

	// Create canonical request
	canonicalURI := req.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalQuerystring := req.URL.RawQuery
	if canonicalQuerystring == "" {
		canonicalQuerystring = ""
	}

	// Create canonical headers
	var headerNames []string
	canonicalHeaders := ""
	
	for name, values := range req.Header {
		name = strings.ToLower(name)
		headerNames = append(headerNames, name)
		canonicalHeaders += name + ":" + strings.Join(values, ",") + "\n"
	}
	
	// Add host header
	headerNames = append(headerNames, "host")
	canonicalHeaders += "host:" + req.Host + "\n"
	
	sort.Strings(headerNames)
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
	req.Header.Set("X-Amz-Date", timestamp)

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

// ReadResponse reads and returns the response body
func ReadResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}