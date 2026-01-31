package food

import (
	"context"

	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/entities"
)

type IService interface {
	Create(items []entities.Food) error
	Search(spec *common.QuerySpec) ([]entities.Food, error)
	GetOne(spec *common.QuerySpec) (*entities.Food, error)
	Remove(spec *common.QuerySpec) error
}

type Service struct {
	foodRepo IRepository
}

func NewService(foodRepo IRepository) *Service {
	return &Service{foodRepo: foodRepo}
}

func (s *Service) Create(items []entities.Food) error {
	return s.foodRepo.Insert(items)
}

func (s *Service) GetOne(spec *common.QuerySpec) (*entities.Food, error) {
	if spec == nil {
		spec = &common.QuerySpec{}
	}
	return s.foodRepo.GetOne(context.Background(), spec)
}

func (s *Service) Search(qs *common.QuerySpec) ([]entities.Food, error) {
	if qs == nil {
		qs = &common.QuerySpec{}
	}
	return s.foodRepo.Find(context.Background(), qs)
}

func (s *Service) Remove(spec *common.QuerySpec) error {
	if spec == nil {
		spec = &common.QuerySpec{}
	}
	return s.foodRepo.Delete(context.Background(), spec)
}
