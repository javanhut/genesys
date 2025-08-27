package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
)

// getGCPCredentials interactively gets GCP credentials
func (ic *InteractiveConfig) getGCPCredentials() (map[string]string, error) {
	fmt.Println("\nGoogle Cloud Platform Credential Configuration")
	fmt.Println("Choose your GCP authentication method:")
	fmt.Println("  • Service Account Key: JSON file from GCP Console")
	fmt.Println("  • Application Default Credentials: From gcloud CLI")
	fmt.Println("")

	credentials := make(map[string]string)

	// Authentication method selection
	authMethods := []string{
		"service_account_key",
		"application_default",
		"service_account_email",
	}

	authMethodLabels := map[string]string{
		"service_account_key": "Service Account Key (JSON file)",
		"application_default": "Application Default Credentials (gcloud)",
		"service_account_email": "Service Account Email + Key",
	}

	var authMethod string
	authPrompt := &survey.Select{
		Message: "Select authentication method:",
		Options: authMethods,
		Description: func(value string, index int) string {
			return authMethodLabels[value]
		},
	}
	if err := survey.AskOne(authPrompt, &authMethod); err != nil {
		return nil, err
	}

	credentials["auth_method"] = authMethod

	switch authMethod {
	case "service_account_key":
		return ic.getGCPServiceAccountKey(credentials)
	case "application_default":
		return ic.getGCPApplicationDefault(credentials)
	case "service_account_email":
		return ic.getGCPServiceAccountEmail(credentials)
	default:
		return credentials, nil
	}
}

func (ic *InteractiveConfig) getGCPServiceAccountKey(credentials map[string]string) (map[string]string, error) {
	// Service Account Key File Path
	var keyFilePath string
	keyFilePrompt := &survey.Input{
		Message: "Path to service account key file (JSON):",
		Help:    "Full path to your service account JSON key file",
	}
	if err := survey.AskOne(keyFilePrompt, &keyFilePath, survey.WithValidator(func(val interface{}) error {
		str := val.(string)
		if str == "" {
			return fmt.Errorf("key file path is required")
		}
		
		// Expand tilde to home directory
		if str[0] == '~' {
			home, _ := os.UserHomeDir()
			str = filepath.Join(home, str[1:])
		}
		
		// Check if file exists
		if _, err := os.Stat(str); err != nil {
			return fmt.Errorf("key file does not exist: %s", str)
		}
		return nil
	})); err != nil {
		return nil, err
	}

	credentials["service_account_key"] = keyFilePath

	// Project ID
	var projectID string
	projectPrompt := &survey.Input{
		Message: "GCP Project ID:",
		Help:    "Your Google Cloud project ID (not the project name)",
	}
	if err := survey.AskOne(projectPrompt, &projectID, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["project_id"] = projectID

	return credentials, nil
}

func (ic *InteractiveConfig) getGCPApplicationDefault(credentials map[string]string) (map[string]string, error) {
	fmt.Println("Using Application Default Credentials (ADC)")
	fmt.Println("Make sure you have run: gcloud auth application-default login")
	
	// Project ID
	var projectID string
	projectPrompt := &survey.Input{
		Message: "GCP Project ID:",
		Help:    "Your Google Cloud project ID (not the project name)",
	}
	if err := survey.AskOne(projectPrompt, &projectID, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["project_id"] = projectID

	return credentials, nil
}

func (ic *InteractiveConfig) getGCPServiceAccountEmail(credentials map[string]string) (map[string]string, error) {
	// Service Account Email
	var serviceAccount string
	emailPrompt := &survey.Input{
		Message: "Service Account Email:",
		Help:    "Format: service-account@project-id.iam.gserviceaccount.com",
	}
	if err := survey.AskOne(emailPrompt, &serviceAccount, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["service_account_email"] = serviceAccount

	// Private Key
	var privateKey string
	keyPrompt := &survey.Password{
		Message: "Private Key (PEM format):",
		Help:    "The private key from your service account JSON file",
	}
	if err := survey.AskOne(keyPrompt, &privateKey, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["private_key"] = privateKey

	// Project ID
	var projectID string
	projectPrompt := &survey.Input{
		Message: "GCP Project ID:",
		Help:    "Your Google Cloud project ID (not the project name)",
	}
	if err := survey.AskOne(projectPrompt, &projectID, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["project_id"] = projectID

	return credentials, nil
}

// getGCPRegion gets GCP region selection
func (ic *InteractiveConfig) getGCPRegion() (string, error) {
	gcpRegions := []string{
		"us-central1",    // Iowa
		"us-east1",       // South Carolina
		"us-east4",       // Northern Virginia
		"us-west1",       // Oregon
		"us-west2",       // Los Angeles
		"us-west3",       // Salt Lake City
		"us-west4",       // Las Vegas
		"europe-west1",   // Belgium
		"europe-west2",   // London
		"europe-west3",   // Frankfurt
		"europe-west4",   // Netherlands
		"europe-west6",   // Zurich
		"europe-north1",  // Finland
		"asia-east1",     // Taiwan
		"asia-east2",     // Hong Kong
		"asia-northeast1", // Tokyo
		"asia-northeast2", // Osaka
		"asia-northeast3", // Seoul
		"asia-south1",    // Mumbai
		"asia-southeast1", // Singapore
		"asia-southeast2", // Jakarta
		"australia-southeast1", // Sydney
		"northamerica-northeast1", // Montreal
		"southamerica-east1", // São Paulo
	}

	regionDescriptions := map[string]string{
		"us-central1":              "Iowa",
		"us-east1":                 "South Carolina",
		"us-east4":                 "Northern Virginia",
		"us-west1":                 "Oregon",
		"us-west2":                 "Los Angeles",
		"us-west3":                 "Salt Lake City",
		"us-west4":                 "Las Vegas",
		"europe-west1":             "Belgium",
		"europe-west2":             "London",
		"europe-west3":             "Frankfurt",
		"europe-west4":             "Netherlands",
		"europe-west6":             "Zurich",
		"europe-north1":            "Finland",
		"asia-east1":               "Taiwan",
		"asia-east2":               "Hong Kong",
		"asia-northeast1":          "Tokyo",
		"asia-northeast2":          "Osaka",
		"asia-northeast3":          "Seoul",
		"asia-south1":              "Mumbai",
		"asia-southeast1":          "Singapore",
		"asia-southeast2":          "Jakarta",
		"australia-southeast1":     "Sydney",
		"northamerica-northeast1":  "Montreal",
		"southamerica-east1":       "São Paulo",
	}

	var selectedRegion string
	prompt := &survey.Select{
		Message:  "Select your preferred GCP region:",
		Options:  gcpRegions,
		Default:  "us-central1",
		PageSize: 10,
		Description: func(value string, index int) string {
			return regionDescriptions[value]
		},
	}

	if err := survey.AskOne(prompt, &selectedRegion); err != nil {
		return "", err
	}

	return selectedRegion, nil
}