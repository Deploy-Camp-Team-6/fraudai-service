package service

import (
	"context"

	"github.com/jules-labs/go-api-prod-template/internal/clients"
)

type VendorService interface {
	Ping(ctx context.Context) (string, error)
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
