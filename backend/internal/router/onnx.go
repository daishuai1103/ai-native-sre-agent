//go:build onnx
// +build onnx

package router

import (
	"fmt"

	ort "github.com/microsoft/onnxruntime-go"
)

// ONNXRouter performs ML-based routing using a LightGBM model exported to ONNX.
type ONNXRouter struct {
	session   *ort.Session
	info      *ModelInfo
	modelPath string
}

func NewONNXRouter(modelPath, vocabPath string) (*ONNXRouter, error) {
	info, err := loadModelInfo(vocabPath)
	if err != nil {
		return nil, err
	}

	env := ort.NewEnvironment(ort.LogLevelWarning, "router-env")
	so := ort.NewSessionOptions()
	session, err := ort.NewSession(env, modelPath, so)
	if err != nil {
		env.Delete()
		return nil, fmt.Errorf("load onnx model: %w", err)
	}

	return &ONNXRouter{
		session:   session,
		info:      info,
		modelPath: modelPath,
	}, nil
}

func (r *ONNXRouter) Available() bool {
	return r.session != nil
}

func (r *ONNXRouter) Route(alert string) (RouteDecision, error) {
	if r.session == nil {
		return RouteDecision{}, fmt.Errorf("onnx session not initialized")
	}

	features := extractFeatures(alert, r.info.Vocab)

	// Create input tensor
	inputName := r.session.InputNames[0]
	shape := []int64{1, int64(len(r.info.Vocab))}
	inputTensor := ort.NewTensor(shape, features)

	// Run inference
	outputs, err := r.session.Run([]string{inputName}, []ort.Value{inputTensor}, r.session.OutputNames)
	if err != nil {
		return RouteDecision{}, fmt.Errorf("onnx inference: %w", err)
	}
	defer func() {
		for _, o := range outputs {
			o.Close()
		}
	}()

	// Get output (first output = probabilities or logits)
	output := outputs[0]
	scores := output.GetData().([]float32)
	probs := softmax(scores)
	bestIdx, confidence := argmax(probs)

	if bestIdx >= len(r.info.Categories) {
		return RouteDecision{}, fmt.Errorf("invalid category index %d", bestIdx)
	}

	return RouteDecision{
		Category:   r.info.Categories[bestIdx],
		Source:     "onnx",
		Confidence: float64(confidence),
	}, nil
}
