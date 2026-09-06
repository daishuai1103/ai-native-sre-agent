//go:build !onnx
// +build !onnx

package router

import "fmt"

// ONNXRouter is a stub that returns an error when ONNX support is not compiled in.
// Use `go build -tags onnx` to enable ONNX runtime support (requires onnxruntime-go).
type ONNXRouter struct {
	modelPath string
	info      *ModelInfo
}

func NewONNXRouter(modelPath, vocabPath string) (*ONNXRouter, error) {
	info, err := loadModelInfo(vocabPath)
	if err != nil {
		return nil, err
	}
	return &ONNXRouter{modelPath: modelPath, info: info}, nil
}

func (r *ONNXRouter) Route(alert string) (RouteDecision, error) {
	return RouteDecision{}, fmt.Errorf("ONNX runtime not available: build with -tags onnx to enable")
}

func (r *ONNXRouter) Available() bool {
	return false
}
