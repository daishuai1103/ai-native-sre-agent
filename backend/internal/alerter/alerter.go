package alerter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ai-native-sre-agent/backend/internal/models"
)

// AlertStatus represents the processing status of an alert.
type AlertStatus string

const (
	StatusDetected   AlertStatus = "detected"
	StatusProcessing AlertStatus = "processing"
	StatusProcessed  AlertStatus = "processed"  // 诊断完成（DRY_RUN 模式）
	StatusResolved   AlertStatus = "resolved"   // 已自愈（真正执行了修复）
	StatusFailed     AlertStatus = "failed"
)

// Alert represents a detected anomaly.
type Alert struct {
	ID           string      `json:"id"`
	Source       string      `json:"source"`     // prometheus, docker
	Severity     string      `json:"severity"`   // warning, critical
	Message      string      `json:"message"`
	Container    string      `json:"container"`
	Metric       string      `json:"metric"`
	Value        float64     `json:"value"`
	Threshold    float64     `json:"threshold"`
	DetectedAt   time.Time   `json:"detected_at"`
	Resolved     bool        `json:"resolved"`
	SessionID    string      `json:"session_id,omitempty"`
	Status       AlertStatus `json:"status"`
	DurationMs   int64       `json:"duration_ms,omitempty"`
	ResolvedAt   *time.Time  `json:"resolved_at,omitempty"`
	ErrorMsg     string      `json:"error_msg,omitempty"`
}

// EventPublisher is the interface for publishing SSE events.
type EventPublisher interface {
	Publish(event models.StepEvent)
}

// Alerter periodically checks metrics and emits alerts.
type Alerter struct {
	promURL      string
	interval     time.Duration
	pub          EventPublisher
	alerts       []*Alert
	mu           sync.RWMutex
	stopCh       chan struct{}
	activeAlerts map[string]bool // key: container:metric
	onAlert      func(alert *Alert) string
}

func New(promURL string, interval time.Duration, pub EventPublisher) *Alerter {
	return &Alerter{
		promURL:      promURL,
		interval:     interval,
		pub:          pub,
		alerts:       []*Alert{},
		stopCh:       make(chan struct{}),
		activeAlerts: make(map[string]bool),
	}
}

func (a *Alerter) SetOnAlert(fn func(alert *Alert) string) {
	a.onAlert = fn
}

func (a *Alerter) Start(ctx context.Context) {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	log.Printf("[Alerter] started, interval=%s", a.interval)
	a.checkAll()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[Alerter] stopped")
			return
		case <-ticker.C:
			a.checkAll()
		case <-a.stopCh:
			return
		}
	}
}

func (a *Alerter) Stop() {
	close(a.stopCh)
}

// MarkProcessing marks an alert as being processed.
func (a *Alerter) MarkProcessing(alertID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, alert := range a.alerts {
		if alert.ID == alertID {
			alert.Status = StatusProcessing
			break
		}
	}
}

// MarkProcessed marks an alert as processed (diagnosed but no action taken, e.g. dry-run).
func (a *Alerter) MarkProcessed(alertID string, durationMs int64) {
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, alert := range a.alerts {
		if alert.ID == alertID {
			alert.Status = StatusProcessed
			alert.Resolved = false
			alert.DurationMs = durationMs
			alert.ResolvedAt = &now
			break
		}
	}
}

// MarkResolved marks an alert as resolved with duration.
func (a *Alerter) MarkResolved(alertID string, durationMs int64) {
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, alert := range a.alerts {
		if alert.ID == alertID {
			alert.Status = StatusResolved
			alert.Resolved = true
			alert.DurationMs = durationMs
			alert.ResolvedAt = &now
			break
		}
	}
}

// MarkFailed marks an alert as failed.
func (a *Alerter) MarkFailed(alertID string, errMsg string) {
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, alert := range a.alerts {
		if alert.ID == alertID {
			alert.Status = StatusFailed
			alert.Resolved = false
			alert.ErrorMsg = errMsg
			alert.ResolvedAt = &now
			break
		}
	}
}

// Stats returns alert statistics.
type AlertStats struct {
	Total          int `json:"total"`
	Warning        int `json:"warning"`
	Critical       int `json:"critical"`
	Detected       int `json:"detected"`
	Processing     int `json:"processing"`
	Processed      int `json:"processed"`
	Resolved       int `json:"resolved"`
	Failed         int `json:"failed"`
	FromDocker     int `json:"from_docker"`
	FromProm       int `json:"from_prometheus"`
	AvgDurationMs  float64 `json:"avg_duration_ms"`
}

// GetStats returns current alert statistics.
func (a *Alerter) GetStats() AlertStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := AlertStats{Total: len(a.alerts)}
	var totalDuration int64
	closedCount := 0

	for _, alert := range a.alerts {
		switch alert.Severity {
		case "warning":
			stats.Warning++
		case "critical":
			stats.Critical++
		}
		switch alert.Status {
		case StatusDetected, "":
			stats.Detected++
		case StatusProcessing:
			stats.Processing++
		case StatusProcessed:
			stats.Processed++
			if alert.DurationMs > 0 {
				totalDuration += alert.DurationMs
				closedCount++
			}
		case StatusResolved:
			stats.Resolved++
			if alert.DurationMs > 0 {
				totalDuration += alert.DurationMs
				closedCount++
			}
		case StatusFailed:
			stats.Failed++
		}
		switch alert.Source {
		case "docker":
			stats.FromDocker++
		case "prometheus":
			stats.FromProm++
		}
	}

	if closedCount > 0 {
		stats.AvgDurationMs = float64(totalDuration) / float64(closedCount)
	}

	return stats
}

func (a *Alerter) List() []*Alert {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]*Alert, len(a.alerts))
	copy(result, a.alerts)
	return result
}

func (a *Alerter) checkAll() {
	a.checkDockerMemory()
	a.checkDockerCPU()
	a.checkDockerStatus()
	a.checkPrometheusRedis()
}

// InjectDemoAlerts injects a batch of simulated alerts for demo purposes.
func (a *Alerter) InjectDemoAlerts() int {
	demoAlerts := []*Alert{
		{
			Source:    "docker",
			Severity:  "critical",
			Message:   "Redis 容器内存使用率达到 92%，超过阈值 80%",
			Container: "netbox-redis-1",
			Metric:    "memory",
			Value:     92.5,
			Threshold: 80.0,
		},
		{
			Source:    "docker",
			Severity:  "warning",
			Message:   "PostgreSQL 容器 CPU 使用率达到 75%",
			Container: "netbox-postgres-1",
			Metric:    "cpu",
			Value:     75.3,
			Threshold: 70.0,
		},
		{
			Source:    "prometheus",
			Severity:  "warning",
			Message:   "Nginx 5xx 错误率上升到 5.2%",
			Container: "netbox-nginx-1",
			Metric:    "error_rate",
			Value:     5.2,
			Threshold: 3.0,
		},
		{
			Source:    "docker",
			Severity:  "critical",
			Message:   "NetBox 主进程响应时间超过 3s",
			Container: "netbox-netbox-1",
			Metric:    "response_time",
			Value:     3100,
			Threshold: 2000,
		},
		{
			Source:    "prometheus",
			Severity:  "warning",
			Message:   "Redis 连接数接近上限，当前 95/100",
			Container: "netbox-redis-1",
			Metric:    "connections",
			Value:     95,
			Threshold: 90,
		},
	}

	count := 0
	for _, alert := range demoAlerts {
		a.addAlert(alert)
		count++
	}
	log.Printf("[Alerter] injected %d demo alerts", count)
	return count
}

func (a *Alerter) addAlert(alert *Alert) {
	a.mu.Lock()
	key := alert.Container + ":" + alert.Metric
	if a.activeAlerts[key] {
		a.mu.Unlock()
		return
	}
	a.activeAlerts[key] = true
	alert.ID = fmt.Sprintf("alert-%d", time.Now().UnixNano())
	alert.DetectedAt = time.Now()
	a.alerts = append([]*Alert{alert}, a.alerts...)
	a.mu.Unlock()

	log.Printf("[ALERT] %s: %s (%.2f > %.2f) [%s]",
		alert.Source, alert.Message, alert.Value, alert.Threshold, alert.Severity)

	// Emit alert_detected event globally (for all connected clients to refresh alert list)
	if a.pub != nil {
		a.pub.Publish(models.StepEvent{
			Type:      models.AlertDetected,
			SessionID: "global",
			Agent:     "alerter",
			Step:      alert.Message,
			Payload: map[string]any{
				"alert":     alert.Message,
				"severity":  alert.Severity,
				"source":    alert.Source,
				"container": alert.Container,
			},
			Timestamp: time.Now(),
		})
	}

	if a.onAlert != nil {
		sessionID := a.onAlert(alert)
		a.mu.Lock()
		alert.SessionID = sessionID
		a.mu.Unlock()
		log.Printf("[Alerter] %s triggered session %s", alert.ID, sessionID)
	}
}

// --- Docker checks ---

func (a *Alerter) checkDockerMemory() {
	out, err := exec.Command("docker", "stats", "--no-stream",
		"--format", "{{.Name}}|{{.MemUsage}}|{{.MemPerc}}").Output()
	if err != nil {
		return
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		name := parts[0]
		memPercStr := strings.TrimSuffix(parts[2], "%")
		memPerc, err := strconv.ParseFloat(memPercStr, 64)
		if err != nil {
			continue
		}

		if memPerc > 80 {
			severity := "warning"
			if memPerc > 90 {
				severity = "critical"
			}
			a.addAlert(&Alert{
				Source:    "docker",
				Severity:  severity,
				Message:   fmt.Sprintf("容器 %s 内存使用率 %.1f%%，超过阈值 80%%", name, memPerc),
				Container: name,
				Metric:    "memory_percent",
				Value:     memPerc,
				Threshold: 80,
			})
		}
	}
}

func (a *Alerter) checkDockerCPU() {
	out, err := exec.Command("docker", "stats", "--no-stream",
		"--format", "{{.Name}}|{{.CPUPerc}}").Output()
	if err != nil {
		return
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		cpuPercStr := strings.TrimSuffix(parts[1], "%")
		cpuPerc, err := strconv.ParseFloat(cpuPercStr, 64)
		if err != nil {
			continue
		}

		if cpuPerc > 90 {
			a.addAlert(&Alert{
				Source:    "docker",
				Severity:  "critical",
				Message:   fmt.Sprintf("容器 %s CPU 使用率 %.1f%%，超过阈值 90%%", name, cpuPerc),
				Container: name,
				Metric:    "cpu_percent",
				Value:     cpuPerc,
				Threshold: 90,
			})
		}
	}
}

func (a *Alerter) checkDockerStatus() {
	out, err := exec.Command("docker", "ps", "-a", "--format",
		"{{.Names}}|{{.Status}}").Output()
	if err != nil {
		return
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		status := parts[1]

		if strings.Contains(status, "Exited") || strings.Contains(status, "Dead") {
			a.addAlert(&Alert{
				Source:    "docker",
				Severity:  "critical",
				Message:   fmt.Sprintf("容器 %s 已停止，状态: %s", name, status),
				Container: name,
				Metric:    "container_status",
				Value:     0,
				Threshold: 1,
			})
		}
		if strings.Contains(status, "Paused") {
			a.addAlert(&Alert{
				Source:    "docker",
				Severity:  "warning",
				Message:   fmt.Sprintf("容器 %s 处于暂停状态，可能影响服务可用性", name),
				Container: name,
				Metric:    "container_paused",
				Value:     1,
				Threshold: 0,
			})
		}
	}
}

// --- Prometheus checks ---

type promQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func (a *Alerter) checkPrometheusRedis() {
	query := "redis_memory_used_bytes / redis_memory_max_bytes * 100"
	result, err := queryPrometheus(a.promURL, query)
	if err != nil {
		return
	}

	for _, r := range result.Data.Result {
		if len(r.Value) < 2 {
			continue
		}
		valStr, ok := r.Value[1].(string)
		if !ok {
			continue
		}
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			continue
		}
		instance := r.Metric["instance"]
		if val > 80 {
			severity := "warning"
			if val > 90 {
				severity = "critical"
			}
			a.addAlert(&Alert{
				Source:    "prometheus",
				Severity:  severity,
				Message:   fmt.Sprintf("Redis %s 内存使用率 %.1f%%，超过阈值 80%%", instance, val),
				Container: instance,
				Metric:    "redis_memory_percent",
				Value:     val,
				Threshold: 80,
			})
		}
	}
}

func queryPrometheus(promURL, query string) (*promQueryResult, error) {
	cmd := exec.Command("curl", "-s", fmt.Sprintf("%s/api/v1/query?query=%s", promURL, query))
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var result promQueryResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s", result.Status)
	}
	return &result, nil
}
