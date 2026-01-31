package entry

import (
	"context"

	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/entities"
)

type IService interface {
	Create(items []entities.Entry) error
	Search(spec *common.QuerySpec) ([]*entities.Entry, error)
	GetOne(spec *common.QuerySpec) (*entities.Entry, error)
	Remove(spec *common.QuerySpec) error
}

type Service struct {
	entryRepo IRepository
}

func NewService(entryRepo IRepository) *Service {
	return &Service{entryRepo: entryRepo}
}

func (s *Service) Create(items []entities.Entry) error {
	return s.entryRepo.Insert(items)
}

func (s *Service) GetOne(spec *common.QuerySpec) (*entities.Entry, error) {
	if spec == nil {
		spec = &common.QuerySpec{}
	}
	return s.entryRepo.GetOne(context.Background(), spec)
}

func (s *Service) Search(qs *common.QuerySpec) ([]entities.Entry, error) {
	if qs == nil {
		qs = &common.QuerySpec{}
	}
	return s.entryRepo.Find(context.Background(), qs)
}

func (s *Service) Remove(spec *common.QuerySpec) error {
	if spec == nil {
		spec = &common.QuerySpec{}
	}
	return s.entryRepo.Delete(context.Background(), spec)
}
