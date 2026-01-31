package treatment

import (
	"context"

	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/entities"
)

type IService interface {
	Create(items []entities.Treatment) error
	Search(spec *common.QuerySpec) ([]entities.Treatment, error)
	GetOne(spec *common.QuerySpec) (*entities.Treatment, error)
	Remove(spec *common.QuerySpec) error
}

type Service struct {
	treatmentRepo IRepository
}

func NewService(treatmentRepo IRepository) *Service {
	return &Service{treatmentRepo: treatmentRepo}
}

func (s *Service) Create(items []entities.Treatment) error {
	return s.treatmentRepo.Insert(items)
}

func (s *Service) GetOne(spec *common.QuerySpec) (*entities.Treatment, error) {
	if spec == nil {
		spec = &common.QuerySpec{}
	}
	return s.treatmentRepo.GetOne(context.Background(), spec)
}

func (s *Service) Search(qs *common.QuerySpec) ([]entities.Treatment, error) {
	if qs == nil {
		qs = &common.QuerySpec{}
	}
	return s.treatmentRepo.Find(context.Background(), qs)
}

func (s *Service) Remove(spec *common.QuerySpec) error {
	if spec == nil {
		spec = &common.QuerySpec{}
	}
	return s.treatmentRepo.Delete(context.Background(), spec)
}
