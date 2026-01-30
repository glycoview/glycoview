package entry

import (
	"context"
	"net/url"

	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/entities"
)

type IService interface {
	CreateEntries(entries []entities.Entry) error
	GetEntries(values url.Values) ([]entities.Entry, error)
}

type Service struct {
	entryRepo          IRepository
	queryHelperService common.IQueryHelper
}

func NewService(entryRepo IRepository, queryHelperService common.IQueryHelper) *Service {
	return &Service{entryRepo: entryRepo, queryHelperService: queryHelperService}
}

func (s *Service) CreateEntries(entries []entities.Entry) error {
	return s.entryRepo.InsertEntries(entries)
}

func (s *Service) GetEntries(values url.Values) ([]entities.Entry, error) {
	spec, err := s.queryHelperService.ParseFind(values)
	if err != nil {
		return nil, err
	}
	return s.entryRepo.Find(context.Background(), spec)
}
