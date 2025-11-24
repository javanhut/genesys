package aws

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/javanhut/genesys/pkg/provider"
)

// MonitoringService implements CloudWatch monitoring using direct API calls
type MonitoringService struct {
	awsProvider *AWSProvider
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(p *AWSProvider) *MonitoringService {
	return &MonitoringService{
		awsProvider: p,
	}
}

// CloudWatch API response structures

// GetMetricStatisticsResponse represents the CloudWatch GetMetricStatistics response
type GetMetricStatisticsResponse struct {
	XMLName    xml.Name `xml:"GetMetricStatisticsResponse"`
	Label      string   `xml:"GetMetricStatisticsResult>Label"`
	Datapoints struct {
		Member []CloudWatchDatapoint `xml:"member"`
	} `xml:"GetMetricStatisticsResult>Datapoints"`
}

// CloudWatchDatapoint represents a single metric datapoint
type CloudWatchDatapoint struct {
	Timestamp   string  `xml:"Timestamp"`
	Average     float64 `xml:"Average"`
	Sum         float64 `xml:"Sum"`
	Minimum     float64 `xml:"Minimum"`
	Maximum     float64 `xml:"Maximum"`
	SampleCount float64 `xml:"SampleCount"`
	Unit        string  `xml:"Unit"`
}

// DescribeAlarmsResponse represents the CloudWatch DescribeAlarms response
type DescribeAlarmsResponse struct {
	XMLName xml.Name `xml:"DescribeAlarmsResponse"`
	Alarms  struct {
		Member []CloudWatchAlarmInfo `xml:"member"`
	} `xml:"DescribeAlarmsResult>MetricAlarms"`
}

// CloudWatchAlarmInfo represents alarm information from CloudWatch
type CloudWatchAlarmInfo struct {
	AlarmName             string  `xml:"AlarmName"`
	AlarmArn              string  `xml:"AlarmArn"`
	AlarmDescription      string  `xml:"AlarmDescription"`
	StateValue            string  `xml:"StateValue"`
	StateReason           string  `xml:"StateReason"`
	MetricName            string  `xml:"MetricName"`
	Namespace             string  `xml:"Namespace"`
	Threshold             float64 `xml:"Threshold"`
	ComparisonOperator    string  `xml:"ComparisonOperator"`
	EvaluationPeriods     int     `xml:"EvaluationPeriods"`
	Period                int     `xml:"Period"`
	Statistic             string  `xml:"Statistic"`
	ActionsEnabled        bool    `xml:"ActionsEnabled"`
	StateUpdatedTimestamp string  `xml:"StateUpdatedTimestamp"`
	AlarmActions          struct {
		Member []string `xml:"member"`
	} `xml:"AlarmActions"`
}

// GetResourceMetrics retrieves metrics for any resource type
func (m *MonitoringService) GetResourceMetrics(ctx context.Context, resourceType, resourceID string, period string) (*provider.MetricsData, error) {
	switch resourceType {
	case "ec2", "compute":
		ec2Metrics, err := m.GetEC2Metrics(ctx, resourceID, period)
		if err != nil {
			return nil, err
		}
		return &provider.MetricsData{
			ResourceID:   resourceID,
			ResourceType: "ec2",
			Period:       period,
			Datapoints:   ec2Metrics.CPUUtilization,
		}, nil
	case "s3", "storage":
		_, err := m.GetS3Metrics(ctx, resourceID, period)
		if err != nil {
			return nil, err
		}
		return &provider.MetricsData{
			ResourceID:   resourceID,
			ResourceType: "s3",
			Period:       period,
		}, nil
	case "lambda", "serverless":
		lambdaMetrics, err := m.GetLambdaMetrics(ctx, resourceID, period)
		if err != nil {
			return nil, err
		}
		return &provider.MetricsData{
			ResourceID:   resourceID,
			ResourceType: "lambda",
			Period:       period,
			Datapoints:   lambdaMetrics.Invocations,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// GetEC2Metrics retrieves CloudWatch metrics for an EC2 instance
func (m *MonitoringService) GetEC2Metrics(ctx context.Context, instanceID string, period string) (*provider.EC2Metrics, error) {
	client, err := m.awsProvider.CreateClient("monitoring")
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch client: %w", err)
	}

	startTime, endTime, periodSeconds := m.parsePeriod(period)

	dimensions := map[string]string{
		"InstanceId": instanceID,
	}

	metrics := &provider.EC2Metrics{
		InstanceID: instanceID,
	}

	// Get CPU Utilization
	cpuMetrics, err := m.getMetricStatistics(client, "AWS/EC2", "CPUUtilization", dimensions, startTime, endTime, periodSeconds, []string{"Average", "Maximum"})
	if err == nil {
		metrics.CPUUtilization = cpuMetrics
	}

	// Get Network In
	networkIn, err := m.getMetricStatistics(client, "AWS/EC2", "NetworkIn", dimensions, startTime, endTime, periodSeconds, []string{"Sum"})
	if err == nil {
		metrics.NetworkIn = networkIn
	}

	// Get Network Out
	networkOut, err := m.getMetricStatistics(client, "AWS/EC2", "NetworkOut", dimensions, startTime, endTime, periodSeconds, []string{"Sum"})
	if err == nil {
		metrics.NetworkOut = networkOut
	}

	// Get Disk Read Operations
	diskReadOps, err := m.getMetricStatistics(client, "AWS/EC2", "DiskReadOps", dimensions, startTime, endTime, periodSeconds, []string{"Sum"})
	if err == nil {
		metrics.DiskReadOps = diskReadOps
	}

	// Get Disk Write Operations
	diskWriteOps, err := m.getMetricStatistics(client, "AWS/EC2", "DiskWriteOps", dimensions, startTime, endTime, periodSeconds, []string{"Sum"})
	if err == nil {
		metrics.DiskWriteOps = diskWriteOps
	}

	// Get Disk Read Bytes
	diskReadBytes, err := m.getMetricStatistics(client, "AWS/EC2", "DiskReadBytes", dimensions, startTime, endTime, periodSeconds, []string{"Sum"})
	if err == nil {
		metrics.DiskReadBytes = diskReadBytes
	}

	// Get Disk Write Bytes
	diskWriteBytes, err := m.getMetricStatistics(client, "AWS/EC2", "DiskWriteBytes", dimensions, startTime, endTime, periodSeconds, []string{"Sum"})
	if err == nil {
		metrics.DiskWriteBytes = diskWriteBytes
	}

	// Get Status Check Failed
	statusFailed, err := m.getMetricStatistics(client, "AWS/EC2", "StatusCheckFailed", dimensions, startTime, endTime, periodSeconds, []string{"Average"})
	if err == nil {
		metrics.StatusCheckFailed = statusFailed
	}

	return metrics, nil
}

// GetS3Metrics retrieves CloudWatch metrics for an S3 bucket
func (m *MonitoringService) GetS3Metrics(ctx context.Context, bucketName string, period string) (*provider.S3Metrics, error) {
	client, err := m.awsProvider.CreateClient("monitoring")
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch client: %w", err)
	}

	startTime, endTime, periodSeconds := m.parsePeriod(period)

	dimensions := map[string]string{
		"BucketName":  bucketName,
		"StorageType": "StandardStorage",
	}

	metrics := &provider.S3Metrics{
		BucketName: bucketName,
	}

	// Get Bucket Size
	bucketSizeMetrics, err := m.getMetricStatistics(client, "AWS/S3", "BucketSizeBytes", dimensions, startTime, endTime, periodSeconds, []string{"Average"})
	if err == nil && len(bucketSizeMetrics) > 0 {
		metrics.BucketSizeBytes = int64(bucketSizeMetrics[len(bucketSizeMetrics)-1].Value)
	}

	// Get Number of Objects
	objectCountMetrics, err := m.getMetricStatistics(client, "AWS/S3", "NumberOfObjects", map[string]string{
		"BucketName":  bucketName,
		"StorageType": "AllStorageTypes",
	}, startTime, endTime, periodSeconds, []string{"Average"})
	if err == nil && len(objectCountMetrics) > 0 {
		metrics.NumberOfObjects = int64(objectCountMetrics[len(objectCountMetrics)-1].Value)
	}

	// Get Request Metrics (requires request metrics to be enabled)
	requestDimensions := map[string]string{
		"BucketName": bucketName,
	}

	allRequests, err := m.getMetricStatistics(client, "AWS/S3", "AllRequests", requestDimensions, startTime, endTime, periodSeconds, []string{"Sum"})
	if err == nil {
		metrics.AllRequests = allRequests
	}

	getRequests, err := m.getMetricStatistics(client, "AWS/S3", "GetRequests", requestDimensions, startTime, endTime, periodSeconds, []string{"Sum"})
	if err == nil {
		metrics.GetRequests = getRequests
	}

	putRequests, err := m.getMetricStatistics(client, "AWS/S3", "PutRequests", requestDimensions, startTime, endTime, periodSeconds, []string{"Sum"})
	if err == nil {
		metrics.PutRequests = putRequests
	}

	return metrics, nil
}

// GetLambdaMetrics retrieves CloudWatch metrics for a Lambda function
func (m *MonitoringService) GetLambdaMetrics(ctx context.Context, functionName string, period string) (*provider.LambdaMetrics, error) {
	client, err := m.awsProvider.CreateClient("monitoring")
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch client: %w", err)
	}

	startTime, endTime, periodSeconds := m.parsePeriod(period)

	dimensions := map[string]string{
		"FunctionName": functionName,
	}

	metrics := &provider.LambdaMetrics{
		FunctionName: functionName,
	}

	// Get Invocations
	invocations, err := m.getMetricStatistics(client, "AWS/Lambda", "Invocations", dimensions, startTime, endTime, periodSeconds, []string{"Sum"})
	if err == nil {
		metrics.Invocations = invocations
	}

	// Get Duration
	duration, err := m.getMetricStatistics(client, "AWS/Lambda", "Duration", dimensions, startTime, endTime, periodSeconds, []string{"Average", "Maximum"})
	if err == nil {
		metrics.Duration = duration
	}

	// Get Errors
	errors, err := m.getMetricStatistics(client, "AWS/Lambda", "Errors", dimensions, startTime, endTime, periodSeconds, []string{"Sum"})
	if err == nil {
		metrics.Errors = errors
	}

	// Get Throttles
	throttles, err := m.getMetricStatistics(client, "AWS/Lambda", "Throttles", dimensions, startTime, endTime, periodSeconds, []string{"Sum"})
	if err == nil {
		metrics.Throttles = throttles
	}

	// Get Concurrent Executions
	concurrent, err := m.getMetricStatistics(client, "AWS/Lambda", "ConcurrentExecutions", dimensions, startTime, endTime, periodSeconds, []string{"Average", "Maximum"})
	if err == nil {
		metrics.ConcurrentExecutions = concurrent
	}

	return metrics, nil
}

// GetResourceHealth retrieves health status for a resource
func (m *MonitoringService) GetResourceHealth(ctx context.Context, resourceType, resourceID string) (*provider.ResourceHealth, error) {
	health := &provider.ResourceHealth{
		ResourceID:   resourceID,
		ResourceType: resourceType,
		LastChecked:  time.Now(),
		Metrics:      make(map[string]float64),
		Status:       "unknown",
	}

	switch resourceType {
	case "ec2", "compute":
		metrics, err := m.GetEC2Metrics(ctx, resourceID, "5m")
		if err != nil {
			health.Status = "unknown"
			health.Issues = append(health.Issues, fmt.Sprintf("Failed to retrieve metrics: %v", err))
			return health, nil
		}

		if len(metrics.CPUUtilization) > 0 {
			latestCPU := metrics.CPUUtilization[len(metrics.CPUUtilization)-1].Value
			health.Metrics["cpu_utilization"] = latestCPU

			if latestCPU > 90 {
				health.Status = "degraded"
				health.Issues = append(health.Issues, fmt.Sprintf("High CPU utilization: %.2f%%", latestCPU))
			} else if latestCPU > 95 {
				health.Status = "unhealthy"
				health.Issues = append(health.Issues, fmt.Sprintf("Critical CPU utilization: %.2f%%", latestCPU))
			} else {
				health.Status = "healthy"
			}
		}

		if len(metrics.StatusCheckFailed) > 0 {
			latestStatus := metrics.StatusCheckFailed[len(metrics.StatusCheckFailed)-1].Value
			if latestStatus > 0 {
				health.Status = "unhealthy"
				health.Issues = append(health.Issues, "Status checks failing")
			}
		}

	case "lambda", "serverless":
		metrics, err := m.GetLambdaMetrics(ctx, resourceID, "5m")
		if err != nil {
			health.Status = "unknown"
			health.Issues = append(health.Issues, fmt.Sprintf("Failed to retrieve metrics: %v", err))
			return health, nil
		}

		health.Status = "healthy"

		if len(metrics.Errors) > 0 {
			totalErrors := 0.0
			for _, dp := range metrics.Errors {
				totalErrors += dp.Value
			}
			health.Metrics["errors"] = totalErrors

			if totalErrors > 10 {
				health.Status = "degraded"
				health.Issues = append(health.Issues, fmt.Sprintf("High error rate: %.0f errors", totalErrors))
			}
		}

		if len(metrics.Throttles) > 0 {
			totalThrottles := 0.0
			for _, dp := range metrics.Throttles {
				totalThrottles += dp.Value
			}
			health.Metrics["throttles"] = totalThrottles

			if totalThrottles > 0 {
				health.Status = "degraded"
				health.Issues = append(health.Issues, fmt.Sprintf("Function throttled: %.0f times", totalThrottles))
			}
		}
	}

	return health, nil
}

// GetAllResourcesHealth retrieves health for all monitored resources
func (m *MonitoringService) GetAllResourcesHealth(ctx context.Context) ([]*provider.ResourceHealth, error) {
	var healthStatuses []*provider.ResourceHealth

	// Get EC2 instances
	instances, err := m.awsProvider.Compute().ListInstances(ctx, map[string]string{"instance-state-name": "running"})
	if err == nil {
		for _, instance := range instances {
			health, err := m.GetResourceHealth(ctx, "ec2", instance.ID)
			if err == nil {
				health.ResourceName = instance.Name
				healthStatuses = append(healthStatuses, health)
			}
		}
	}

	return healthStatuses, nil
}

// ListResourceAlarms lists CloudWatch alarms for a resource
func (m *MonitoringService) ListResourceAlarms(ctx context.Context, resourceID string) ([]*provider.CloudWatchAlarm, error) {
	client, err := m.awsProvider.CreateClient("monitoring")
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch client: %w", err)
	}

	params := map[string]string{
		"Action":  "DescribeAlarms",
		"Version": "2010-08-01",
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe alarms: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("DescribeAlarms failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var describeResp DescribeAlarmsResponse
	if err := xml.Unmarshal(body, &describeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var alarms []*provider.CloudWatchAlarm
	for _, alarmInfo := range describeResp.Alarms.Member {
		// Filter alarms related to this resource
		if strings.Contains(alarmInfo.AlarmName, resourceID) {
			updatedAt, _ := time.Parse(time.RFC3339, alarmInfo.StateUpdatedTimestamp)

			var actions []string
			actions = append(actions, alarmInfo.AlarmActions.Member...)

			alarms = append(alarms, &provider.CloudWatchAlarm{
				AlarmName:         alarmInfo.AlarmName,
				AlarmArn:          alarmInfo.AlarmArn,
				AlarmDescription:  alarmInfo.AlarmDescription,
				State:             alarmInfo.StateValue,
				StateReason:       alarmInfo.StateReason,
				MetricName:        alarmInfo.MetricName,
				Namespace:         alarmInfo.Namespace,
				Threshold:         alarmInfo.Threshold,
				ComparisonOp:      alarmInfo.ComparisonOperator,
				EvaluationPeriods: alarmInfo.EvaluationPeriods,
				Period:            alarmInfo.Period,
				Statistic:         alarmInfo.Statistic,
				ActionsEnabled:    alarmInfo.ActionsEnabled,
				AlarmActions:      actions,
				UpdatedAt:         updatedAt,
			})
		}
	}

	return alarms, nil
}

// GetAlarmState retrieves the state of a specific alarm
func (m *MonitoringService) GetAlarmState(ctx context.Context, alarmName string) (string, error) {
	client, err := m.awsProvider.CreateClient("monitoring")
	if err != nil {
		return "", fmt.Errorf("failed to create CloudWatch client: %w", err)
	}

	params := map[string]string{
		"Action":              "DescribeAlarms",
		"Version":             "2010-08-01",
		"AlarmNames.member.1": alarmName,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return "", fmt.Errorf("failed to describe alarm: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return "", fmt.Errorf("DescribeAlarms failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var describeResp DescribeAlarmsResponse
	if err := xml.Unmarshal(body, &describeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(describeResp.Alarms.Member) == 0 {
		return "", fmt.Errorf("alarm not found: %s", alarmName)
	}

	return describeResp.Alarms.Member[0].StateValue, nil
}

// Helper methods

// getMetricStatistics makes a CloudWatch GetMetricStatistics API call
func (m *MonitoringService) getMetricStatistics(client *AWSClient, namespace, metricName string, dimensions map[string]string, startTime, endTime time.Time, period int, statistics []string) ([]provider.MetricDatapoint, error) {
	params := map[string]string{
		"Action":     "GetMetricStatistics",
		"Version":    "2010-08-01",
		"Namespace":  namespace,
		"MetricName": metricName,
		"StartTime":  startTime.UTC().Format(time.RFC3339),
		"EndTime":    endTime.UTC().Format(time.RFC3339),
		"Period":     strconv.Itoa(period),
	}

	// Add dimensions
	dimIndex := 1
	for key, value := range dimensions {
		params[fmt.Sprintf("Dimensions.member.%d.Name", dimIndex)] = key
		params[fmt.Sprintf("Dimensions.member.%d.Value", dimIndex)] = value
		dimIndex++
	}

	// Add statistics
	for i, stat := range statistics {
		params[fmt.Sprintf("Statistics.member.%d", i+1)] = stat
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric statistics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("GetMetricStatistics failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var statsResp GetMetricStatisticsResponse
	if err := xml.Unmarshal(body, &statsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var datapoints []provider.MetricDatapoint
	for _, dp := range statsResp.Datapoints.Member {
		timestamp, _ := time.Parse(time.RFC3339, dp.Timestamp)

		// Use the first available statistic
		value := dp.Average
		statistic := "Average"
		if value == 0 && dp.Sum != 0 {
			value = dp.Sum
			statistic = "Sum"
		} else if value == 0 && dp.Maximum != 0 {
			value = dp.Maximum
			statistic = "Maximum"
		}

		datapoints = append(datapoints, provider.MetricDatapoint{
			Timestamp: timestamp,
			Value:     value,
			Unit:      dp.Unit,
			Statistic: statistic,
		})
	}

	return datapoints, nil
}

// parsePeriod converts a period string to start/end times and period seconds
func (m *MonitoringService) parsePeriod(period string) (time.Time, time.Time, int) {
	endTime := time.Now()
	var startTime time.Time
	periodSeconds := 300 // Default 5 minutes

	switch period {
	case "5m", "5min":
		startTime = endTime.Add(-5 * time.Minute)
		periodSeconds = 60
	case "15m", "15min":
		startTime = endTime.Add(-15 * time.Minute)
		periodSeconds = 60
	case "30m", "30min":
		startTime = endTime.Add(-30 * time.Minute)
		periodSeconds = 60
	case "1h", "hour":
		startTime = endTime.Add(-1 * time.Hour)
		periodSeconds = 300
	case "3h":
		startTime = endTime.Add(-3 * time.Hour)
		periodSeconds = 300
	case "6h":
		startTime = endTime.Add(-6 * time.Hour)
		periodSeconds = 300
	case "12h":
		startTime = endTime.Add(-12 * time.Hour)
		periodSeconds = 3600
	case "24h", "day", "1d":
		startTime = endTime.Add(-24 * time.Hour)
		periodSeconds = 3600
	case "3d":
		startTime = endTime.Add(-3 * 24 * time.Hour)
		periodSeconds = 3600
	case "7d", "week", "1w":
		startTime = endTime.Add(-7 * 24 * time.Hour)
		periodSeconds = 3600
	case "14d", "2w":
		startTime = endTime.Add(-14 * 24 * time.Hour)
		periodSeconds = 3600
	case "30d", "month", "1M":
		startTime = endTime.Add(-30 * 24 * time.Hour)
		periodSeconds = 3600
	default:
		// Default to 1 hour
		startTime = endTime.Add(-1 * time.Hour)
		periodSeconds = 300
	}

	return startTime, endTime, periodSeconds
}
