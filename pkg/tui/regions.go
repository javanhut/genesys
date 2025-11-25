package tui

// AWSRegion represents an AWS region with metadata
type AWSRegion struct {
	Code        string
	Name        string
	Description string
}

// AWSRegions contains all available AWS regions for S3
var AWSRegions = []AWSRegion{
	{Code: "us-east-1", Name: "N. Virginia", Description: "US East (N. Virginia)"},
	{Code: "us-east-2", Name: "Ohio", Description: "US East (Ohio)"},
	{Code: "us-west-1", Name: "N. California", Description: "US West (N. California)"},
	{Code: "us-west-2", Name: "Oregon", Description: "US West (Oregon)"},
	{Code: "af-south-1", Name: "Cape Town", Description: "Africa (Cape Town)"},
	{Code: "ap-east-1", Name: "Hong Kong", Description: "Asia Pacific (Hong Kong)"},
	{Code: "ap-south-1", Name: "Mumbai", Description: "Asia Pacific (Mumbai)"},
	{Code: "ap-south-2", Name: "Hyderabad", Description: "Asia Pacific (Hyderabad)"},
	{Code: "ap-northeast-1", Name: "Tokyo", Description: "Asia Pacific (Tokyo)"},
	{Code: "ap-northeast-2", Name: "Seoul", Description: "Asia Pacific (Seoul)"},
	{Code: "ap-northeast-3", Name: "Osaka", Description: "Asia Pacific (Osaka)"},
	{Code: "ap-southeast-1", Name: "Singapore", Description: "Asia Pacific (Singapore)"},
	{Code: "ap-southeast-2", Name: "Sydney", Description: "Asia Pacific (Sydney)"},
	{Code: "ap-southeast-3", Name: "Jakarta", Description: "Asia Pacific (Jakarta)"},
	{Code: "ap-southeast-4", Name: "Melbourne", Description: "Asia Pacific (Melbourne)"},
	{Code: "ca-central-1", Name: "Canada", Description: "Canada (Central)"},
	{Code: "eu-central-1", Name: "Frankfurt", Description: "Europe (Frankfurt)"},
	{Code: "eu-central-2", Name: "Zurich", Description: "Europe (Zurich)"},
	{Code: "eu-west-1", Name: "Ireland", Description: "Europe (Ireland)"},
	{Code: "eu-west-2", Name: "London", Description: "Europe (London)"},
	{Code: "eu-west-3", Name: "Paris", Description: "Europe (Paris)"},
	{Code: "eu-north-1", Name: "Stockholm", Description: "Europe (Stockholm)"},
	{Code: "eu-south-1", Name: "Milan", Description: "Europe (Milan)"},
	{Code: "eu-south-2", Name: "Spain", Description: "Europe (Spain)"},
	{Code: "il-central-1", Name: "Tel Aviv", Description: "Israel (Tel Aviv)"},
	{Code: "me-south-1", Name: "Bahrain", Description: "Middle East (Bahrain)"},
	{Code: "me-central-1", Name: "UAE", Description: "Middle East (UAE)"},
	{Code: "sa-east-1", Name: "Sao Paulo", Description: "South America (Sao Paulo)"},
}

// GetRegionCodes returns a list of all region codes
func GetRegionCodes() []string {
	codes := make([]string, len(AWSRegions))
	for i, r := range AWSRegions {
		codes[i] = r.Code
	}
	return codes
}

// GetRegionByCode returns the region information for a given code
func GetRegionByCode(code string) *AWSRegion {
	for _, r := range AWSRegions {
		if r.Code == code {
			return &r
		}
	}
	return nil
}

// GetRegionDisplay returns a formatted display string for a region
func GetRegionDisplay(code string) string {
	r := GetRegionByCode(code)
	if r != nil {
		return r.Code + " (" + r.Name + ")"
	}
	return code
}
