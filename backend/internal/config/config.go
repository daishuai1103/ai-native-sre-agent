package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	OpenAIAPIKey   string
	OpenAIBaseURL  string
	OpenAIModel    string
	AgentDryRun    bool
	PrometheusURL  string
	NetBoxAPIToken string
	ServerPort     string
	AlerterEnabled bool
	AlerterIntervalSec int
	// Router config
	ONNXModelPath    string
	ONNXVocabPath    string
	RouterThreshold  float64
}

func Load() Config {
	return Config{
		OpenAIAPIKey:     getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL:    getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIModel:      getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		AgentDryRun:      getEnv("AGENT_DRY_RUN", "true") == "true",
		PrometheusURL:    getEnv("PROMETHEUS_URL", "http://localhost:9090"),
		NetBoxAPIToken:   getEnv("NETBOX_API_TOKEN", ""),
		ServerPort:       getEnv("SERVER_PORT", "8081"),
		AlerterEnabled:   getEnv("ALERTER_ENABLED", "false") == "true",
		AlerterIntervalSec: getEnvInt("ALERTER_INTERVAL", 30),
		ONNXModelPath:    getEnv("ONNX_MODEL_PATH", "internal/router/model.onnx"),
		ONNXVocabPath:    getEnv("ONNX_VOCAB_PATH", "internal/router/vocab.json"),
		RouterThreshold:  getEnvFloat("ROUTER_THRESHOLD", 0.6),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
