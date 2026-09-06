package router

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ModelInfo holds vocab and categories for feature extraction.
type ModelInfo struct {
	Vocab      []string `json:"vocab"`
	Categories []string `json:"categories"`
}

// loadModelInfo reads the vocab.json file bundled with the binary.
func loadModelInfo(path string) (*ModelInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read vocab: %w", err)
	}
	var info ModelInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse vocab: %w", err)
	}
	return &info, nil
}

// extractFeatures converts text to a float32 feature vector based on vocab.
// Uses simple bag-of-keywords (presence of each keyword).
func extractFeatures(text string, vocab []string) []float32 {
	textLower := strings.ToLower(text)
	features := make([]float32, len(vocab))
	for i, kw := range vocab {
		if strings.Contains(textLower, strings.ToLower(kw)) {
			features[i] = 1.0
		}
	}
	return features
}

// argmax returns the index and value of the maximum element.
func argmax(scores []float32) (int, float32) {
	if len(scores) == 0 {
		return 0, 0
	}
	bestIdx := 0
	bestVal := scores[0]
	for i, v := range scores {
		if v > bestVal {
			bestVal = v
			bestIdx = i
		}
	}
	return bestIdx, bestVal
}

// softmax normalizes scores to probabilities.
func softmax(scores []float32) []float32 {
	if len(scores) == 0 {
		return nil
	}
	// Find max for numerical stability
	maxVal := scores[0]
	for _, v := range scores[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	// Compute exp
	expVals := make([]float32, len(scores))
	sum := float32(0)
	for i, v := range scores {
		e := float32(1) // placeholder, we compute below
		// Use simple exp approximation via math
		_ = e
		expVals[i] = fastExp(v - maxVal)
		sum += expVals[i]
	}
	// Normalize
	for i := range expVals {
		expVals[i] /= sum
	}
	return expVals
}

// fastExp is a simple exp approximation (good enough for softmax of 5 values).
func fastExp(x float32) float32 {
	// Use taylor series for small x, clamp for large x
	if x > 10 {
		return 22026
	}
	if x < -10 {
		return 0.00005
	}
	// e^x ≈ (1 + x/256)^256, but let's just use a simple taylor
	result := float32(1)
	term := float32(1)
	for i := 1; i <= 20; i++ {
		term *= x / float32(i)
		result += term
	}
	return result
}
