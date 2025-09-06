# Genesys MVP Architecture

## Core Philosophy
**Simplicity-first IaaS that focuses on outcomes, not resources**

## Primary Design Principles

### 1. Single Command Experience
```bash
# Everything through one verb
genesys [outcome] [options]

# Preview by default (safe)
genesys static-site            # Shows what would happen
genesys static-site --apply     # Actually does it
```

### 2. Outcome-Based Resources
Users ask for what they want to achieve, not AWS resources:
- `static-site` → S3 + CloudFront + Route53
- `db` → RDS with proper VPC, security groups, backups
- `function` → Lambda + API Gateway + IAM roles
- `api` → API Gateway + Lambda + DynamoDB
- `webapp` → EC2/ECS + ALB + Auto-scaling

### 3. Discovery-First Approach
```bash
# Always check what exists first
genesys static-site
> Found existing S3 bucket 'my-site-bucket'
> Would you like to:
  1. Adopt and manage existing resources
  2. Create new resources
  3. Modify existing to match requirements
```

### 4. Human-Readable Plans
```
Plan for: static-site deployment

What will happen:
1. Create S3 bucket for hosting your website files
2. Set up CloudFront CDN for fast global delivery  
3. Configure DNS with your domain (if provided)
4. Enable HTTPS with automatic certificate

Permissions needed:
- S3: Create and manage buckets
- CloudFront: Create distributions
- ACM: Request certificates
- Route53: Manage DNS records

Estimated cost: ~$5/month
Time to deploy: ~3 minutes
```

### 5. State-Less User Experience
- No local state files to manage
- State automatically stored in provider (S3/DynamoDB)
- Auto-discovery of state location
- Transparent state management

### 6. Interactive Mode
```bash
genesys
> What would you like to deploy?
  1. Static website
  2. Database
  3. API endpoint
  4. Serverless function
  5. Web application
  
> Selected: Static website
> Domain name (optional): example.com
> Enable CDN? (Y/n): Y
> Preview or Apply? (p/A): p

[Shows plan...]
```

## MVP Technical Architecture

### Core Components

```
genesys/
├── cmd/
│   └── genesys/
│       └── main.go          # Single CLI entry point
├── pkg/
│   ├── outcomes/            # Outcome definitions
│   │   ├── static_site.go
│   │   ├── database.go
│   │   ├── function.go
│   │   └── registry.go
│   ├── discovery/           # Resource discovery
│   │   ├── scanner.go       # Find existing resources
│   │   └── adopter.go       # Adopt existing resources
│   ├── planner/            # Plan generation
│   │   ├── human.go        # Human-readable plans
│   │   └── executor.go     # Execution engine
│   ├── state/              # State management
│   │   └── provider.go     # Provider-backed state
│   ├── interactive/        # Interactive mode
│   │   ├── prompt.go
│   │   └── wizard.go
│   └── aws/               # AWS provider
│       ├── client.go
│       └── resources/
└── templates/             # Optional templates
    └── outcomes/
```

### Outcome Definition Structure

```go
type Outcome interface {
    Name() string                    // e.g., "static-site"
    Description() string             // Human description
    Discover(ctx Context) (*Discovery, error)  // Find existing
    Plan(ctx Context, opts Options) (*Plan, error)  // Generate plan
    Execute(ctx Context, plan *Plan) error  // Apply changes
    Validate(ctx Context) error     // Post-deploy validation
}

type StaticSiteOutcome struct {
    Domain     string
    EnableCDN  bool
    EnableHTTPS bool
}

func (s *StaticSiteOutcome) Plan(ctx Context, opts Options) (*Plan, error) {
    plan := &Plan{
        Steps: []Step{
            {Description: "Create S3 bucket for hosting"},
            {Description: "Configure bucket for static hosting"},
            {Description: "Set up CloudFront distribution"},
            {Description: "Configure custom domain"},
        },
        Permissions: []string{
            "s3:CreateBucket",
            "cloudfront:CreateDistribution",
        },
        EstimatedCost: "$5/month",
        EstimatedTime: "3 minutes",
    }
    return plan, nil
}
```

### Discovery Mechanism

```go
type Discovery struct {
    ExistingResources []Resource
    Recommendations   []string
    AdoptionPlan     *Plan
}

func DiscoverStaticSite(ctx Context) (*Discovery, error) {
    // 1. Scan for S3 buckets with static hosting
    // 2. Check for CloudFront distributions
    // 3. Look for Route53 zones
    // 4. Return findings with adoption options
}
```

### State Management

```go
type StateManager struct {
    backend StateBackend
}

type StateBackend interface {
    Init() error                    // Auto-setup S3/DynamoDB
    Lock(key string) error          // Distributed locking
    Read(key string) (*State, error)
    Write(key string, state *State) error
}

// Automatic state backend discovery
func AutoDiscoverBackend() StateBackend {
    // 1. Check for existing Genesys state in S3
    // 2. If not found, create new backend
    // 3. Store location in ~/.genesys/config
}
```

### Interactive Mode Flow

```go
type InteractiveWizard struct {
    outcomes []Outcome
}

func (w *InteractiveWizard) Run() error {
    // 1. Select outcome
    outcome := w.selectOutcome()
    
    // 2. Gather parameters
    params := w.gatherParams(outcome)
    
    // 3. Discovery phase
    discovery := outcome.Discover()
    action := w.handleDiscovery(discovery)
    
    // 4. Generate plan
    plan := outcome.Plan(params)
    w.displayPlan(plan)
    
    // 5. Confirm execution
    if w.confirmApply() {
        return outcome.Execute(plan)
    }
    return nil
}
```

## MVP Command Structure

### Basic Commands
```bash
# Preview (default)
genesys static-site --domain example.com
genesys db --type postgres --size small
genesys function --runtime python --trigger http

# Apply
genesys static-site --domain example.com --apply
genesys db --type postgres --apply

# Interactive mode
genesys                        # No arguments launches wizard
genesys --interactive          # Explicit interactive mode

# Discovery
genesys discover               # Scan account for resources
genesys adopt [resource-id]    # Adopt existing resource
```

### Options
```bash
--apply              # Execute the plan
--output json        # JSON output for automation
--profile [name]     # AWS profile to use
--region [region]    # AWS region
--yes               # Skip confirmation prompts
--dry-run           # More detailed preview
```

## MVP Outcomes

### 1. Static Site
```go
Resources:
- S3 bucket (website hosting enabled)
- CloudFront distribution (optional)
- Route53 records (if domain provided)
- ACM certificate (if HTTPS enabled)

Parameters:
- domain: Custom domain (optional)
- enable-cdn: Use CloudFront (default: true)
- enable-https: SSL/TLS (default: true)
- index-doc: Index document (default: index.html)
```

### 2. Database
```go
Resources:
- RDS instance
- VPC + Subnets (if not existing)
- Security groups
- Parameter groups
- Backup configuration

Parameters:
- type: postgres|mysql|mariadb
- size: small|medium|large
- storage: GB of storage
- multi-az: High availability
- backup-retention: Days to retain
```

### 3. Function
```go
Resources:
- Lambda function
- IAM execution role
- API Gateway (if HTTP trigger)
- CloudWatch logs
- EventBridge rule (if scheduled)

Parameters:
- runtime: python|node|go|java
- trigger: http|schedule|s3|sqs
- memory: MB of memory
- timeout: Execution timeout
- schedule: Cron expression (if scheduled)
```

## Implementation Phases

### Phase 1: Core Framework (Week 1)
- [ ] Basic CLI structure
- [ ] Outcome interface definition
- [ ] Plan generation
- [ ] Human-readable output

### Phase 2: AWS Integration (Week 2)
- [ ] AWS SDK integration
- [ ] State backend (S3/DynamoDB)
- [ ] Resource discovery
- [ ] Basic IAM permission checking

### Phase 3: First Outcomes (Week 3)
- [ ] Static site outcome
- [ ] Database outcome  
- [ ] Function outcome

### Phase 4: Interactive Mode (Week 4)
- [ ] Interactive wizard
- [ ] Parameter gathering
- [ ] Discovery integration
- [ ] Adoption workflow

### Phase 5: Polish (Week 5)
- [ ] Error handling
- [ ] Progress indicators
- [ ] Cost estimation
- [ ] Testing suite

## Success Criteria

1. **Simplicity**: Deploy a static site in < 30 seconds
2. **Safety**: Preview by default prevents mistakes
3. **Discovery**: Never create duplicate resources
4. **Clarity**: Anyone can understand the plan
5. **Reliability**: State management "just works"

## Future Enhancements (Post-MVP)

- Multi-cloud support (Azure, GCP)
- More outcomes (container apps, ML workloads)
- Team collaboration features
- Cost optimization suggestions
- Automated testing of infrastructure
- GitOps integration