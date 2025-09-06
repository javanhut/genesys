package config

import (
	"fmt"
	"math"
	"strings"
)

// PricingEstimate represents cost estimation for AWS resources
type PricingEstimate struct {
	InstanceType     string  `json:"instance_type"`
	Region           string  `json:"region"`
	HourlyRate       float64 `json:"hourly_rate"`
	MonthlyRate      float64 `json:"monthly_rate"`
	StorageCostGB    float64 `json:"storage_cost_gb"`
	TotalStorageCost float64 `json:"total_storage_cost"`
	TotalMonthlyCost float64 `json:"total_monthly_cost"`
	Currency         string  `json:"currency"`
	LastUpdated      string  `json:"last_updated"`
}

// EC2PricingData contains pricing information for EC2 instances
// Note: These are approximate US East (N. Virginia) prices as of 2024
// Real implementations should fetch from AWS Pricing API
var EC2PricingData = map[string]map[string]float64{
	// Instance type -> Region -> Hourly price in USD
	"t3.small": {
		"us-east-1":      0.0208, // Virginia
		"us-east-2":      0.0208, // Ohio
		"us-west-1":      0.0248, // N. California
		"us-west-2":      0.0208, // Oregon
		"eu-west-1":      0.0230, // Ireland
		"eu-central-1":   0.0240, // Frankfurt
		"ap-southeast-1": 0.0251, // Singapore
		"ap-northeast-1": 0.0251, // Tokyo
	},
	"t3.medium": {
		"us-east-1":      0.0416,
		"us-east-2":      0.0416,
		"us-west-1":      0.0496,
		"us-west-2":      0.0416,
		"eu-west-1":      0.0461,
		"eu-central-1":   0.0480,
		"ap-southeast-1": 0.0502,
		"ap-northeast-1": 0.0502,
	},
	"t3.large": {
		"us-east-1":      0.0832,
		"us-east-2":      0.0832,
		"us-west-1":      0.0992,
		"us-west-2":      0.0832,
		"eu-west-1":      0.0922,
		"eu-central-1":   0.0960,
		"ap-southeast-1": 0.1003,
		"ap-northeast-1": 0.1003,
	},
	"t3.xlarge": {
		"us-east-1":      0.1664,
		"us-east-2":      0.1664,
		"us-west-1":      0.1984,
		"us-west-2":      0.1664,
		"eu-west-1":      0.1843,
		"eu-central-1":   0.1920,
		"ap-southeast-1": 0.2006,
		"ap-northeast-1": 0.2006,
	},
}

// EBS Storage pricing per GB per month
var EBSPricingData = map[string]map[string]float64{
	// Volume type -> Region -> Price per GB per month in USD
	"gp3": {
		"us-east-1":      0.08,
		"us-east-2":      0.08,
		"us-west-1":      0.10,
		"us-west-2":      0.08,
		"eu-west-1":      0.089,
		"eu-central-1":   0.095,
		"ap-southeast-1": 0.12,
		"ap-northeast-1": 0.12,
	},
	"gp2": {
		"us-east-1":      0.10,
		"us-east-2":      0.10,
		"us-west-1":      0.12,
		"us-west-2":      0.10,
		"eu-west-1":      0.11,
		"eu-central-1":   0.119,
		"ap-southeast-1": 0.14,
		"ap-northeast-1": 0.14,
	},
	"io1": {
		"us-east-1":      0.125,
		"us-east-2":      0.125,
		"us-west-1":      0.138,
		"us-west-2":      0.125,
		"eu-west-1":      0.138,
		"eu-central-1":   0.149,
		"ap-southeast-1": 0.15,
		"ap-northeast-1": 0.15,
	},
	"st1": {
		"us-east-1":      0.045,
		"us-east-2":      0.045,
		"us-west-1":      0.054,
		"us-west-2":      0.045,
		"eu-west-1":      0.05,
		"eu-central-1":   0.054,
		"ap-southeast-1": 0.068,
		"ap-northeast-1": 0.068,
	},
}

// EstimateEC2Costs calculates estimated costs for an EC2 configuration
func EstimateEC2Costs(config EC2ComputeResource, region string) (*PricingEstimate, error) {
	estimate := &PricingEstimate{
		InstanceType: config.Type,
		Region:       region,
		Currency:     "USD",
		LastUpdated:  "2024-01 (Approximate)",
	}

	// Get instance hourly rate
	instancePricing, exists := EC2PricingData[config.Type]
	if !exists {
		return nil, fmt.Errorf("pricing data not available for instance type: %s", config.Type)
	}

	hourlyRate, exists := instancePricing[region]
	if !exists {
		// Use us-east-1 as fallback with 10% markup for unknown regions
		if fallback, ok := instancePricing["us-east-1"]; ok {
			hourlyRate = fallback * 1.1
		} else {
			return nil, fmt.Errorf("pricing data not available for region: %s", region)
		}
	}

	estimate.HourlyRate = hourlyRate
	estimate.MonthlyRate = hourlyRate * 24 * 30 // Approximate month

	// Calculate storage costs if configured
	if config.Storage != nil {
		volumeType := strings.ToLower(config.Storage.VolumeType)
		if volumeType == "" {
			volumeType = "gp3" // Default
		}

		storagePricing, exists := EBSPricingData[volumeType]
		if exists {
			storageRate, exists := storagePricing[region]
			if !exists {
				// Use us-east-1 as fallback
				if fallback, ok := storagePricing["us-east-1"]; ok {
					storageRate = fallback * 1.1
				} else {
					storageRate = 0.08 // Default gp3 rate
				}
			}
			
			estimate.StorageCostGB = storageRate
			estimate.TotalStorageCost = storageRate * float64(config.Storage.Size)
		}
	}

	estimate.TotalMonthlyCost = estimate.MonthlyRate + estimate.TotalStorageCost

	return estimate, nil
}

// FormatCostEstimate returns a human-readable cost breakdown
func (pe *PricingEstimate) FormatCostEstimate() string {
	var result strings.Builder
	
	result.WriteString(fmt.Sprintf("ESTIMATED MONTHLY COSTS:\n"))
	result.WriteString(fmt.Sprintf("  Instance (%s): $%.2f/month ($%.4f/hour)\n", 
		pe.InstanceType, pe.MonthlyRate, pe.HourlyRate))
	
	if pe.TotalStorageCost > 0 {
		result.WriteString(fmt.Sprintf("  EBS Storage:       $%.2f/month ($%.3f/GB/month)\n", 
			pe.TotalStorageCost, pe.StorageCostGB))
	}
	
	result.WriteString(fmt.Sprintf("  ─────────────────────────────────\n"))
	result.WriteString(fmt.Sprintf("  TOTAL:            $%.2f/month\n", pe.TotalMonthlyCost))
	result.WriteString(fmt.Sprintf("  Region: %s | Currency: %s\n", pe.Region, pe.Currency))
	
	// Add cost warnings
	if pe.TotalMonthlyCost > 100 {
		result.WriteString(fmt.Sprintf("\nHIGH COST WARNING: >$100/month\n"))
	} else if pe.TotalMonthlyCost > 50 {
		result.WriteString(fmt.Sprintf("\nMODERATE COST: >$50/month\n"))
	} else if pe.TotalMonthlyCost < 10 {
		result.WriteString(fmt.Sprintf("\nLOW COST: <$10/month\n"))
	}
	
	result.WriteString(fmt.Sprintf("\nCost Breakdown:\n"))
	instancePercent := (pe.MonthlyRate / pe.TotalMonthlyCost) * 100
	result.WriteString(fmt.Sprintf("  Instance: %.1f%% | ", instancePercent))
	
	if pe.TotalStorageCost > 0 {
		storagePercent := (pe.TotalStorageCost / pe.TotalMonthlyCost) * 100
		result.WriteString(fmt.Sprintf("Storage: %.1f%%", storagePercent))
	} else {
		result.WriteString(fmt.Sprintf("Storage: 0%%"))
	}
	
	result.WriteString(fmt.Sprintf("\n\nCOST OPTIMIZATION TIPS:\n"))
	
	// Instance type suggestions
	if strings.Contains(pe.InstanceType, "xlarge") {
		result.WriteString(fmt.Sprintf("  • Consider smaller instance types (t3.large or t3.medium)\n"))
	}
	
	// Storage suggestions
	if pe.StorageCostGB > 0.10 {
		result.WriteString(fmt.Sprintf("  • Consider gp3 volumes for better price/performance\n"))
	}
	
	// General suggestions
	result.WriteString(fmt.Sprintf("  • Use Spot Instances for 70%% savings (non-critical workloads)\n"))
	result.WriteString(fmt.Sprintf("  • Set up billing alerts in AWS Console\n"))
	result.WriteString(fmt.Sprintf("  • Consider Reserved Instances for long-term use (up to 72%% off)\n"))
	
	result.WriteString(fmt.Sprintf("\nNote: Prices are estimates. Actual costs may vary.\n"))
	result.WriteString(fmt.Sprintf("Data transfer, snapshots, and other services cost extra.\n"))
	
	return result.String()
}

// GetCostWarningLevel returns a warning level based on monthly cost
func (pe *PricingEstimate) GetCostWarningLevel() string {
	if pe.TotalMonthlyCost > 100 {
		return "HIGH"
	} else if pe.TotalMonthlyCost > 50 {
		return "MODERATE"
	} else if pe.TotalMonthlyCost < 10 {
		return "LOW"
	}
	return "NORMAL"
}

// EstimateS3Costs calculates estimated costs for S3 storage
func EstimateS3Costs(region string, sizeMB int) (float64, error) {
	// S3 Standard pricing per GB per month (approximate)
	s3Pricing := map[string]float64{
		"us-east-1":      0.023, // First 50 TB
		"us-east-2":      0.023,
		"us-west-1":      0.026,
		"us-west-2":      0.023,
		"eu-west-1":      0.025,
		"eu-central-1":   0.025,
		"ap-southeast-1": 0.025,
		"ap-northeast-1": 0.025,
	}
	
	pricePerGB, exists := s3Pricing[region]
	if !exists {
		pricePerGB = 0.023 // Default to us-east-1 pricing
	}
	
	sizeGB := float64(sizeMB) / 1024
	if sizeGB < 1 {
		sizeGB = 1 // Minimum 1GB for estimation
	}
	
	monthlyCost := sizeGB * pricePerGB
	return math.Max(monthlyCost, 0.01), nil // Minimum $0.01
}