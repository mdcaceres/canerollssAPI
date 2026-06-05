package usecase

import (
	"canerollss/domain"
	"canerollss/ports/output"
)

type ReportService struct {
	repo output.MonthlyClosingRepository
}

func NewReportService(repo output.MonthlyClosingRepository) *ReportService {
	return &ReportService{repo: repo}
}

func (s *ReportService) GetReportHistory(page, pageSize int) ([]domain.MonthlyClosing, int64, error) {
	return s.repo.GetHistory(page, pageSize)
}
