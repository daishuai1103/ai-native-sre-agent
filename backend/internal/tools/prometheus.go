package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/wk8/go-ordered-map/v2"
)

// PrometheusQueryTool queries Prometheus for a metric.
type PrometheusQueryTool struct{}

func (PrometheusQueryTool) Name() string        { return "prometheus_query" }
func (PrometheusQueryTool) Description() string { return "Query Prometheus for a metric." }
func (PrometheusQueryTool) Schema() *jsonschema.Schema {
	props := orderedmap.New[string, *jsonschema.Schema]()
	props.Set("query", &jsonschema.Schema{Type: "string", Description: "PromQL query"})
	props.Set("time_range", &jsonschema.Schema{Type: "string", Description: "query lookback window", Default: "5m"})
	return &jsonschema.Schema{
		Type:       "object",
		Properties: props,
		Required:   []string{"query"},
	}
}

func (PrometheusQueryTool) Call(ctx context.Context, args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return "prometheus query failed: missing required argument \"query\"", nil
	}

	timeRange := "5m"
	if v, ok := args["time_range"].(string); ok && v != "" {
		timeRange = v
	}

	baseURL := os.Getenv("PROMETHEUS_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9090"
	}

	endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + "/api/v1/query")
	if err != nil {
		return fmt.Sprintf("prometheus query failed: invalid PROMETHEUS_URL: %v", err), nil
	}
	values := endpoint.Query()
	values.Set("query", query)
	endpoint.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Sprintf("prometheus query failed: %v", err), nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Sprintf("prometheus query failed: %v", err), nil
	}
	defer resp.Body.Close()

	var payload struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []any             `json:"value"`
				Values [][]any           `json:"values"`
			} `json:"result"`
			ErrorType string `json:"errorType"`
			Error     string `json:"error"`
		} `json:"data"`
		ErrorType string `json:"errorType"`
		Error     string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Sprintf("prometheus query failed: decode response: %v", err), nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := payload.Error
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Sprintf("prometheus query failed: http %s: %s", resp.Status, msg), nil
	}
	if payload.Status != "" && payload.Status != "success" {
		msg := payload.Data.Error
		if msg == "" {
			msg = payload.Error
		}
		if msg == "" {
			msg = "unknown error"
		}
		return fmt.Sprintf("prometheus query failed: %s", msg), nil
	}

	if len(payload.Data.Result) == 0 {
		return fmt.Sprintf("Prometheus query %q returned no results.", query), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Prometheus query %q returned %d result(s) [%s] over %s.", query, len(payload.Data.Result), payload.Data.ResultType, timeRange)
	for i, r := range payload.Data.Result {
		fmt.Fprintf(&b, "\n%d. metric=%s", i+1, formatMetric(r.Metric))
		if len(r.Value) >= 2 {
			fmt.Fprintf(&b, " value=%v at=%v", r.Value[1], r.Value[0])
		}
		if len(r.Values) > 0 {
			fmt.Fprintf(&b, " samples=%d", len(r.Values))
		}
	}
	return b.String(), nil
}

func formatMetric(metric map[string]string) string {
	if len(metric) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(metric))
	for k, v := range metric {
		parts = append(parts, fmt.Sprintf("%s=%q", k, v))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
