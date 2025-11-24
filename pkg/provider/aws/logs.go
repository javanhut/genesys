package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/javanhut/genesys/pkg/provider"
)

// LogsService implements CloudWatch Logs using direct API calls
type LogsService struct {
	awsProvider *AWSProvider
}

// NewLogsService creates a new logs service
func NewLogsService(p *AWSProvider) *LogsService {
	return &LogsService{
		awsProvider: p,
	}
}

// CloudWatch Logs API response structures

// DescribeLogGroupsResponse represents the CloudWatch Logs DescribeLogGroups response
type DescribeLogGroupsResponse struct {
	LogGroups []LogGroupInfo `json:"logGroups"`
	NextToken string         `json:"nextToken,omitempty"`
}

// LogGroupInfo represents log group information
type LogGroupInfo struct {
	LogGroupName      string `json:"logGroupName"`
	CreationTime      int64  `json:"creationTime"`
	RetentionInDays   int    `json:"retentionInDays"`
	MetricFilterCount int    `json:"metricFilterCount"`
	StoredBytes       int64  `json:"storedBytes"`
	KmsKeyID          string `json:"kmsKeyId,omitempty"`
}

// DescribeLogStreamsResponse represents the CloudWatch Logs DescribeLogStreams response
type DescribeLogStreamsResponse struct {
	LogStreams []LogStreamInfo `json:"logStreams"`
	NextToken  string          `json:"nextToken,omitempty"`
}

// LogStreamInfo represents log stream information
type LogStreamInfo struct {
	LogStreamName       string `json:"logStreamName"`
	CreationTime        int64  `json:"creationTime"`
	FirstEventTimestamp int64  `json:"firstEventTimestamp,omitempty"`
	LastEventTimestamp  int64  `json:"lastEventTimestamp,omitempty"`
	LastIngestionTime   int64  `json:"lastIngestionTime,omitempty"`
	UploadSequenceToken string `json:"uploadSequenceToken,omitempty"`
	StoredBytes         int64  `json:"storedBytes,omitempty"`
}

// FilterLogEventsResponse represents the CloudWatch Logs FilterLogEvents response
type FilterLogEventsResponse struct {
	Events             []LogEventInfo      `json:"events"`
	NextToken          string              `json:"nextToken,omitempty"`
	SearchedLogStreams []SearchedLogStream `json:"searchedLogStreams,omitempty"`
}

// LogEventInfo represents a log event
type LogEventInfo struct {
	LogStreamName string `json:"logStreamName"`
	Timestamp     int64  `json:"timestamp"`
	Message       string `json:"message"`
	IngestionTime int64  `json:"ingestionTime"`
	EventID       string `json:"eventId"`
}

// SearchedLogStream represents a searched log stream
type SearchedLogStream struct {
	LogStreamName      string `json:"logStreamName"`
	SearchedCompletely bool   `json:"searchedCompletely"`
}

// ListLogGroups lists all log groups
func (l *LogsService) ListLogGroups(ctx context.Context) ([]*provider.LogGroup, error) {
	client, err := l.awsProvider.CreateClient("logs")
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch Logs client: %w", err)
	}

	requestBody := map[string]interface{}{}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://logs.%s.amazonaws.com/", l.awsProvider.region), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328.DescribeLogGroups")

	if err := client.signRequest(req, jsonData); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to describe log groups: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("DescribeLogGroups failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var describeResp DescribeLogGroupsResponse
	if err := json.Unmarshal(body, &describeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var logGroups []*provider.LogGroup
	for _, lg := range describeResp.LogGroups {
		logGroups = append(logGroups, &provider.LogGroup{
			LogGroupName:      lg.LogGroupName,
			CreationTime:      time.Unix(lg.CreationTime/1000, 0),
			RetentionInDays:   lg.RetentionInDays,
			MetricFilterCount: lg.MetricFilterCount,
			StoredBytes:       lg.StoredBytes,
			KmsKeyID:          lg.KmsKeyID,
		})
	}

	return logGroups, nil
}

// ListLogStreams lists log streams in a log group
func (l *LogsService) ListLogStreams(ctx context.Context, logGroupName string) ([]*provider.LogStream, error) {
	client, err := l.awsProvider.CreateClient("logs")
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch Logs client: %w", err)
	}

	requestBody := map[string]interface{}{
		"logGroupName": logGroupName,
		"descending":   true,
		"orderBy":      "LastEventTime",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://logs.%s.amazonaws.com/", l.awsProvider.region), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328.DescribeLogStreams")

	if err := client.signRequest(req, jsonData); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to describe log streams: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("DescribeLogStreams failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var describeResp DescribeLogStreamsResponse
	if err := json.Unmarshal(body, &describeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var logStreams []*provider.LogStream
	for _, ls := range describeResp.LogStreams {
		logStreams = append(logStreams, &provider.LogStream{
			LogStreamName:       ls.LogStreamName,
			CreationTime:        time.Unix(ls.CreationTime/1000, 0),
			FirstEventTime:      time.Unix(ls.FirstEventTimestamp/1000, 0),
			LastEventTime:       time.Unix(ls.LastEventTimestamp/1000, 0),
			LastIngestionTime:   time.Unix(ls.LastIngestionTime/1000, 0),
			UploadSequenceToken: ls.UploadSequenceToken,
			StoredBytes:         ls.StoredBytes,
		})
	}

	return logStreams, nil
}

// GetLogEvents retrieves log events from a specific log stream
func (l *LogsService) GetLogEvents(ctx context.Context, logGroupName, logStreamName string, startTime, endTime int64) ([]*provider.LogEvent, error) {
	client, err := l.awsProvider.CreateClient("logs")
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch Logs client: %w", err)
	}

	requestBody := map[string]interface{}{
		"logGroupName":  logGroupName,
		"logStreamName": logStreamName,
	}

	if startTime > 0 {
		requestBody["startTime"] = startTime
	}
	if endTime > 0 {
		requestBody["endTime"] = endTime
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://logs.%s.amazonaws.com/", l.awsProvider.region), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328.GetLogEvents")

	if err := client.signRequest(req, jsonData); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get log events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("GetLogEvents failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var filterResp FilterLogEventsResponse
	if err := json.Unmarshal(body, &filterResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var events []*provider.LogEvent
	for _, event := range filterResp.Events {
		events = append(events, &provider.LogEvent{
			Timestamp:     time.Unix(event.Timestamp/1000, 0),
			Message:       event.Message,
			LogStreamName: event.LogStreamName,
			EventID:       event.EventID,
			IngestionTime: time.Unix(event.IngestionTime/1000, 0),
		})
	}

	return events, nil
}

// GetLambdaLogs retrieves logs for a Lambda function
func (l *LogsService) GetLambdaLogs(ctx context.Context, functionName string, startTime, endTime int64, limit int) ([]*provider.LogEvent, error) {
	client, err := l.awsProvider.CreateClient("logs")
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch Logs client: %w", err)
	}

	logGroupName := fmt.Sprintf("/aws/lambda/%s", functionName)

	requestBody := map[string]interface{}{
		"logGroupName": logGroupName,
		"interleaved":  true,
	}

	if startTime > 0 {
		requestBody["startTime"] = startTime
	} else {
		// Default to last hour
		requestBody["startTime"] = time.Now().Add(-1*time.Hour).Unix() * 1000
	}

	if endTime > 0 {
		requestBody["endTime"] = endTime
	} else {
		requestBody["endTime"] = time.Now().Unix() * 1000
	}

	if limit > 0 {
		requestBody["limit"] = limit
	} else {
		requestBody["limit"] = 100
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://logs.%s.amazonaws.com/", l.awsProvider.region), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328.FilterLogEvents")

	if err := client.signRequest(req, jsonData); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to filter log events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("FilterLogEvents failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var filterResp FilterLogEventsResponse
	if err := json.Unmarshal(body, &filterResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var events []*provider.LogEvent
	for _, event := range filterResp.Events {
		events = append(events, &provider.LogEvent{
			Timestamp:     time.Unix(event.Timestamp/1000, 0),
			Message:       event.Message,
			LogStreamName: event.LogStreamName,
			EventID:       event.EventID,
			IngestionTime: time.Unix(event.IngestionTime/1000, 0),
		})
	}

	// Sort by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	return events, nil
}

// TailLambdaLogs tails Lambda logs in real-time
func (l *LogsService) TailLambdaLogs(ctx context.Context, functionName string) (<-chan *provider.LogEvent, <-chan error) {
	eventChan := make(chan *provider.LogEvent, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(eventChan)
		defer close(errChan)

		lastTimestamp := time.Now().Add(-5*time.Minute).Unix() * 1000
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				events, err := l.GetLambdaLogs(ctx, functionName, lastTimestamp, 0, 100)
				if err != nil {
					errChan <- err
					return
				}

				for _, event := range events {
					eventTimestamp := event.Timestamp.Unix() * 1000
					if eventTimestamp > lastTimestamp {
						select {
						case eventChan <- event:
							lastTimestamp = eventTimestamp
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return eventChan, errChan
}
