package admin

import "photostudio/internal/domain"

type Service struct {
	// deps ServiceDeps
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetPendingStudios(page, limit int) (StudioListResponse, error) {
	return StudioListResponse{
		Studios: []domain.Studio{},
		Total:   0,
		Page:    page,
		Limit:   limit,
	}, nil
}

func (s *Service) VerifyStudio(studioID int64, adminNotes string) error {
	return nil
}

func (s *Service) RejectStudio(studioID int64, reason string) error {
	return nil
}

func (s *Service) GetStatistics() (StatisticsResponse, error) {
	return StatisticsResponse{}, nil
}
