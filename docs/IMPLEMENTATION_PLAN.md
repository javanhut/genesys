# Genesys Implementation Plan

## Phase 0: Skeleton Foundation

### Core CLI Scaffold
```go
// Main command structure
genesys execute [intent] [flags]    // Preview/apply mode
genesys interact                    // Interactive wizard
genesys version                     // Version info
genesys help                        // Help system
```

### Intent Parser
**Supported intents (Phase 0 subset):**
- `network` → VPC + Subnets
- `bucket` → S3 bucket with best practices
- `function` → Lambda function

**Parser architecture:**
```go
type Intent struct {
    Type       string              // network|bucket|function
    Action     string              // create|adopt|modify
    Parameters map[string]string   // Intent-specific params
    Modifiers  []string           // Additional flags/options
}

type IntentParser interface {
    Parse(args []string) (*Intent, error)
    Validate(intent *Intent) error
}
```

### English Plan Formatter
```go
type PlanStep struct {
    Action      string   // What will happen
    Resource    string   // What resource
    Reason      string   // Why this is needed
    IAMActions  []string // Required permissions
}

type HumanPlan struct {
    Summary     string
    Steps       []PlanStep
    Permissions IAMForecast
    Cost        string
    Duration    string
}

// Example output:
Plan: Deploy secure S3 bucket

What will happen:
1. Create S3 bucket 'my-data-bucket'
   → Store your application data securely
   
2. Enable versioning on bucket
   → Protect against accidental deletion
   
3. Apply encryption with AWS managed keys
   → Secure data at rest
   
4. Block all public access
   → Prevent data exposure

Permissions needed:
- s3:CreateBucket
- s3:PutBucketVersioning
- s3:PutBucketEncryption
- s3:PutBucketPublicAccessBlock

Estimated cost: ~$0.023/GB per month
Time to complete: ~30 seconds
```

### Unit Test Structure
```go
// tests/parser_test.go
func TestIntentParser(t *testing.T) {
    cases := []struct{
        input    string
        expected Intent
    }{
        {"bucket my-bucket", Intent{Type: "bucket", ...}},
        {"network vpc-prod", Intent{Type: "network", ...}},
        {"function api-handler", Intent{Type: "function", ...}},
    }
}

// tests/formatter_test.go
func TestPlanFormatter(t *testing.T) {
    // Test human-readable output
    // Verify IAM forecast accuracy
    // Check cost estimation
}
```

## Phase 1: Core AWS Implementation

### AWS Inventory System
```go
type Inventory interface {
    DiscoverVPCs(ctx context.Context) ([]*VPC, error)
    DiscoverSubnets(ctx context.Context) ([]*Subnet, error)
    DiscoverS3Buckets(ctx context.Context) ([]*Bucket, error)
    DiscoverRDSInstances(ctx context.Context) ([]*DBInstance, error)
    DiscoverLambdas(ctx context.Context) ([]*Function, error)
}

type InventoryResult struct {
    Resources   []Resource
    Adoptable   []Resource  // Can be managed by Genesys
    Conflicts   []Conflict  // Naming or config conflicts
    Suggestions []string    // Recommendations
}
```

### Planner & Executor
```go
type Planner interface {
    PlanNetwork(intent *Intent, inventory *InventoryResult) (*Plan, error)
    PlanBucket(intent *Intent, inventory *InventoryResult) (*Plan, error)
    PlanStaticSite(intent *Intent, inventory *InventoryResult) (*Plan, error)
}

type Executor interface {
    Execute(ctx context.Context, plan *Plan) (*ExecutionResult, error)
    Validate(ctx context.Context, result *ExecutionResult) error
    Rollback(ctx context.Context, result *ExecutionResult) error
}

// Default-or-create pattern
func PlanNetwork(intent *Intent, inv *InventoryResult) (*Plan, error) {
    // 1. Check for default VPC
    // 2. If exists and suitable, use it
    // 3. Otherwise, create new VPC with best practices
    // 4. Always ensure proper subnets (public/private)
}
```

### State Management (S3 + DynamoDB)
```go
type StateBackend struct {
    bucket   string        // S3 bucket for state
    table    string        // DynamoDB for locking
    region   string
}

func (s *StateBackend) AutoSetup() error {
    // 1. Check for existing genesys state bucket
    // 2. If not exists, create with:
    //    - Versioning enabled
    //    - Encryption enabled
    //    - Lifecycle rules
    // 3. Create DynamoDB table for locks
    // 4. Save config to ~/.genesys/backend
}

type DriftDetector struct {
    backend StateBackend
}

func (d *DriftDetector) DetectDrift(resource Resource) (*DriftReport, error) {
    // Compare actual vs expected state
    // Return human-readable drift report
}
```

### IAM Forecast & Ephemeral Roles
```go
type IAMForecast struct {
    RequiredActions   []string
    RequiredResources []string
    EstimatedPolicy   string
}

type EphemeralRoleManager struct {
    ttl time.Duration  // Default: 1 hour
}

func (e *EphemeralRoleManager) CreateRole(forecast IAMForecast) (*Role, error) {
    // 1. Create minimal IAM role
    // 2. Attach precise policy
    // 3. Set expiration via Lambda cleaner
    // 4. Return credentials
}
```

## Phase 2: Lambda Builder System

### Python Langpack with UV
```go
type PythonLangpack struct {
    builder  *AL2023Builder
    packager *UVPackager
}

func (p *PythonLangpack) Build(source string) (*Artifact, error) {
    // 1. Detect requirements.txt/pyproject.toml
    // 2. Use uv for fast dependency resolution
    // 3. Build on AL2023 base image
    // 4. Package as zip or container
}

type AL2023Builder struct {
    baseImage string // Amazon Linux 2023
    cache     BuildCache
}
```

### Layer Management
```go
type LayerManager struct {
    registry LayerRegistry
}

type Layer struct {
    Hash        string   // Content hash
    Runtime     string   // python3.11, nodejs18, etc
    Arch        string   // x86_64, arm64
    Dependencies []string // Package list
}

func (l *LayerManager) PublishOrReuse(deps []string) (*Layer, error) {
    // 1. Calculate content hash
    // 2. Check if layer exists
    // 3. If exists, reuse
    // 4. Otherwise, build and publish
    // 5. Track in registry
}
```

### Lambda Triggers
```go
type TriggerConfig struct {
    Type   string // function-url|sqs|eventbridge
    Config map[string]interface{}
}

func ConfigureFunctionURL(fn *Function) (*FunctionURL, error) {
    // Enable Function URL with CORS
    // Return public endpoint
}

func ConfigureSQSTrigger(fn *Function, queue *Queue) error {
    // Set up event source mapping
    // Configure batch size and visibility
}

func ConfigureEventBridge(fn *Function, rule *Rule) error {
    // Create rule with schedule/pattern
    // Add Lambda as target
}
```

### Interactive Lambda Wizard
```go
type LambdaWizard struct {
    watcher FileWatcher
}

func (w *LambdaWizard) Run() error {
    // 1. Select runtime (Python/Node/Go)
    // 2. Choose trigger type
    // 3. Edit function code
    // 4. Watch for changes
    // 5. Auto-replan on save
    // 6. Hot reload if applied
}

type FileWatcher struct {
    path     string
    onChange func() error
}
```

## Phase 3: Polish & Guardrails

### Policy Assertions
```go
type PolicyEngine struct {
    rules []PolicyRule
}

type PolicyRule interface {
    Evaluate(resource Resource) (*Violation, error)
}

// Built-in policies
var DefaultPolicies = []PolicyRule{
    BlockPublicBuckets{},
    RequireEncryption{},
    EnforceTagging{},
    PreventRootAccount{},
}

func (p *PolicyEngine) Validate(plan *Plan) error {
    // Check all resources against policies
    // Block execution if violations
    // Suggest fixes
}
```

### Cost Estimation
```go
type CostEstimator struct {
    pricing PricingAPI
}

func (c *CostEstimator) EstimatePlan(plan *Plan) (*CostBreakdown, error) {
    // Coarse estimation based on:
    // - Instance types/sizes
    // - Storage volumes
    // - Data transfer estimates
    // - Lambda invocations
}

type CostBreakdown struct {
    Monthly  float64
    Hourly   float64
    PerUnit  map[string]float64
    Warning  string  // If costs seem high
}
```

### Undo/Rollback UX
```go
type UndoManager struct {
    history []ExecutionResult
}

func (u *UndoManager) Undo(id string) error {
    // 1. Find execution by ID
    // 2. Generate reverse plan
    // 3. Show undo preview
    // 4. Execute rollback
    // 5. Verify state restored
}

// CLI usage:
// genesys undo                    // Undo last action
// genesys undo <execution-id>     // Undo specific action
// genesys history                  // Show execution history
```

## Acceptance Criteria Validation

### Test Scenarios

1. **Preview-only default**
```bash
genesys execute bucket my-bucket
# Should show plan without making changes
# Must include IAM forecast
```

2. **Bucket creation idempotency**
```bash
genesys execute bucket my-bucket --apply
# First run: creates bucket
# Second run: detects existing, no changes
```

3. **Interactive S3 adoption**
```bash
genesys interact
# Select: S3 bucket
# Detect existing buckets
# Offer adoption choice
# Save as template
```

4. **Static site deployment**
```bash
genesys execute static-site --domain example.com --apply
# Creates S3 bucket
# Configures static hosting
# Sets up CloudFront
# Outputs working URL
```

5. **Lambda Python function**
```bash
genesys execute function api-handler --runtime python --apply
# Builds with uv
# Creates/reuses layer
# Deploys function
# Exposes Function URL
```

6. **Auto-managed state**
```bash
# No state configuration needed
# Automatically creates S3 bucket
# Sets up DynamoDB table
# Handles locking
```

## Risk Mitigation Strategies

### Throttling & Consistency
```go
type RetryConfig struct {
    MaxAttempts  int
    BackoffBase  time.Duration
    BackoffMax   time.Duration
}

func WithRetry(fn func() error, config RetryConfig) error {
    // Exponential backoff with jitter
    // Verify loops for eventual consistency
    // Circuit breaker for repeated failures
}
```

### IAM Management
```go
type IAMOptimizer struct {
    reuseThreshold time.Duration  // Reuse roles created within window
    ttl            time.Duration  // Auto-cleanup after expiry
}

func (i *IAMOptimizer) GetOrCreateRole(requirements IAMForecast) (*Role, error) {
    // 1. Check for suitable existing role
    // 2. If found and recent, reuse
    // 3. Otherwise create minimal new role
    // 4. Schedule cleanup via Lambda
}
```

### Layer Collision Prevention
```go
func GenerateLayerKey(deps []string, runtime string, arch string) string {
    // Content-addressable storage
    content := strings.Join(deps, "\n")
    hash := sha256.Sum256([]byte(content))
    return fmt.Sprintf("%s-%s-%s-%x", runtime, arch, hash[:8])
}
```

### Adoption Confidence
```go
type AdoptionAnalyzer struct {
    threshold float64  // Default: 0.8
}

type AdoptionScore struct {
    Score       float64
    Confidence  string  // high|medium|low
    Reasons     []string
    Risks       []string
}

func (a *AdoptionAnalyzer) Analyze(resource Resource) (*AdoptionScore, error) {
    // Score based on:
    // - Naming patterns
    // - Tag presence
    // - Configuration similarity
    // - State consistency
    
    // Require confirmation if score < threshold
}
```

## Implementation Timeline

### Week 1: Phase 0 Skeleton
- Day 1-2: CLI scaffold + command structure
- Day 3-4: Intent parser implementation
- Day 5: English plan formatter
- Day 6-7: Unit tests + documentation

### Week 2-3: Phase 1 Core AWS
- Day 1-3: AWS inventory system
- Day 4-5: Network planner/executor
- Day 6-7: Bucket + static-site planner/executor
- Day 8-9: State management (S3/DynamoDB)
- Day 10-11: IAM forecast + ephemeral roles
- Day 12-14: Integration testing

### Week 4: Phase 2 Lambda Builder
- Day 1-2: Python langpack with uv
- Day 3-4: Layer management system
- Day 5-6: Trigger configuration
- Day 7: Interactive wizard with hot reload

### Week 5: Phase 3 Polish
- Day 1-2: Policy assertions
- Day 3: Cost estimation
- Day 4: Undo/rollback UX
- Day 5-7: Documentation + examples

## Success Metrics

1. **Time to first resource**: < 60 seconds from install to deployed bucket
2. **Plan clarity**: 90% of users understand plan without documentation
3. **Safety**: Zero accidental deletions in preview mode
4. **Adoption accuracy**: > 95% correct resource identification
5. **Build speed**: Python Lambda builds < 10 seconds with caching