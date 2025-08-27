package config

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

// getAWSCredentials interactively gets AWS credentials
func (ic *InteractiveConfig) getAWSCredentials() (map[string]string, error) {
	fmt.Println("\nAWS Credential Configuration")
	fmt.Println("Please provide your AWS credentials. You can find these in:")
	fmt.Println("  • AWS Console → IAM → Users → Security credentials")
	fmt.Println("  • AWS CLI: aws configure list")
	fmt.Println("")

	credentials := make(map[string]string)

	// AWS Access Key ID
	var accessKey string
	var accessKeyPrompt = &survey.Input{
		Message: "AWS Access Key ID:",
		Help:    "Your AWS access key (starts with AKIA...)",
	}
	if err := survey.AskOne(accessKeyPrompt, &accessKey, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["access_key_id"] = accessKey

	// AWS Secret Access Key
	var secretKey string
	var secretKeyPrompt = &survey.Password{
		Message: "AWS Secret Access Key:",
		Help:    "Your AWS secret access key (40 characters)",
	}
	if err := survey.AskOne(secretKeyPrompt, &secretKey, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["secret_access_key"] = secretKey

	// Optional: Session Token
	var useSessionToken bool
	sessionPrompt := &survey.Confirm{
		Message: "Do you need to provide a session token (for temporary credentials)?",
		Default: false,
	}
	if err := survey.AskOne(sessionPrompt, &useSessionToken); err != nil {
		return nil, err
	}

	if useSessionToken {
		var sessionToken string
		var sessionTokenPrompt = &survey.Password{
			Message: "AWS Session Token:",
			Help:    "Temporary session token (for assumed roles or STS)",
		}
		if err := survey.AskOne(sessionTokenPrompt, &sessionToken); err != nil {
			return nil, err
		}
		credentials["session_token"] = sessionToken
	}

	// Optional: Profile name
	var profile string
	profilePrompt := &survey.Input{
		Message: "AWS Profile name (optional):",
		Help:    "Named profile for organizing multiple AWS accounts",
		Default: "default",
	}
	if err := survey.AskOne(profilePrompt, &profile); err != nil {
		return nil, err
	}
	if profile != "" && profile != "default" {
		credentials["profile"] = profile
	}

	return credentials, nil
}

// getAWSRegion gets AWS region selection
func (ic *InteractiveConfig) getAWSRegion() (string, error) {
	awsRegions := []string{
		"us-east-1",      // US East (N. Virginia)
		"us-east-2",      // US East (Ohio)
		"us-west-1",      // US West (N. California)
		"us-west-2",      // US West (Oregon)
		"eu-west-1",      // Europe (Ireland)
		"eu-west-2",      // Europe (London)
		"eu-west-3",      // Europe (Paris)
		"eu-central-1",   // Europe (Frankfurt)
		"eu-north-1",     // Europe (Stockholm)
		"ap-east-1",      // Asia Pacific (Hong Kong)
		"ap-south-1",     // Asia Pacific (Mumbai)
		"ap-southeast-1", // Asia Pacific (Singapore)
		"ap-southeast-2", // Asia Pacific (Sydney)
		"ap-northeast-1", // Asia Pacific (Tokyo)
		"ap-northeast-2", // Asia Pacific (Seoul)
		"ap-northeast-3", // Asia Pacific (Osaka)
		"ca-central-1",   // Canada (Central)
		"sa-east-1",      // South America (São Paulo)
		"af-south-1",     // Africa (Cape Town)
		"me-south-1",     // Middle East (Bahrain)
	}

	regionDescriptions := map[string]string{
		"us-east-1":      "US East (N. Virginia)",
		"us-east-2":      "US East (Ohio)",
		"us-west-1":      "US West (N. California)",
		"us-west-2":      "US West (Oregon)",
		"eu-west-1":      "Europe (Ireland)",
		"eu-west-2":      "Europe (London)",
		"eu-west-3":      "Europe (Paris)",
		"eu-central-1":   "Europe (Frankfurt)",
		"eu-north-1":     "Europe (Stockholm)",
		"ap-east-1":      "Asia Pacific (Hong Kong)",
		"ap-south-1":     "Asia Pacific (Mumbai)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
		"ap-southeast-2": "Asia Pacific (Sydney)",
		"ap-northeast-1": "Asia Pacific (Tokyo)",
		"ap-northeast-2": "Asia Pacific (Seoul)",
		"ap-northeast-3": "Asia Pacific (Osaka)",
		"ca-central-1":   "Canada (Central)",
		"sa-east-1":      "South America (São Paulo)",
		"af-south-1":     "Africa (Cape Town)",
		"me-south-1":     "Middle East (Bahrain)",
	}

	var selectedRegion string
	prompt := &survey.Select{
		Message:  "Select your preferred AWS region:",
		Options:  awsRegions,
		Default:  "us-east-1",
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