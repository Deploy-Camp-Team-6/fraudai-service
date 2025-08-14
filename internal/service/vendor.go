package service

import (
	"context"
	"sort"

	"github.com/jules-labs/go-api-prod-template/internal/clients"
	"github.com/rs/zerolog"
)

type VendorService interface {
	Ping(ctx context.Context) (string, error)
	ListModels(ctx context.Context) ([]Model, error)
	Predict(ctx context.Context, req PredictRequest) (PredictResponse, error)
}

type vendorService struct {
	client *clients.ThirdPartyClient
	logger zerolog.Logger
}

func NewVendorService(client *clients.ThirdPartyClient, logger zerolog.Logger) VendorService {
	return &vendorService{
		client: client,
		logger: logger,
	}
}

func (s *vendorService) Ping(ctx context.Context) (string, error) {
	return s.client.Ping(ctx)
}

// Model represents a single model returned by the vendor.
type Model struct {
	ModelType       string   `json:"model_type"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Stage           string   `json:"stage"`
	RunID           string   `json:"run_id"`
	SignatureInputs []string `json:"signature_inputs"`
}

// PredictRequest contains the input for a prediction.
type PredictRequest struct {
	Model    string                 `json:"model" validate:"required"`
	Features map[string]interface{} `json:"features" validate:"required"`
}

// PredictResponse is the subset of the vendor response we expose to clients.
type PredictResponse struct {
	Meta struct {
		ModelName string  `json:"model_name"`
		RunID     string  `json:"run_id"`
		RequestID string  `json:"request_id"`
		Timestamp string  `json:"timestamp"`
		LatencyMs float64 `json:"latency_ms"`
	} `json:"meta"`
	Result struct {
		Prediction int     `json:"prediction"`
		Score      float64 `json:"score"`
		Threshold  float64 `json:"threshold"`
	} `json:"result"`
}

func (s *vendorService) ListModels(ctx context.Context) ([]Model, error) {
	version, err := s.client.Version(ctx)
	if err != nil {
		return nil, err
	}

	models := make([]Model, 0, len(version.LoadedModels))
	for modelType, m := range version.LoadedModels {
		models = append(models, Model{
			ModelType:       modelType,
			Name:            m.Name,
			Version:         m.Version,
			Stage:           m.Stage,
			RunID:           m.RunID,
			SignatureInputs: m.SignatureInputs,
		})
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].ModelType < models[j].ModelType
	})

	return models, nil
}

// Predict calls the vendor predict endpoint and maps the response.
func (s *vendorService) Predict(ctx context.Context, req PredictRequest) (PredictResponse, error) {
	s.logger.Info().Str("model", req.Model).Msg("predict request")

	vendorReq := clients.PredictRequest{
		Model:    req.Model,
		Features: req.Features,
	}

	vendorResp, err := s.client.Predict(ctx, vendorReq)
	if err != nil {
		s.logger.Error().Err(err).Msg("predict request failed")
		return PredictResponse{}, err
	}

	var resp PredictResponse
	resp.Meta.ModelName = vendorResp.Meta.ModelName
	resp.Meta.RunID = vendorResp.Meta.RunID
	resp.Meta.RequestID = vendorResp.Meta.RequestID
	resp.Meta.Timestamp = vendorResp.Meta.Timestamp
	resp.Meta.LatencyMs = vendorResp.Meta.LatencyMs
	resp.Result.Prediction = vendorResp.Result.Prediction
	resp.Result.Score = vendorResp.Result.Score
	resp.Result.Threshold = vendorResp.Result.Threshold

	s.logger.Info().
		Str("model", resp.Meta.ModelName).
		Int("prediction", resp.Result.Prediction).
		Float64("score", resp.Result.Score).
		Msg("predict response")

	return resp, nil
}
