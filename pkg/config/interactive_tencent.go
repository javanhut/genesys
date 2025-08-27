package config

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

// getTencentCredentials interactively gets Tencent Cloud credentials
func (ic *InteractiveConfig) getTencentCredentials() (map[string]string, error) {
	fmt.Println("\nTencent Cloud Credential Configuration")
	fmt.Println("Please provide your Tencent Cloud credentials. You can find these in:")
	fmt.Println("  • Tencent Cloud Console → Cloud Access Management → API Keys")
	fmt.Println("  • https://console.cloud.tencent.com/cam/capi")
	fmt.Println("")

	credentials := make(map[string]string)

	// Secret ID
	var secretID string
	secretIDPrompt := &survey.Input{
		Message: "Tencent Cloud Secret ID:",
		Help:    "Your SecretId from Tencent Cloud API key management",
	}
	if err := survey.AskOne(secretIDPrompt, &secretID, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["secret_id"] = secretID

	// Secret Key
	var secretKey string
	secretKeyPrompt := &survey.Password{
		Message: "Tencent Cloud Secret Key:",
		Help:    "Your SecretKey from Tencent Cloud API key management",
	}
	if err := survey.AskOne(secretKeyPrompt, &secretKey, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["secret_key"] = secretKey

	// Optional: Security Token
	var useSecurityToken bool
	tokenPrompt := &survey.Confirm{
		Message: "Do you need to provide a security token (for temporary credentials)?",
		Default: false,
	}
	if err := survey.AskOne(tokenPrompt, &useSecurityToken); err != nil {
		return nil, err
	}

	if useSecurityToken {
		var securityToken string
		securityTokenPrompt := &survey.Password{
			Message: "Tencent Cloud Security Token:",
			Help:    "Temporary security token (for federated users or role assumption)",
		}
		if err := survey.AskOne(securityTokenPrompt, &securityToken); err != nil {
			return nil, err
		}
		credentials["security_token"] = securityToken
	}

	return credentials, nil
}

// getTencentRegion gets Tencent Cloud region selection
func (ic *InteractiveConfig) getTencentRegion() (string, error) {
	tencentRegions := []string{
		"ap-beijing",      // Beijing
		"ap-beijing-1",    // Beijing Zone 1
		"ap-beijing-2",    // Beijing Zone 2
		"ap-beijing-3",    // Beijing Zone 3
		"ap-beijing-4",    // Beijing Zone 4
		"ap-beijing-5",    // Beijing Zone 5
		"ap-chengdu",      // Chengdu
		"ap-chongqing",    // Chongqing
		"ap-guangzhou",    // Guangzhou
		"ap-guangzhou-2",  // Guangzhou Zone 2
		"ap-guangzhou-3",  // Guangzhou Zone 3
		"ap-guangzhou-4",  // Guangzhou Zone 4
		"ap-guangzhou-6",  // Guangzhou Zone 6
		"ap-guangzhou-7",  // Guangzhou Zone 7
		"ap-hongkong",     // Hong Kong
		"ap-mumbai",       // Mumbai
		"ap-nanjing",      // Nanjing
		"ap-seoul",        // Seoul
		"ap-shanghai",     // Shanghai
		"ap-shanghai-2",   // Shanghai Zone 2
		"ap-shanghai-3",   // Shanghai Zone 3
		"ap-shanghai-4",   // Shanghai Zone 4
		"ap-shanghai-5",   // Shanghai Zone 5
		"ap-shenzhen-fsi", // Shenzhen Finance
		"ap-singapore",    // Singapore
		"ap-tokyo",        // Tokyo
		"eu-frankfurt",    // Frankfurt
		"eu-moscow",       // Moscow
		"na-ashburn",      // Virginia
		"na-siliconvalley", // Silicon Valley
		"na-toronto",      // Toronto
	}

	regionDescriptions := map[string]string{
		"ap-beijing":       "Beijing",
		"ap-beijing-1":     "Beijing Zone 1",
		"ap-beijing-2":     "Beijing Zone 2",
		"ap-beijing-3":     "Beijing Zone 3",
		"ap-beijing-4":     "Beijing Zone 4",
		"ap-beijing-5":     "Beijing Zone 5",
		"ap-chengdu":       "Chengdu",
		"ap-chongqing":     "Chongqing",
		"ap-guangzhou":     "Guangzhou",
		"ap-guangzhou-2":   "Guangzhou Zone 2",
		"ap-guangzhou-3":   "Guangzhou Zone 3",
		"ap-guangzhou-4":   "Guangzhou Zone 4",
		"ap-guangzhou-6":   "Guangzhou Zone 6",
		"ap-guangzhou-7":   "Guangzhou Zone 7",
		"ap-hongkong":      "Hong Kong",
		"ap-mumbai":        "Mumbai",
		"ap-nanjing":       "Nanjing",
		"ap-seoul":         "Seoul",
		"ap-shanghai":      "Shanghai",
		"ap-shanghai-2":    "Shanghai Zone 2",
		"ap-shanghai-3":    "Shanghai Zone 3",
		"ap-shanghai-4":    "Shanghai Zone 4",
		"ap-shanghai-5":    "Shanghai Zone 5",
		"ap-shenzhen-fsi":  "Shenzhen Finance",
		"ap-singapore":     "Singapore",
		"ap-tokyo":         "Tokyo",
		"eu-frankfurt":     "Frankfurt",
		"eu-moscow":        "Moscow",
		"na-ashburn":       "Virginia",
		"na-siliconvalley": "Silicon Valley",
		"na-toronto":       "Toronto",
	}

	var selectedRegion string
	prompt := &survey.Select{
		Message:  "Select your preferred Tencent Cloud region:",
		Options:  tencentRegions,
		Default:  "ap-guangzhou",
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