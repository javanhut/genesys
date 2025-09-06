# Genesys Enhanced Resource Naming and IAM Role Management

## Summary of Implementation

This implementation completely resolves the two major issues with Genesys resource creation:

1. **Fragile resource naming** - Users no longer need to worry about AWS naming conventions
2. **Lambda IAM role attachment failures** - Robust error handling and automatic retry logic

## 🎯 **Problem 1 SOLVED: Smart Resource Naming**

### **Before (Fragile)**
- Users had to know specific AWS naming rules for each resource type
- Names could fail at execution time with cryptic errors
- No guidance on what makes a valid name
- Manual trial-and-error process

### **After (Intelligent)**
- **Automatic name formatting** for all AWS resource types
- **Real-time validation** with clear error messages
- **Smart suggestions** when names need adjustment
- **User confirmation** before applying changes

### **Example: Lambda Function Naming**
```bash
User enters: "My API Handler!"
System shows: "✓ Name formatted for AWS Lambda: My API Handler! → My-API-Handler"
User confirms: "Use formatted name 'My-API-Handler'? [Y/n]"
```

### **Supported Resource Types:**
- **Lambda Functions**: Removes invalid chars, ensures valid start/end
- **S3 Buckets**: Lowercase, DNS-compliant, globally unique format
- **EC2 Instances**: Supports spaces and special chars per AWS rules
- **IAM Roles**: Proper character set and length validation
- **IAM Policies**: Enterprise naming convention support

## 🎯 **Problem 2 SOLVED: Robust IAM Role Management**

### **Before (Unreliable)**
- Role vs ARN confusion causing 400 errors
- IAM propagation delays causing random failures
- No retry logic for temporary AWS API issues
- Limited to single permissions per service

### **After (Enterprise-Ready)**
- **Intelligent role resolution** (handles both names and ARNs)
- **IAM propagation handling** with exponential backoff
- **Comprehensive retry logic** for AWS eventual consistency
- **Multiple permissions per service** with granular control
- **Automatic rollback** on partial failures

### **Enhanced Permission Management**
```bash
Service: DynamoDB
├── Read-only access (AmazonDynamoDBReadOnlyAccess)
└── Full access (AmazonDynamoDBFullAccess)

Service: S3  
├── Read-only access (AmazonS3ReadOnlyAccess)
└── Full access (AmazonS3FullAccess)

Custom Policies: ✓ Supported
Multiple Policies per Service: ✓ Supported
Automatic Basic Logging: ✓ Always included
```

## 🏗️ **Technical Implementation Details**

### **1. Resource Naming Framework** (`pkg/validation/naming.go`)
- **Comprehensive rule engine** for all AWS resource types
- **Auto-formatting functions** specific to each service
- **Validation with helpful error messages**
- **Smart defaults** and suggestions

```go
// Example usage
formattedName, err := validation.ValidateAndFormatName("lambda", userInput)
if formattedName != userInput {
    fmt.Printf("✓ Name formatted: %s → %s\n", userInput, formattedName)
}
```

### **2. Enhanced IAM Service** (`pkg/provider/aws/iam.go`)
- **CreateRoleWithPolicies()**: Atomic role creation with rollback
- **AttachPolicyWithRetry()**: Handles AWS API throttling
- **waitForRolePropagation()**: Exponential backoff for consistency
- **ValidateRole()**: Comprehensive role validation

### **3. Improved Lambda Service** (`pkg/provider/aws/serverless.go`)
- **resolveRoleArn()**: Handles both role names and ARNs
- **waitForRoleReady()**: Ensures role is ready before function creation
- **createFunctionWithRetry()**: Intelligent retry with detailed error messages
- **Enhanced error handling**: Clear AWS error message translation

### **4. Interactive Workflow Enhancements**
- **Lambda**: Multi-permission selection with granular control
- **EC2**: Smart name formatting with uniqueness checking  
- **S3**: DNS-compliant naming with global uniqueness awareness
- **All**: Real-time validation with user-friendly feedback

## 📋 **Configuration File Structure**

### **Enhanced Lambda IAM Configuration**
```toml
[iam]
role_name = "genesys-lambda-my-api-handler"
required_policies = [
    "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
    "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess",
    "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"
]
custom_policies = [
    "arn:aws:iam::123456789012:policy/MyCustomPolicy"
]
auto_manage = true
auto_cleanup = true

[iam.policy_details]
"arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole" = "Basic CloudWatch Logs access"
"arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess" = "Full access to DynamoDB"
```

## 🚀 **User Experience Improvements**

### **Before**
```bash
❌ Function name contains invalid characters
❌ Role cannot be assumed by Lambda  
❌ InvalidParameterValueException: Role is not authorized
```

### **After**  
```bash
✓ Name formatted for AWS Lambda: My Function! → My-Function
✓ IAM role 'genesys-lambda-my-function' will be created
✓ Selected permissions:
  • Basic CloudWatch Logs (always included)
  • DynamoDB: Full access
  • S3: Read-only access
IAM role not ready, waiting 2s before retry 1/3...
✓ Lambda function created successfully!
```

## 🎯 **Key Benefits**

### **For Users**
- **Zero AWS expertise required** for naming
- **No more trial-and-error** with resource names
- **Clear feedback** on what's happening
- **Reliable deployments** without random IAM failures

### **For Operations**
- **Consistent naming conventions** across all resources
- **Comprehensive audit trail** of permission grants
- **Automatic cleanup** on failures
- **Production-ready error handling**

### **For Developers**
- **Extensible framework** for new resource types
- **Comprehensive test coverage** for edge cases
- **Clean separation** of concerns
- **Enterprise-grade** error handling

## 🔧 **Validation Examples**

### **Lambda Function Names**
```bash
Input: "my-λ-function!"     → Output: "my-lambda-function"
Input: "123function"        → Output: "lambda-123function"  
Input: "very_long_name..."  → Output: "very-long-name" (truncated)
```

### **S3 Bucket Names**
```bash
Input: "My Bucket!"         → Output: "my-bucket"
Input: "CAPS-bucket"        → Output: "caps-bucket"
Input: "a"                  → Output: "genesys-a"
```

### **IAM Role Names**
```bash
Input: "My Role @2024"      → Output: "My-Role-2024"
Input: "#invalid_role"      → Output: "genesys-invalid-role"
```

This comprehensive implementation transforms Genesys from a fragile prototype into a production-ready infrastructure tool that "just works" for users while maintaining enterprise-grade reliability and flexibility.