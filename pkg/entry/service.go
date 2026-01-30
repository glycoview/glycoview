package entry

import (
	"context"

	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/entities"
)

type IService interface {
	Create(items []entities.Entry) error
	Search(spec *common.QuerySpec) ([]entities.Entry, error)
	SearchWithIdOrTypeFilter(spec string, qs *common.QuerySpec) ([]entities.Entry, error)
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
func (s *Service) SearchWithIdOrTypeFilter(spec string, qs *common.QuerySpec) ([]entities.Entry, error) {
	if qs == nil {
		qs = &common.QuerySpec{}
	}
	qs.Filters = append(qs.Filters, common.Filter{Field: "type", Op: common.OpEq, Value: spec})
	qs.Filters = append(qs.Filters, common.Filter{Field: "_id", Op: common.OpEq, Value: spec})
	return s.entryRepo.Find(context.Background(), qs)
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
