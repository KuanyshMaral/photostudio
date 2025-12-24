package catalog

import (
	"context"

	"photostudio/internal/domain"
	"photostudio/internal/repository"
)

type Service struct {
	repo *repository.StudioRepository
}

func NewService(repo *repository.StudioRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetStudios(
	ctx context.Context,
	f repository.StudioFilters,
) ([]domain.Studio, int64, error) {
	return s.repo.GetAll(ctx, f)
}

func (s *Service) GetStudioByID(
	ctx context.Context,
	id int64,
) (*domain.Studio, error) {
	return s.repo.GetByID(ctx, id)
}
