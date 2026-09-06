package router

import (
	"log"
)

// HybridRouter combines ONNX ML routing with rule-based fallback.
//
// Strategy:
// 1. Try ONNX model first (if available)
// 2. If ONNX confidence >= threshold, use ONNX result
// 3. If ONNX confidence < threshold or ONNX is unavailable, fall back to rule-based
// 4. If rule-based returns "unknown", use ONNX result anyway (even low confidence)
type HybridRouter struct {
	onnx       *ONNXRouter
	rule       RuleRouter
	threshold  float64
	onnxAvail  bool
}

// NewHybridRouter creates a hybrid router.
// modelPath: path to the .onnx file
// vocabPath: path to vocab.json
// threshold: minimum confidence to use ONNX result (0.0 - 1.0)
func NewHybridRouter(modelPath, vocabPath string, threshold float64) *HybridRouter {
	hr := &HybridRouter{
		rule:      RuleRouter{},
		threshold: threshold,
	}

	onnx, err := NewONNXRouter(modelPath, vocabPath)
	if err != nil {
		log.Printf("[Router] ONNX not available: %v (using rule-based fallback)", err)
		hr.onnxAvail = false
	} else {
		hr.onnx = onnx
		hr.onnxAvail = onnx.Available()
		if hr.onnxAvail {
			log.Printf("[Router] ONNX model loaded successfully from %s", modelPath)
		}
	}

	return hr
}

// Route returns the routing decision.
func (h *HybridRouter) Route(alert string) RouteDecision {
	// Try ONNX first
	if h.onnxAvail {
		decision, err := h.onnx.Route(alert)
		if err == nil {
			// If confidence is high enough, use ONNX
			if decision.Confidence >= h.threshold {
				return decision
			}
			// Low confidence, try rule-based as cross-check
			ruleDecision := h.rule.Route(alert)
			if ruleDecision.Category != "unknown" {
				// Rule matched, use rule result but note ONNX was consulted
				ruleDecision.Source = "hybrid-rule"
				return ruleDecision
			}
			// No rule matched, use ONNX even with low confidence
			decision.Source = "onnx-lowconf"
			return decision
		}
		log.Printf("[Router] ONNX inference failed: %v, falling back to rule", err)
	}

	// Fall back to rule-based
	decision := h.rule.Route(alert)
	return decision
}

// ONNXAvailable returns whether the ONNX runtime is available.
func (h *HybridRouter) ONNXAvailable() bool {
	return h.onnxAvail
}
