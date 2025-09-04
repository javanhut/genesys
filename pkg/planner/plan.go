package planner

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Plan represents an execution plan
type Plan struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Steps       []PlanStep   `json:"steps"`
	Permissions IAMForecast  `json:"permissions"`
	Cost        CostEstimate `json:"cost"`
	Duration    string       `json:"duration"`
	CreatedAt   time.Time    `json:"created_at"`
}

// PlanStep represents a single step in the plan
type PlanStep struct {
	ID          string   `json:"id"`
	Action      string   `json:"action"`
	Resource    string   `json:"resource"`
	Description string   `json:"description"`
	Reason      string   `json:"reason,omitempty"`
	IAMActions  []string `json:"iam_actions,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Optional    bool     `json:"optional,omitempty"`
}

// IAMForecast represents required IAM permissions
type IAMForecast struct {
	Actions   []string `json:"actions"`
	Resources []string `json:"resources"`
	Policy    string   `json:"policy,omitempty"`
}

// CostEstimate represents cost information
type CostEstimate struct {
	Monthly     float64            `json:"monthly"`
	Hourly      float64            `json:"hourly"`
	Breakdown   map[string]float64 `json:"breakdown,omitempty"`
	Currency    string             `json:"currency"`
	Warning     string             `json:"warning,omitempty"`
	Confidence  string             `json:"confidence"` // low|medium|high
}

// ToHumanReadable converts the plan to a human-readable format
func (p *Plan) ToHumanReadable() string {
	var output strings.Builder

	// Header
	output.WriteString(fmt.Sprintf("Plan: %s\n", p.Title))
	output.WriteString(strings.Repeat("=", len(p.Title)+8) + "\n\n")

	if p.Description != "" {
		output.WriteString(fmt.Sprintf("%s\n\n", p.Description))
	}

	// Steps
	output.WriteString("What will happen:\n")
	for i, step := range p.Steps {
		icon := "▶"
		if step.Optional {
			icon = "◦"
		}

		output.WriteString(fmt.Sprintf("%s %d. %s\n", icon, i+1, step.Description))
		if step.Reason != "" {
			output.WriteString(fmt.Sprintf("     → %s\n", step.Reason))
		}
	}

	// Permissions
	if len(p.Permissions.Actions) > 0 {
		output.WriteString("\nPermissions needed:\n")
		for _, action := range p.Permissions.Actions {
			output.WriteString(fmt.Sprintf("- %s\n", action))
		}
	}

	// Cost
	if p.Cost.Monthly > 0 {
		output.WriteString("\nCost estimate:\n")
		output.WriteString(fmt.Sprintf("- Monthly: $%.2f %s\n", p.Cost.Monthly, p.Cost.Currency))
		if p.Cost.Hourly > 0 {
			output.WriteString(fmt.Sprintf("- Hourly: $%.4f %s\n", p.Cost.Hourly, p.Cost.Currency))
		}
		output.WriteString(fmt.Sprintf("- Confidence: %s\n", p.Cost.Confidence))
		if p.Cost.Warning != "" {
			output.WriteString(fmt.Sprintf("Warning: %s\n", p.Cost.Warning))
		}
	}

	// Duration
	if p.Duration != "" {
		output.WriteString(fmt.Sprintf("\nTime to complete: %s\n", p.Duration))
	}

	// Footer
	output.WriteString(fmt.Sprintf("\nPlan ID: %s\n", p.ID))
	output.WriteString(fmt.Sprintf("Created: %s\n", p.CreatedAt.Format("2006-01-02 15:04:05")))

	return output.String()
}

// ToJSON converts the plan to JSON format
func (p *Plan) ToJSON() string {
	data, _ := json.MarshalIndent(p, "", "  ")
	return string(data)
}

// NewBucketPlan creates a plan for bucket deployment
func NewBucketPlan(name string, params map[string]string) *Plan {
	planID := fmt.Sprintf("bucket-%d", time.Now().Unix())
	
	plan := &Plan{
		ID:          planID,
		Title:       fmt.Sprintf("Deploy S3 Bucket '%s'", name),
		Description: "Create a secure, versioned storage bucket following best practices",
		Steps:       []PlanStep{},
		CreatedAt:   time.Now(),
		Duration:    "30 seconds",
	}

	// Add steps based on parameters
	steps := []PlanStep{
		{
			ID:          "create-bucket",
			Action:      "create",
			Resource:    "s3-bucket",
			Description: fmt.Sprintf("Create S3 bucket '%s'", name),
			Reason:      "Store your application data securely",
			IAMActions:  []string{"s3:CreateBucket"},
		},
	}

	if params["versioning"] == "true" {
		steps = append(steps, PlanStep{
			ID:          "enable-versioning",
			Action:      "configure",
			Resource:    "s3-bucket-versioning",
			Description: "Enable versioning on bucket",
			Reason:      "Protect against accidental deletion or modification",
			IAMActions:  []string{"s3:PutBucketVersioning"},
			DependsOn:   []string{"create-bucket"},
		})
	}

	if params["encryption"] == "true" {
		steps = append(steps, PlanStep{
			ID:          "enable-encryption",
			Action:      "configure",
			Resource:    "s3-bucket-encryption",
			Description: "Enable encryption with AWS managed keys",
			Reason:      "Secure data at rest",
			IAMActions:  []string{"s3:PutBucketEncryption"},
			DependsOn:   []string{"create-bucket"},
		})
	}

	if params["public"] != "true" {
		steps = append(steps, PlanStep{
			ID:          "block-public-access",
			Action:      "configure",
			Resource:    "s3-bucket-public-access",
			Description: "Block all public access",
			Reason:      "Prevent data exposure",
			IAMActions:  []string{"s3:PutBucketPublicAccessBlock"},
			DependsOn:   []string{"create-bucket"},
		})
	}

	plan.Steps = steps

	// Set permissions
	var actions []string
	for _, step := range steps {
		actions = append(actions, step.IAMActions...)
	}
	plan.Permissions = IAMForecast{
		Actions:   removeDuplicates(actions),
		Resources: []string{fmt.Sprintf("arn:aws:s3:::%s", name), fmt.Sprintf("arn:aws:s3:::%s/*", name)},
	}

	// Set cost estimate
	plan.Cost = CostEstimate{
		Monthly:    5.00,  // Rough estimate
		Hourly:     0.007,
		Currency:   "USD",
		Confidence: "medium",
		Breakdown: map[string]float64{
			"Storage (first 50TB)": 2.30,
			"Requests":             0.50,
			"Data transfer":        2.20,
		},
	}

	return plan
}

// NewNetworkPlan creates a plan for network deployment
func NewNetworkPlan(name string, params map[string]string) *Plan {
	planID := fmt.Sprintf("network-%d", time.Now().Unix())
	cidr := params["cidr"]
	if cidr == "" {
		cidr = "10.0.0.0/16"
	}
	
	plan := &Plan{
		ID:          planID,
		Title:       fmt.Sprintf("Deploy Network '%s'", name),
		Description: fmt.Sprintf("Create VPC with CIDR %s including public and private subnets", cidr),
		CreatedAt:   time.Now(),
		Duration:    "2 minutes",
	}

	steps := []PlanStep{
		{
			ID:          "create-vpc",
			Action:      "create",
			Resource:    "vpc",
			Description: fmt.Sprintf("Create VPC with CIDR %s", cidr),
			Reason:      "Isolated network environment for your resources",
			IAMActions:  []string{"ec2:CreateVpc"},
		},
		{
			ID:          "create-igw",
			Action:      "create",
			Resource:    "internet-gateway",
			Description: "Create Internet Gateway",
			Reason:      "Enable internet access for public subnets",
			IAMActions:  []string{"ec2:CreateInternetGateway", "ec2:AttachInternetGateway"},
			DependsOn:   []string{"create-vpc"},
		},
		{
			ID:          "create-public-subnet",
			Action:      "create",
			Resource:    "subnet",
			Description: "Create public subnet (10.0.1.0/24)",
			Reason:      "Host resources that need direct internet access",
			IAMActions:  []string{"ec2:CreateSubnet"},
			DependsOn:   []string{"create-vpc"},
		},
		{
			ID:          "create-private-subnet",
			Action:      "create",
			Resource:    "subnet",
			Description: "Create private subnet (10.0.2.0/24)",
			Reason:      "Host resources that don't need direct internet access",
			IAMActions:  []string{"ec2:CreateSubnet"},
			DependsOn:   []string{"create-vpc"},
		},
		{
			ID:          "create-route-tables",
			Action:      "create",
			Resource:    "route-table",
			Description: "Create and configure route tables",
			Reason:      "Control traffic routing between subnets",
			IAMActions:  []string{"ec2:CreateRouteTable", "ec2:CreateRoute", "ec2:AssociateRouteTable"},
			DependsOn:   []string{"create-public-subnet", "create-private-subnet", "create-igw"},
		},
	}

	plan.Steps = steps

	// Set permissions
	var actions []string
	for _, step := range steps {
		actions = append(actions, step.IAMActions...)
	}
	plan.Permissions = IAMForecast{
		Actions: removeDuplicates(actions),
	}

	// Set cost estimate (VPC is free, just data transfer costs)
	plan.Cost = CostEstimate{
		Monthly:    0.00,
		Hourly:     0.00,
		Currency:   "USD",
		Confidence: "high",
		Breakdown: map[string]float64{
			"VPC":              0.00,
			"Subnets":          0.00,
			"Internet Gateway": 0.00,
		},
	}

	return plan
}

// NewFunctionPlan creates a plan for function deployment
func NewFunctionPlan(name string, params map[string]string) *Plan {
	planID := fmt.Sprintf("function-%d", time.Now().Unix())
	runtime := params["runtime"]
	if runtime == "" {
		runtime = "python3.11"
	}
	
	plan := &Plan{
		ID:          planID,
		Title:       fmt.Sprintf("Deploy Function '%s'", name),
		Description: fmt.Sprintf("Create serverless function with %s runtime", runtime),
		CreatedAt:   time.Now(),
		Duration:    "1 minute",
	}

	steps := []PlanStep{
		{
			ID:          "create-execution-role",
			Action:      "create",
			Resource:    "iam-role",
			Description: "Create Lambda execution role",
			Reason:      "Allow function to write logs and access AWS services",
			IAMActions:  []string{"iam:CreateRole", "iam:AttachRolePolicy"},
		},
		{
			ID:          "create-function",
			Action:      "create",
			Resource:    "lambda-function",
			Description: fmt.Sprintf("Create Lambda function with %s runtime", runtime),
			Reason:      "Deploy your serverless code",
			IAMActions:  []string{"lambda:CreateFunction"},
			DependsOn:   []string{"create-execution-role"},
		},
		{
			ID:          "create-log-group",
			Action:      "create",
			Resource:    "cloudwatch-log-group",
			Description: "Create CloudWatch log group",
			Reason:      "Store function execution logs",
			IAMActions:  []string{"logs:CreateLogGroup"},
			DependsOn:   []string{"create-function"},
		},
	}

	if params["trigger"] == "http" || params["url"] == "true" {
		steps = append(steps, PlanStep{
			ID:          "create-function-url",
			Action:      "create",
			Resource:    "lambda-function-url",
			Description: "Create Function URL for HTTP access",
			Reason:      "Enable direct HTTP invocation",
			IAMActions:  []string{"lambda:CreateFunctionUrlConfig"},
			DependsOn:   []string{"create-function"},
		})
	}

	plan.Steps = steps

	// Set permissions
	var actions []string
	for _, step := range steps {
		actions = append(actions, step.IAMActions...)
	}
	plan.Permissions = IAMForecast{
		Actions: removeDuplicates(actions),
	}

	// Set cost estimate
	memory := 256
	if params["memory"] != "" {
		fmt.Sscanf(params["memory"], "%d", &memory)
	}
	
	monthlyCost := float64(memory) * 0.0000166667 * 100000 // Rough estimate for 100k invocations
	
	plan.Cost = CostEstimate{
		Monthly:    monthlyCost,
		Currency:   "USD",
		Confidence: "medium",
		Breakdown: map[string]float64{
			"Compute time": monthlyCost * 0.8,
			"Requests":     monthlyCost * 0.2,
		},
	}

	return plan
}

// removeDuplicates removes duplicate strings from slice
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}