//go:build tracing

package fixtures

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

const (
	// OTELCollectorNamespace is the k8s namespace where the collector runs
	OTELCollectorNamespace = "argo"
	// OTELCollectorPodLabel is the label selector for the collector pod
	OTELCollectorPodLabel = "app=otel-collector"
	// OTELCollectorGRPCPort is the NodePort for gRPC OTLP
	OTELCollectorGRPCPort = "30317"
	// OTELCollectorHTTPPort is the NodePort for HTTP OTLP
	OTELCollectorHTTPPort = "30318"
)

// OTELCollector provides access to the OTEL Collector running in k8s
type OTELCollector struct {
	grpcHost        string
	grpcPort        string
	httpHost        string
	httpPort        string
	mu              sync.RWMutex
	baselineSpanIDs map[string]struct{}
}

// NewOTELCollector creates an OTELCollector that connects to the k8s-deployed collector.
// The collector should already be running via PROFILE=telemetry make start.
func NewOTELCollector(ctx context.Context) (*OTELCollector, error) {
	// Verify the collector pod is running
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", OTELCollectorNamespace,
		"-l", OTELCollectorPodLabel, "-o", "jsonpath={.items[0].status.phase}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to check collector pod status: %w (is PROFILE=telemetry make start running?)", err)
	}
	phase := strings.TrimSpace(string(output))
	if phase != "Running" {
		return nil, fmt.Errorf("collector pod is not running (phase=%s)", phase)
	}

	return &OTELCollector{
		grpcHost: "localhost",
		grpcPort: OTELCollectorGRPCPort,
		httpHost: "localhost",
		httpPort: OTELCollectorHTTPPort,
	}, nil
}

// GRPCEndpoint returns the gRPC endpoint for OTLP export
func (c *OTELCollector) GRPCEndpoint() string {
	return net.JoinHostPort(c.grpcHost, c.grpcPort)
}

// HTTPEndpoint returns the HTTP endpoint for OTLP export
func (c *OTELCollector) HTTPEndpoint() string {
	return fmt.Sprintf("http://%s", net.JoinHostPort(c.httpHost, c.httpPort))
}

// Logs returns the collector pod logs for debugging
func (c *OTELCollector) Logs(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "logs", "-n", OTELCollectorNamespace,
		"-l", OTELCollectorPodLabel, "--tail=100000")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get collector logs: %w (stderr: %s)", err, stderr.String())
	}
	return stdout.String(), nil
}

// Terminate is a no-op for k8s deployment (collector lifecycle managed by k8s)
func (c *OTELCollector) Terminate(ctx context.Context) error {
	return nil
}

// MarkCleared snapshots the current set of span IDs as a baseline; subsequent
// calls to GetSpans (and anything built on it) will exclude these spans.
// This avoids clock-skew issues between the test runner and the k8s cluster.
func (c *OTELCollector) MarkCleared(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	logs, err := c.Logs(ctx)
	if err != nil {
		c.baselineSpanIDs = nil
		return
	}

	spans := parseSpansFromDebugLogs(logs)
	c.baselineSpanIDs = make(map[string]struct{}, len(spans))
	for _, span := range spans {
		c.baselineSpanIDs[span.SpanID] = struct{}{}
	}
}

// CollectedSpan represents a span parsed from the OTEL debug exporter logs
type CollectedSpan struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Name         string
	Kind         int
	StartTime    time.Time
	EndTime      time.Time
	Attributes   map[string]any
	Status       SpanStatus
}

// SpanStatus represents the status of a span
type SpanStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

// GetSpans reads and parses all spans from the collector's debug exporter logs.
// Spans whose IDs were recorded by MarkCleared are excluded.
func (c *OTELCollector) GetSpans(ctx context.Context) ([]CollectedSpan, error) {
	allSpans, err := c.GetAllSpans(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	baseline := c.baselineSpanIDs
	c.mu.RUnlock()

	if len(baseline) == 0 {
		return allSpans, nil
	}

	filtered := make([]CollectedSpan, 0, len(allSpans))
	for _, span := range allSpans {
		if _, excluded := baseline[span.SpanID]; !excluded {
			filtered = append(filtered, span)
		}
	}
	return filtered, nil
}

// GetAllSpans returns all parsed spans without baseline filtering.
func (c *OTELCollector) GetAllSpans(ctx context.Context) ([]CollectedSpan, error) {
	logs, err := c.Logs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get collector logs: %w", err)
	}
	return parseSpansFromDebugLogs(logs), nil
}

// parseSpansFromDebugLogs extracts span information from OTEL debug exporter logs
func parseSpansFromDebugLogs(logs string) []CollectedSpan {
	var spans []CollectedSpan
	lines := strings.Split(logs, "\n")

	var currentSpan *CollectedSpan
	inSpanAttributes := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Resource/scope sections appear between spans in the debug exporter
		// output. Stop capturing attributes so that resource-level
		// "-> key: Type(value)" lines are not mixed into span attributes.
		// We do NOT reset currentSpan here because the span will be saved
		// when the next "Span #" marker is encountered.
		if strings.HasPrefix(line, "Resource") || strings.HasPrefix(line, "ScopeSpans") || strings.HasPrefix(line, "InstrumentationScope") {
			inSpanAttributes = false
			continue
		}

		// Look for span start marker
		if strings.Contains(line, "Span #") {
			if currentSpan != nil {
				spans = append(spans, *currentSpan)
			}
			currentSpan = &CollectedSpan{
				Attributes: make(map[string]any),
			}
			inSpanAttributes = false
			continue
		}

		if currentSpan == nil {
			continue
		}

		// Track when we enter the Attributes section of a span
		if line == "Attributes:" {
			inSpanAttributes = true
			continue
		}

		// Parse span properties
		switch {
		case strings.HasPrefix(line, "Trace ID"):
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				currentSpan.TraceID = strings.TrimSpace(parts[1])
			}
		case strings.HasPrefix(line, "Parent ID"):
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				currentSpan.ParentSpanID = strings.TrimSpace(parts[1])
			}
		case strings.HasPrefix(line, "ID") && !strings.HasPrefix(line, "ID:"):
			// Handle "ID       : xxx" format
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				currentSpan.SpanID = strings.TrimSpace(parts[1])
			}
		case strings.HasPrefix(line, "Name"):
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				currentSpan.Name = strings.TrimSpace(parts[1])
			}
		case strings.HasPrefix(line, "Kind"):
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				kindStr := strings.TrimSpace(parts[1])
				switch kindStr {
				case "Internal":
					currentSpan.Kind = 1
				case "Server":
					currentSpan.Kind = 2
				case "Client":
					currentSpan.Kind = 3
				case "Producer":
					currentSpan.Kind = 4
				case "Consumer":
					currentSpan.Kind = 5
				}
			}
		case strings.HasPrefix(line, "Start time"):
			// Format: "Start time     : 2026-01-30 09:14:42.089153775 +0000 UTC"
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				tsStr := strings.TrimSpace(parts[1])
				if t, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", tsStr); err == nil {
					currentSpan.StartTime = t
				}
			}
		case strings.HasPrefix(line, "End time"):
			// Format: "End time       : 2026-01-30 09:14:42.091970286 +0000 UTC"
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				tsStr := strings.TrimSpace(parts[1])
				if t, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", tsStr); err == nil {
					currentSpan.EndTime = t
				}
			}
		case strings.HasPrefix(line, "->") && inSpanAttributes:
			// Attribute line: "     -> key: Type(value)"
			line = strings.TrimPrefix(line, "->")
			line = strings.TrimSpace(line)
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Extract value from Type(value) format
				if idx := strings.Index(value, "("); idx != -1 {
					if endIdx := strings.LastIndex(value, ")"); endIdx > idx {
						value = value[idx+1 : endIdx]
					}
				}
				currentSpan.Attributes[key] = value
			}
		}
	}

	// Don't forget the last span
	if currentSpan != nil && currentSpan.Name != "" {
		spans = append(spans, *currentSpan)
	}

	return spans
}

// GetSpanByName returns the first span with the given name
func (c *OTELCollector) GetSpanByName(ctx context.Context, name string) (*CollectedSpan, error) {
	spans, err := c.GetSpans(ctx)
	if err != nil {
		return nil, err
	}

	for _, span := range spans {
		if span.Name == name {
			return &span, nil
		}
	}

	return nil, fmt.Errorf("span with name %q not found", name)
}

// GetSpansByName returns all spans with the given name
func (c *OTELCollector) GetSpansByName(ctx context.Context, name string) ([]CollectedSpan, error) {
	spans, err := c.GetSpans(ctx)
	if err != nil {
		return nil, err
	}

	var result []CollectedSpan
	for _, span := range spans {
		if span.Name == name {
			result = append(result, span)
		}
	}

	return result, nil
}

// GetSpansByTraceID returns all spans with the given trace ID
func (c *OTELCollector) GetSpansByTraceID(ctx context.Context, traceID string) ([]CollectedSpan, error) {
	spans, err := c.GetSpans(ctx)
	if err != nil {
		return nil, err
	}

	var result []CollectedSpan
	for _, span := range spans {
		if span.TraceID == traceID {
			result = append(result, span)
		}
	}

	return result, nil
}

// WaitForSpans waits until at least the specified number of spans are collected or timeout
func (c *OTELCollector) WaitForSpans(ctx context.Context, minCount int, timeout time.Duration) ([]CollectedSpan, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			spans, err := c.GetSpans(ctx)
			if err != nil {
				return nil, err
			}
			if len(spans) >= minCount {
				return spans, nil
			}
			if time.Now().After(deadline) {
				return spans, fmt.Errorf("timeout waiting for %d spans, got %d", minCount, len(spans))
			}
		}
	}
}

// WaitForSpanByName waits until a span with the given name appears or timeout
func (c *OTELCollector) WaitForSpanByName(ctx context.Context, name string, timeout time.Duration) (*CollectedSpan, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			span, err := c.GetSpanByName(ctx, name)
			if err == nil {
				return span, nil
			}
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for span %q", name)
			}
		}
	}
}

// WaitForWorkflowSpan waits until a span named "workflow" appears and returns
// both the workflow span and all collected spans at that point.
func (c *OTELCollector) WaitForWorkflowSpan(ctx context.Context, timeout time.Duration) (*CollectedSpan, []CollectedSpan, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-ticker.C:
			spans, err := c.GetSpans(ctx)
			if err != nil {
				return nil, nil, err
			}
			// Search from the end to find the most recent workflow span
			for i := len(spans) - 1; i >= 0; i-- {
				if spans[i].Name == "workflow" {
					return &spans[i], spans, nil
				}
			}
			if time.Now().After(deadline) {
				names := make(map[string]int)
				for _, s := range spans {
					names[s.Name]++
				}
				return nil, spans, fmt.Errorf("timeout waiting for workflow span, got %d spans: %v", len(spans), names)
			}
		}
	}
}

// HasParent returns true if the span has a parent span ID
func (s *CollectedSpan) HasParent() bool {
	return s.ParentSpanID != ""
}

// TraceIDBytes returns the trace ID as bytes (for comparison with trace.TraceID)
func (s *CollectedSpan) TraceIDBytes() (trace.TraceID, error) {
	var traceID trace.TraceID
	if len(s.TraceID) != 32 {
		return traceID, fmt.Errorf("invalid trace ID length: %d", len(s.TraceID))
	}
	_, err := fmt.Sscanf(s.TraceID, "%032x", &traceID)
	return traceID, err
}

// SpanIDBytes returns the span ID as bytes (for comparison with trace.SpanID)
func (s *CollectedSpan) SpanIDBytes() (trace.SpanID, error) {
	var spanID trace.SpanID
	if len(s.SpanID) != 16 {
		return spanID, fmt.Errorf("invalid span ID length: %d", len(s.SpanID))
	}
	_, err := fmt.Sscanf(s.SpanID, "%016x", &spanID)
	return spanID, err
}

// GetAttribute returns the value of an attribute by key
func (s *CollectedSpan) GetAttribute(key string) (any, bool) {
	v, ok := s.Attributes[key]
	return v, ok
}
