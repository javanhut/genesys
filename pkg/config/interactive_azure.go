package config

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

// getAzureCredentials interactively gets Azure credentials
func (ic *InteractiveConfig) getAzureCredentials() (map[string]string, error) {
	fmt.Println("\nMicrosoft Azure Credential Configuration")
	fmt.Println("Choose your Azure authentication method:")
	fmt.Println("  • Service Principal: Recommended for automation")
	fmt.Println("  • Azure CLI: Use existing 'az login' authentication")
	fmt.Println("  • Managed Identity: For Azure VMs/App Services")
	fmt.Println("")

	credentials := make(map[string]string)

	// Authentication method selection
	authMethods := []string{
		"service_principal",
		"azure_cli",
		"managed_identity",
	}

	authMethodLabels := map[string]string{
		"service_principal": "Service Principal (Client ID + Secret)",
		"azure_cli":         "Azure CLI (az login)",
		"managed_identity":  "Managed Identity",
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
	case "service_principal":
		return ic.getAzureServicePrincipal(credentials)
	case "azure_cli":
		return ic.getAzureCLI(credentials)
	case "managed_identity":
		return ic.getAzureManagedIdentity(credentials)
	default:
		return credentials, nil
	}
}

func (ic *InteractiveConfig) getAzureServicePrincipal(credentials map[string]string) (map[string]string, error) {
	fmt.Println("\nService Principal Configuration")
	fmt.Println("You can create a service principal in Azure Portal:")
	fmt.Println("  1. Go to Azure Active Directory")
	fmt.Println("  2. App registrations → New registration")
	fmt.Println("  3. Certificates & secrets → New client secret")
	fmt.Println("")

	// Client ID
	var clientID string
	clientIDPrompt := &survey.Input{
		Message: "Azure Client ID (Application ID):",
		Help:    "The Application (client) ID from your App registration",
	}
	if err := survey.AskOne(clientIDPrompt, &clientID, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["client_id"] = clientID

	// Client Secret
	var clientSecret string
	clientSecretPrompt := &survey.Password{
		Message: "Azure Client Secret:",
		Help:    "The client secret value (not the secret ID)",
	}
	if err := survey.AskOne(clientSecretPrompt, &clientSecret, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["client_secret"] = clientSecret

	// Tenant ID
	var tenantID string
	tenantIDPrompt := &survey.Input{
		Message: "Azure Tenant ID (Directory ID):",
		Help:    "Found in Azure Active Directory → Properties → Directory ID",
	}
	if err := survey.AskOne(tenantIDPrompt, &tenantID, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["tenant_id"] = tenantID

	// Subscription ID
	var subscriptionID string
	subscriptionPrompt := &survey.Input{
		Message: "Azure Subscription ID:",
		Help:    "Your Azure subscription ID for resource management",
	}
	if err := survey.AskOne(subscriptionPrompt, &subscriptionID, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["subscription_id"] = subscriptionID

	return credentials, nil
}

func (ic *InteractiveConfig) getAzureCLI(credentials map[string]string) (map[string]string, error) {
	fmt.Println("Using Azure CLI Authentication")
	fmt.Println("Make sure you have run: az login")
	fmt.Println("")

	// Subscription ID (still needed)
	var subscriptionID string
	subscriptionPrompt := &survey.Input{
		Message: "Azure Subscription ID (optional):",
		Help:    "Leave empty to use your default subscription",
	}
	if err := survey.AskOne(subscriptionPrompt, &subscriptionID); err != nil {
		return nil, err
	}
	if subscriptionID != "" {
		credentials["subscription_id"] = subscriptionID
	}

	return credentials, nil
}

func (ic *InteractiveConfig) getAzureManagedIdentity(credentials map[string]string) (map[string]string, error) {
	fmt.Println("Using Managed Identity Authentication")
	fmt.Println("This method works when running on Azure VMs or App Services")
	fmt.Println("")

	// Optional: Client ID for user-assigned managed identity
	var useUserAssigned bool
	userAssignedPrompt := &survey.Confirm{
		Message: "Are you using a user-assigned managed identity?",
		Default: false,
	}
	if err := survey.AskOne(userAssignedPrompt, &useUserAssigned); err != nil {
		return nil, err
	}

	if useUserAssigned {
		var clientID string
		clientIDPrompt := &survey.Input{
			Message: "User-assigned managed identity Client ID:",
			Help:    "The client ID of your user-assigned managed identity",
		}
		if err := survey.AskOne(clientIDPrompt, &clientID, survey.WithValidator(survey.Required)); err != nil {
			return nil, err
		}
		credentials["client_id"] = clientID
	}

	// Subscription ID
	var subscriptionID string
	subscriptionPrompt := &survey.Input{
		Message: "Azure Subscription ID:",
		Help:    "Your Azure subscription ID for resource management",
	}
	if err := survey.AskOne(subscriptionPrompt, &subscriptionID, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}
	credentials["subscription_id"] = subscriptionID

	return credentials, nil
}

// getAzureRegion gets Azure region selection
func (ic *InteractiveConfig) getAzureRegion() (string, error) {
	azureRegions := []string{
		"eastus",         // East US
		"eastus2",        // East US 2
		"westus",         // West US
		"westus2",        // West US 2
		"westus3",        // West US 3
		"centralus",      // Central US
		"northcentralus", // North Central US
		"southcentralus", // South Central US
		"westcentralus",  // West Central US
		"canadacentral",  // Canada Central
		"canadaeast",     // Canada East
		"brazilsouth",    // Brazil South
		"northeurope",    // North Europe
		"westeurope",     // West Europe
		"uksouth",        // UK South
		"ukwest",         // UK West
		"francecentral",  // France Central
		"francesouth",    // France South
		"germanywest",    // Germany West
		"germanynorth",   // Germany North
		"norwayeast",     // Norway East
		"switzerlandnorth", // Switzerland North
		"eastasia",       // East Asia
		"southeastasia",  // Southeast Asia
		"australiaeast",  // Australia East
		"australiasoutheast", // Australia Southeast
		"australiacentral",   // Australia Central
		"japaneast",      // Japan East
		"japanwest",      // Japan West
		"koreacentral",   // Korea Central
		"koreasouth",     // Korea South
		"southindia",     // South India
		"westindia",      // West India
		"centralindia",   // Central India
		"uaenorth",       // UAE North
		"southafricanorth", // South Africa North
	}

	regionDescriptions := map[string]string{
		"eastus":             "East US (Virginia)",
		"eastus2":            "East US 2 (Virginia)",
		"westus":             "West US (California)",
		"westus2":            "West US 2 (Washington)",
		"westus3":            "West US 3 (Arizona)",
		"centralus":          "Central US (Iowa)",
		"northcentralus":     "North Central US (Illinois)",
		"southcentralus":     "South Central US (Texas)",
		"westcentralus":      "West Central US (Wyoming)",
		"canadacentral":      "Canada Central (Toronto)",
		"canadaeast":         "Canada East (Quebec City)",
		"brazilsouth":        "Brazil South (São Paulo)",
		"northeurope":        "North Europe (Ireland)",
		"westeurope":         "West Europe (Netherlands)",
		"uksouth":            "UK South (London)",
		"ukwest":             "UK West (Cardiff)",
		"francecentral":      "France Central (Paris)",
		"francesouth":        "France South (Marseille)",
		"germanywest":        "Germany West (Frankfurt)",
		"germanynorth":       "Germany North (Berlin)",
		"norwayeast":         "Norway East (Oslo)",
		"switzerlandnorth":   "Switzerland North (Zurich)",
		"eastasia":           "East Asia (Hong Kong)",
		"southeastasia":      "Southeast Asia (Singapore)",
		"australiaeast":      "Australia East (New South Wales)",
		"australiasoutheast": "Australia Southeast (Victoria)",
		"australiacentral":   "Australia Central (Canberra)",
		"japaneast":          "Japan East (Tokyo)",
		"japanwest":          "Japan West (Osaka)",
		"koreacentral":       "Korea Central (Seoul)",
		"koreasouth":         "Korea South (Busan)",
		"southindia":         "South India (Chennai)",
		"westindia":          "West India (Mumbai)",
		"centralindia":       "Central India (Pune)",
		"uaenorth":           "UAE North (Dubai)",
		"southafricanorth":   "South Africa North (Johannesburg)",
	}

	var selectedRegion string
	prompt := &survey.Select{
		Message:  "Select your preferred Azure region:",
		Options:  azureRegions,
		Default:  "eastus",
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