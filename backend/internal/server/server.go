package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ai-native-sre-agent/backend/internal/alerter"
	"github.com/ai-native-sre-agent/backend/internal/agent"
	"github.com/ai-native-sre-agent/backend/internal/config"
	"github.com/ai-native-sre-agent/backend/internal/llm"
	"github.com/ai-native-sre-agent/backend/internal/router"
	"github.com/ai-native-sre-agent/backend/internal/tools"
	"github.com/gin-gonic/gin"
)

// Server holds HTTP handlers and session state.
type Server struct {
	cfg            config.Config
	orchestrator   *agent.Orchestrator
	router         *router.HybridRouter
	alerter        *alerter.Alerter
	sessions       map[string]*agent.ContextState
	sessionToAlert map[string]string // sessionID -> alertID
	mu             sync.RWMutex
	bus            *eventBus
}

func New(cfg config.Config, registry *tools.Registry) *Server {
	client := llm.NewClient(cfg.OpenAIAPIKey, cfg.OpenAIBaseURL, cfg.OpenAIModel)
	bus := newEventBus(50)
	s := &Server{
		cfg:            cfg,
		orchestrator:   agent.NewOrchestrator(registry, client, bus),
		sessions:       make(map[string]*agent.ContextState),
		sessionToAlert: make(map[string]string),
		bus:            bus,
	}

	// Setup hybrid router (ONNX + rule fallback)
	s.router = router.NewHybridRouter(cfg.ONNXModelPath, cfg.ONNXVocabPath, cfg.RouterThreshold)

	// Setup alerter (only if enabled)
	interval := time.Duration(cfg.AlerterIntervalSec) * time.Second
	if interval < 5*time.Second {
		interval = 30 * time.Second
	}
	s.alerter = alerter.New(cfg.PrometheusURL, interval, bus)
	s.alerter.SetOnAlert(func(alert *alerter.Alert) string {
		sessionID := s.triggerSession(alert.Message, alert.ID)
		return sessionID
	})

	if cfg.AlerterEnabled {
		log.Printf("[Server] Alerter enabled, interval=%s", interval)
	} else {
		log.Printf("[Server] Alerter disabled (set ALERTER_ENABLED=true to enable)")
	}

	return s
}

func (s *Server) StartAlerter(ctx context.Context) {
	if s.cfg.AlerterEnabled {
		go s.alerter.Start(ctx)
	}
}

func (s *Server) Register(r *gin.Engine) {
	r.GET("/api/health", s.health)
	r.GET("/api/router/status", s.routerStatus)
	r.GET("/api/alerts", s.listAlerts)
	r.GET("/api/alerts/stats", s.alertStats)
	r.POST("/api/alerts", s.handleAlert)
	r.POST("/api/alerts/inject", s.injectDemoAlerts)
	r.GET("/api/sessions/:id", s.getSession)
	r.GET("/api/sessions/:id/events", s.sseEvents)
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) routerStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"onnx_available": s.router.ONNXAvailable(),
		"threshold":      s.cfg.RouterThreshold,
		"model_path":     s.cfg.ONNXModelPath,
		"subscribers":    s.bus.SubscriberCount(),
	})
}

func (s *Server) listAlerts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"alerts": s.alerter.List()})
}

func (s *Server) alertStats(c *gin.Context) {
	c.JSON(http.StatusOK, s.alerter.GetStats())
}

func (s *Server) injectDemoAlerts(c *gin.Context) {
	count := s.alerter.InjectDemoAlerts()
	c.JSON(http.StatusOK, gin.H{
		"injected": count,
		"message":  fmt.Sprintf("已注入 %d 条模拟告警", count),
	})
}

type alertRequest struct {
	Alert string `json:"alert" binding:"required"`
}

func (s *Server) handleAlert(c *gin.Context) {
	var req alertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := s.triggerSession(req.Alert, "")
	c.JSON(http.StatusAccepted, gin.H{"session_id": sessionID})
}

func (s *Server) triggerSession(alertMsg string, alertID string) string {
	sessionID := generateID()
	routeDecision := s.router.Route(alertMsg)
	task := agent.Task{
		Alert:           alertMsg,
		SessionID:       sessionID,
		RouteCategory:   routeDecision.Category,
		RouteSource:     routeDecision.Source,
		RouteConfidence: routeDecision.Confidence,
	}

	initialState := &agent.ContextState{Task: task}
	initialState.AddMessage(llm.Message{Role: "user", Content: alertMsg})
	s.mu.Lock()
	s.sessions[sessionID] = initialState
	if alertID != "" {
		s.sessionToAlert[sessionID] = alertID
		s.alerter.MarkProcessing(alertID)
	}
	s.mu.Unlock()

	log.Printf("[Session] %s created, alert=%q, route=%s (%.2f)", sessionID, alertMsg, routeDecision.Category, routeDecision.Confidence)

	go func() {
		start := time.Now()
		runCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		state, err := s.orchestrator.Run(runCtx, initialState)
		duration := time.Since(start).Milliseconds()

		s.mu.Lock()
		if err != nil {
			log.Printf("[Session] %s FAILED: %v", sessionID, err)
			s.sessions[sessionID].Report = "执行失败: " + err.Error()
			if alertID != "" {
				s.alerter.MarkFailed(alertID, err.Error())
			}
		} else {
			s.sessions[sessionID] = state
			if alertID != "" {
				if s.cfg.AgentDryRun {
					s.alerter.MarkProcessed(alertID, duration)
				} else {
					s.alerter.MarkResolved(alertID, duration)
				}
			}
			log.Printf("[Session] %s completed in %dms", sessionID, duration)
		}
		s.mu.Unlock()
	}()

	return sessionID
}

func (s *Server) getSession(c *gin.Context) {
	id := c.Param("id")
	s.mu.RLock()
	state, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, state)
}
