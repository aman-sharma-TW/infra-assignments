package service

import (
	"context"

	"github.com/amansharma/config-service/internal/model"
	"github.com/amansharma/config-service/internal/repository"
)

type ConfigService struct {
	repo *repository.ConfigRepository
}

func NewConfigService(repo *repository.ConfigRepository) *ConfigService {
	return &ConfigService{repo: repo}
}

func (s *ConfigService) GetConfig(ctx context.Context, id string) (*model.Config, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ConfigService) UpsertConfig(ctx context.Context, req model.UpsertConfigRequest) (*model.Config, bool, error) {
	return s.repo.Upsert(ctx, req)
}

func (s *ConfigService) HealthCheck(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
