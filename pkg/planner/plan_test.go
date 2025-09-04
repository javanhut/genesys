package planner

import (
	"strings"
	"testing"
	"time"
)

func TestNewBucketPlan(t *testing.T) {
	params := map[string]string{
		"versioning": "true",
		"encryption": "true",
		"public":     "false",
	}

	plan := NewBucketPlan("test-bucket", params)

	// Test basic fields
	if plan.Title != "Deploy S3 Bucket 'test-bucket'" {
		t.Errorf("Expected title 'Deploy S3 Bucket 'test-bucket'', got %s", plan.Title)
	}

	if plan.Duration != "30 seconds" {
		t.Errorf("Expected duration '30 seconds', got %s", plan.Duration)
	}

	// Test steps
	expectedSteps := []string{
		"Create S3 bucket 'test-bucket'",
		"Enable versioning on bucket",
		"Enable encryption with AWS managed keys",
		"Block all public access",
	}

	if len(plan.Steps) != len(expectedSteps) {
		t.Errorf("Expected %d steps, got %d", len(expectedSteps), len(plan.Steps))
	}

	for i, expectedDesc := range expectedSteps {
		if i >= len(plan.Steps) {
			t.Errorf("Missing step %d: %s", i+1, expectedDesc)
			continue
		}
		if plan.Steps[i].Description != expectedDesc {
			t.Errorf("Step %d: expected '%s', got '%s'", i+1, expectedDesc, plan.Steps[i].Description)
		}
	}

	// Test permissions
	expectedActions := []string{
		"s3:CreateBucket",
		"s3:PutBucketVersioning",
		"s3:PutBucketEncryption",
		"s3:PutBucketPublicAccessBlock",
	}

	if len(plan.Permissions.Actions) != len(expectedActions) {
		t.Errorf("Expected %d permissions, got %d", len(expectedActions), len(plan.Permissions.Actions))
	}

	// Test cost estimate
	if plan.Cost.Monthly <= 0 {
		t.Errorf("Expected positive monthly cost, got %f", plan.Cost.Monthly)
	}

	if plan.Cost.Currency != "USD" {
		t.Errorf("Expected currency USD, got %s", plan.Cost.Currency)
	}
}

func TestNewNetworkPlan(t *testing.T) {
	params := map[string]string{
		"cidr": "10.0.0.0/16",
	}

	plan := NewNetworkPlan("test-vpc", params)

	if plan.Title != "Deploy Network 'test-vpc'" {
		t.Errorf("Expected title 'Deploy Network 'test-vpc'', got %s", plan.Title)
	}

	if !strings.Contains(plan.Description, "10.0.0.0/16") {
		t.Errorf("Expected description to contain CIDR, got %s", plan.Description)
	}

	// Network should have multiple steps
	if len(plan.Steps) < 3 {
		t.Errorf("Expected at least 3 steps for network, got %d", len(plan.Steps))
	}

	// Check for expected steps
	stepDescriptions := make([]string, len(plan.Steps))
	for i, step := range plan.Steps {
		stepDescriptions[i] = step.Description
	}

	expectedSteps := []string{"VPC", "Internet Gateway", "subnet"}
	for _, expected := range expectedSteps {
		found := false
		for _, desc := range stepDescriptions {
			if strings.Contains(strings.ToLower(desc), strings.ToLower(expected)) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find step containing '%s', but didn't find it in %v", expected, stepDescriptions)
		}
	}
}

func TestNewFunctionPlan(t *testing.T) {
	params := map[string]string{
		"runtime": "python3.11",
		"memory":  "512",
		"trigger": "http",
	}

	plan := NewFunctionPlan("test-function", params)

	if plan.Title != "Deploy Function 'test-function'" {
		t.Errorf("Expected title 'Deploy Function 'test-function'', got %s", plan.Title)
	}

	if !strings.Contains(plan.Description, "python3.11") {
		t.Errorf("Expected description to contain runtime, got %s", plan.Description)
	}

	// Should have execution role, function, and log group at minimum
	if len(plan.Steps) < 3 {
		t.Errorf("Expected at least 3 steps for function, got %d", len(plan.Steps))
	}

	// With HTTP trigger, should have function URL step
	hasURLStep := false
	for _, step := range plan.Steps {
		if strings.Contains(step.Description, "Function URL") {
			hasURLStep = true
			break
		}
	}
	if !hasURLStep {
		t.Errorf("Expected Function URL step for HTTP trigger")
	}
}

func TestPlan_ToHumanReadable(t *testing.T) {
	plan := &Plan{
		ID:          "test-123",
		Title:       "Test Plan",
		Description: "A test plan",
		Steps: []PlanStep{
			{
				Description: "First step",
				Reason:      "Because we need to",
			},
			{
				Description: "Second step",
				Optional:    true,
			},
		},
		Permissions: IAMForecast{
			Actions: []string{"s3:CreateBucket", "s3:PutObject"},
		},
		Cost: CostEstimate{
			Monthly:    10.50,
			Currency:   "USD",
			Confidence: "high",
		},
		Duration:  "5 minutes",
		CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	output := plan.ToHumanReadable()

	// Check for key components
	expectedParts := []string{
		"Test Plan",
		"A test plan",
		"First step",
		"Second step",
		"Because we need to",
		"s3:CreateBucket",
		"s3:PutObject",
		"$10.50 USD",
		"5 minutes",
		"test-123",
		"2024-01-01 12:00:00",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("Expected output to contain '%s', but it didn't.\nOutput: %s", part, output)
		}
	}

	// Check for optional step indicator
	if !strings.Contains(output, "◦ 2.") {
		t.Errorf("Expected optional step indicator '◦', but didn't find it.\nOutput: %s", output)
	}

	// Check for regular step indicator
	if !strings.Contains(output, "▶ 1.") {
		t.Errorf("Expected regular step indicator '▶', but didn't find it.\nOutput: %s", output)
	}
}

func TestPlan_ToJSON(t *testing.T) {
	plan := &Plan{
		ID:    "test-123",
		Title: "Test Plan",
		Steps: []PlanStep{
			{
				ID:          "step-1",
				Description: "First step",
			},
		},
		CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	jsonOutput := plan.ToJSON()

	// Should be valid JSON containing key fields
	if !strings.Contains(jsonOutput, `"id": "test-123"`) {
		t.Errorf("Expected JSON to contain plan ID")
	}

	if !strings.Contains(jsonOutput, `"title": "Test Plan"`) {
		t.Errorf("Expected JSON to contain plan title")
	}

	if !strings.Contains(jsonOutput, `"steps"`) {
		t.Errorf("Expected JSON to contain steps")
	}
}

func TestRemoveDuplicates(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b", "d"}
	expected := []string{"a", "b", "c", "d"}

	result := removeDuplicates(input)

	if len(result) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
	}

	// Check that all expected items are present
	for _, expectedItem := range expected {
		found := false
		for _, resultItem := range result {
			if resultItem == expectedItem {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find '%s' in result %v", expectedItem, result)
		}
	}
}