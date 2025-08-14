package service

import (
	"context"
	"sort"

	"github.com/jules-labs/go-api-prod-template/internal/clients"
)

type VendorService interface {
	Ping(ctx context.Context) (string, error)
	ListModels(ctx context.Context) ([]Model, error)
}

type vendorService struct {
	client *clients.ThirdPartyClient
}

func NewVendorService(client *clients.ThirdPartyClient) VendorService {
	return &vendorService{
		client: client,
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
