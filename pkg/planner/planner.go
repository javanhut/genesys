package planner

import (
	"context"
	"fmt"
	"time"

	"github.com/javanhut/genesys/pkg/intent"
	"github.com/javanhut/genesys/pkg/provider"
)

// Planner generates execution plans
type Planner struct {
	provider provider.Provider
}

// New creates a new planner instance
func New(p provider.Provider) *Planner {
	return &Planner{
		provider: p,
	}
}

// PlanFromIntent generates a plan from a parsed intent
func (p *Planner) PlanFromIntent(ctx context.Context, i *intent.Intent) (*Plan, error) {
	switch i.Type {
	case intent.IntentBucket:
		return p.planBucket(ctx, i)
	case intent.IntentNetwork:
		return p.planNetwork(ctx, i)
	case intent.IntentFunction:
		return p.planFunction(ctx, i)
	case intent.IntentStaticSite:
		return p.planStaticSite(ctx, i)
	case intent.IntentDatabase:
		return p.planDatabase(ctx, i)
	case intent.IntentAPI:
		return p.planAPI(ctx, i)
	case intent.IntentWebapp:
		return p.planWebapp(ctx, i)
	default:
		return nil, fmt.Errorf("unsupported intent type: %s", i.Type)
	}
}

// planBucket creates a plan for bucket deployment
func (p *Planner) planBucket(ctx context.Context, i *intent.Intent) (*Plan, error) {
	name := i.Name
	if name == "" {
		name = fmt.Sprintf("genesys-bucket-%d", time.Now().Unix())
	}

	// Check if bucket already exists
	existing, err := p.provider.Storage().GetBucket(ctx, name)
	if err == nil && existing != nil {
		// Bucket exists - create adoption plan
		return p.createAdoptionPlan(ctx, "bucket", name, existing)
	}

	return NewBucketPlan(name, i.Parameters), nil
}

// planNetwork creates a plan for network deployment
func (p *Planner) planNetwork(ctx context.Context, i *intent.Intent) (*Plan, error) {
	name := i.Name
	if name == "" {
		name = fmt.Sprintf("genesys-vpc-%d", time.Now().Unix())
	}

	return NewNetworkPlan(name, i.Parameters), nil
}

// planFunction creates a plan for function deployment
func (p *Planner) planFunction(ctx context.Context, i *intent.Intent) (*Plan, error) {
	name := i.Name
	if name == "" {
		name = fmt.Sprintf("genesys-function-%d", time.Now().Unix())
	}

	return NewFunctionPlan(name, i.Parameters), nil
}

// planStaticSite creates a plan for static site deployment
func (p *Planner) planStaticSite(ctx context.Context, i *intent.Intent) (*Plan, error) {
	domain := i.Parameters["domain"]
	if domain == "" {
		domain = "example.com"
	}

	plan := &Plan{
		ID:          fmt.Sprintf("static-site-%d", time.Now().Unix()),
		Title:       "Deploy Static Website",
		Description: fmt.Sprintf("Create a static website hosted on %s with CDN", domain),
		CreatedAt:   time.Now(),
		Duration:    "3-5 minutes",
	}

	steps := []PlanStep{
		{
			ID:          "create-bucket",
			Action:      "create",
			Resource:    "s3-bucket",
			Description: "Create S3 bucket for website files",
			Reason:      "Store your website content",
			IAMActions:  []string{"s3:CreateBucket", "s3:PutBucketWebsite"},
		},
		{
			ID:          "configure-hosting",
			Action:      "configure",
			Resource:    "s3-bucket-website",
			Description: "Configure bucket for static website hosting",
			Reason:      "Enable web access to your content",
			IAMActions:  []string{"s3:PutBucketWebsite", "s3:PutBucketPolicy"},
			DependsOn:   []string{"create-bucket"},
		},
	}

	if i.Parameters["cdn"] == "true" {
		steps = append(steps, PlanStep{
			ID:          "create-cloudfront",
			Action:      "create",
			Resource:    "cloudfront-distribution",
			Description: "Set up CloudFront CDN for fast global delivery",
			Reason:      "Improve performance worldwide",
			IAMActions:  []string{"cloudfront:CreateDistribution"},
			DependsOn:   []string{"configure-hosting"},
		})
	}

	if i.Parameters["https"] == "true" {
		steps = append(steps, PlanStep{
			ID:          "request-certificate",
			Action:      "create",
			Resource:    "acm-certificate",
			Description: "Request SSL certificate for HTTPS",
			Reason:      "Secure your website with encryption",
			IAMActions:  []string{"acm:RequestCertificate"},
			Optional:    true,
		})
	}

	if domain != "" && domain != "example.com" {
		steps = append(steps, PlanStep{
			ID:          "configure-dns",
			Action:      "configure",
			Resource:    "route53-records",
			Description: fmt.Sprintf("Configure DNS for %s", domain),
			Reason:      "Point your domain to the website",
			IAMActions:  []string{"route53:ChangeResourceRecordSets"},
			Optional:    true,
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
	plan.Cost = CostEstimate{
		Monthly:    15.00,
		Currency:   "USD",
		Confidence: "medium",
		Breakdown: map[string]float64{
			"S3 hosting":         2.00,
			"CloudFront":         8.00,
			"Route53":            0.50,
			"Data transfer":      4.50,
		},
	}

	return plan, nil
}

// planDatabase creates a plan for database deployment
func (p *Planner) planDatabase(ctx context.Context, i *intent.Intent) (*Plan, error) {
	name := i.Name
	if name == "" {
		name = fmt.Sprintf("genesys-db-%d", time.Now().Unix())
	}

	engine := i.Parameters["engine"]
	if engine == "" {
		engine = "postgres"
	}

	size := i.Parameters["size"]
	if size == "" {
		size = "small"
	}

	plan := &Plan{
		ID:          fmt.Sprintf("database-%d", time.Now().Unix()),
		Title:       fmt.Sprintf("Deploy %s Database '%s'", engine, name),
		Description: fmt.Sprintf("Create managed %s database with automated backups", engine),
		CreatedAt:   time.Now(),
		Duration:    "10-15 minutes",
	}

	steps := []PlanStep{
		{
			ID:          "create-subnet-group",
			Action:      "create",
			Resource:    "db-subnet-group",
			Description: "Create database subnet group",
			Reason:      "Define network placement for database",
			IAMActions:  []string{"rds:CreateDBSubnetGroup"},
		},
		{
			ID:          "create-security-group",
			Action:      "create",
			Resource:    "security-group",
			Description: "Create database security group",
			Reason:      "Control network access to database",
			IAMActions:  []string{"ec2:CreateSecurityGroup", "ec2:AuthorizeSecurityGroupIngress"},
		},
		{
			ID:          "create-database",
			Action:      "create",
			Resource:    "rds-instance",
			Description: fmt.Sprintf("Create %s database (%s)", engine, size),
			Reason:      "Deploy your managed database",
			IAMActions:  []string{"rds:CreateDBInstance"},
			DependsOn:   []string{"create-subnet-group", "create-security-group"},
		},
	}

	if i.Parameters["backup"] == "true" {
		steps = append(steps, PlanStep{
			ID:          "configure-backups",
			Action:      "configure",
			Resource:    "rds-backup",
			Description: "Configure automated backups (7 day retention)",
			Reason:      "Protect your data with regular backups",
			IAMActions:  []string{"rds:ModifyDBInstance"},
			DependsOn:   []string{"create-database"},
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

	// Set cost estimate based on size
	var monthlyCost float64
	switch size {
	case "small":
		monthlyCost = 25.00
	case "medium":
		monthlyCost = 75.00
	case "large":
		monthlyCost = 150.00
	default:
		monthlyCost = 25.00
	}

	plan.Cost = CostEstimate{
		Monthly:    monthlyCost,
		Currency:   "USD",
		Confidence: "high",
		Breakdown: map[string]float64{
			"Database instance": monthlyCost * 0.8,
			"Storage":           monthlyCost * 0.15,
			"Backups":           monthlyCost * 0.05,
		},
	}

	return plan, nil
}

// planAPI creates a plan for API deployment
func (p *Planner) planAPI(ctx context.Context, i *intent.Intent) (*Plan, error) {
	name := i.Name
	if name == "" {
		name = fmt.Sprintf("genesys-api-%d", time.Now().Unix())
	}

	plan := &Plan{
		ID:          fmt.Sprintf("api-%d", time.Now().Unix()),
		Title:       fmt.Sprintf("Deploy API '%s'", name),
		Description: "Create REST API with Lambda backend and API Gateway",
		CreatedAt:   time.Now(),
		Duration:    "2-3 minutes",
	}

	steps := []PlanStep{
		{
			ID:          "create-lambda",
			Action:      "create",
			Resource:    "lambda-function",
			Description: "Create Lambda function for API logic",
			Reason:      "Handle API requests serverlessly",
			IAMActions:  []string{"lambda:CreateFunction", "iam:CreateRole"},
		},
		{
			ID:          "create-api-gateway",
			Action:      "create",
			Resource:    "api-gateway",
			Description: "Create API Gateway REST API",
			Reason:      "Expose Lambda function as HTTP API",
			IAMActions:  []string{"apigateway:CreateRestApi", "apigateway:CreateResource"},
			DependsOn:   []string{"create-lambda"},
		},
		{
			ID:          "deploy-api",
			Action:      "deploy",
			Resource:    "api-gateway-deployment",
			Description: "Deploy API to production stage",
			Reason:      "Make API publicly accessible",
			IAMActions:  []string{"apigateway:CreateDeployment"},
			DependsOn:   []string{"create-api-gateway"},
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

	plan.Cost = CostEstimate{
		Monthly:    10.00,
		Currency:   "USD",
		Confidence: "medium",
		Breakdown: map[string]float64{
			"Lambda requests": 5.00,
			"API Gateway":     4.00,
			"CloudWatch":      1.00,
		},
	}

	return plan, nil
}

// planWebapp creates a plan for web application deployment
func (p *Planner) planWebapp(ctx context.Context, i *intent.Intent) (*Plan, error) {
	name := i.Name
	if name == "" {
		name = fmt.Sprintf("genesys-webapp-%d", time.Now().Unix())
	}

	instanceType := i.Parameters["type"]
	if instanceType == "" {
		instanceType = "medium"
	}

	plan := &Plan{
		ID:          fmt.Sprintf("webapp-%d", time.Now().Unix()),
		Title:       fmt.Sprintf("Deploy Web Application '%s'", name),
		Description: fmt.Sprintf("Create scalable web application with %s instances", instanceType),
		CreatedAt:   time.Now(),
		Duration:    "5-8 minutes",
	}

	steps := []PlanStep{
		{
			ID:          "create-security-group",
			Action:      "create",
			Resource:    "security-group",
			Description: "Create security group for web servers",
			Reason:      "Control access to your application",
			IAMActions:  []string{"ec2:CreateSecurityGroup", "ec2:AuthorizeSecurityGroupIngress"},
		},
		{
			ID:          "create-launch-template",
			Action:      "create",
			Resource:    "launch-template",
			Description: fmt.Sprintf("Create launch template with %s instances", instanceType),
			Reason:      "Define instance configuration",
			IAMActions:  []string{"ec2:CreateLaunchTemplate"},
			DependsOn:   []string{"create-security-group"},
		},
	}

	if i.Parameters["lb"] == "true" {
		steps = append(steps, PlanStep{
			ID:          "create-load-balancer",
			Action:      "create",
			Resource:    "application-load-balancer",
			Description: "Create Application Load Balancer",
			Reason:      "Distribute traffic across instances",
			IAMActions:  []string{"elasticloadbalancing:CreateLoadBalancer", "elasticloadbalancing:CreateTargetGroup"},
		})
	}

	if i.Parameters["scaling"] == "auto" {
		steps = append(steps, PlanStep{
			ID:          "create-autoscaling",
			Action:      "create",
			Resource:    "autoscaling-group",
			Description: "Create Auto Scaling Group (min: 1, max: 3)",
			Reason:      "Automatically scale based on demand",
			IAMActions:  []string{"autoscaling:CreateAutoScalingGroup"},
			DependsOn:   []string{"create-launch-template"},
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

	// Set cost estimate based on instance type
	var monthlyCost float64
	switch instanceType {
	case "small":
		monthlyCost = 50.00
	case "medium":
		monthlyCost = 100.00
	case "large":
		monthlyCost = 200.00
	default:
		monthlyCost = 100.00
	}

	plan.Cost = CostEstimate{
		Monthly:    monthlyCost,
		Currency:   "USD",
		Confidence: "high",
		Breakdown: map[string]float64{
			"EC2 instances":     monthlyCost * 0.7,
			"Load balancer":     monthlyCost * 0.2,
			"Data transfer":     monthlyCost * 0.1,
		},
	}

	return plan, nil
}

// createAdoptionPlan creates a plan for adopting existing resources
func (p *Planner) createAdoptionPlan(ctx context.Context, resourceType, name string, resource interface{}) (*Plan, error) {
	plan := &Plan{
		ID:          fmt.Sprintf("adopt-%s-%d", resourceType, time.Now().Unix()),
		Title:       fmt.Sprintf("Adopt Existing %s '%s'", resourceType, name),
		Description: fmt.Sprintf("Import and manage existing %s with Genesys", resourceType),
		CreatedAt:   time.Now(),
		Duration:    "30 seconds",
	}

	steps := []PlanStep{
		{
			ID:          "analyze-resource",
			Action:      "analyze",
			Resource:    resourceType,
			Description: fmt.Sprintf("Analyze existing %s configuration", resourceType),
			Reason:      "Understand current resource state",
			IAMActions:  []string{fmt.Sprintf("%s:Describe*", getServicePrefix(resourceType))},
		},
		{
			ID:          "import-state",
			Action:      "import",
			Resource:    "state",
			Description: "Import resource into Genesys state",
			Reason:      "Track resource in Genesys management",
			DependsOn:   []string{"analyze-resource"},
		},
		{
			ID:          "apply-best-practices",
			Action:      "configure",
			Resource:    resourceType,
			Description: "Apply Genesys best practices (if needed)",
			Reason:      "Ensure resource follows security guidelines",
			Optional:    true,
			DependsOn:   []string{"import-state"},
		},
	}

	plan.Steps = steps
	plan.Permissions = IAMForecast{
		Actions: []string{fmt.Sprintf("%s:Describe*", getServicePrefix(resourceType))},
	}
	plan.Cost = CostEstimate{
		Monthly:    0.00, // No additional cost for adoption
		Currency:   "USD",
		Confidence: "high",
	}

	return plan, nil
}

// getServicePrefix returns the AWS service prefix for IAM actions
func getServicePrefix(resourceType string) string {
	switch resourceType {
	case "bucket":
		return "s3"
	case "network", "vpc":
		return "ec2"
	case "function":
		return "lambda"
	case "database":
		return "rds"
	default:
		return "ec2"
	}
}