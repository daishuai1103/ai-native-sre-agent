package router

import "strings"

// RouteDecision captures the routing result.
type RouteDecision struct {
	Category   string  `json:"category"`
	Source     string  `json:"source"`
	Confidence float64 `json:"confidence"`
}

// RuleRouter routes alerts using keyword rules.
type RuleRouter struct{}

// keywordRules maps category to list of keywords (both Chinese and English).
var keywordRules = map[string][]string{
	"cache": {
		"redis", "cache", "缓存", "内存",
	},
	"container": {
		"cpu", "memory", "mem", "container", "docker",
		"容器", "内存使用率", "cpu使用率", "飙升",
	},
	"db": {
		"postgres", "mysql", "db", "database", "sql", "slow query",
		"数据库", "慢查询", "postgres", "pg_",
	},
	"network": {
		"nginx", "network", "502", "503", "connection", "timeout", "latency",
		"网络", "连接", "超时", "延迟",
	},
	"web": {
		"netbox", "web", "http", "500", "404", "api", "service",
		"页面", "接口", "服务", "500错误",
	},
}

func (RuleRouter) Route(alert string) RouteDecision {
	lower := strings.ToLower(alert)

	// Count keyword matches per category
	bestCategory := "unknown"
	bestCount := 0

	for cat, keywords := range keywordRules {
		count := 0
		for _, kw := range keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				count++
			}
		}
		if count > bestCount {
			bestCount = count
			bestCategory = cat
		}
	}

	// Special: if "redis" found, always cache (even if "memory" also matches)
	if strings.Contains(lower, "redis") {
		bestCategory = "cache"
		bestCount++
	}

	confidence := 1.0
	if bestCount == 0 {
		confidence = 0.3
	} else if bestCount == 1 {
		confidence = 0.7
	} else {
		confidence = 0.95
	}

	return RouteDecision{
		Category:   bestCategory,
		Source:     "rule",
		Confidence: confidence,
	}
}
