package entry

import (
	"context"

	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/entities"
)

type IService interface {
	CreateEntries(entries []entities.Entry) error
	GetEntries(find string) ([]entities.Entry, error)
	GetEntriesWithIdOrTypeFiler(spec string, find string) ([]entities.Entry, error)
	RemoveEntries(find string) error
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

func (s *Service) GetEntriesWithIdOrTypeFiler(spec string, find string) ([]entities.Entry, error) {
	querySpec, err := s.queryHelperService.ParseFindWithIdOrTypeFiler(spec, find)
	if err != nil {
		return nil, err
	}
	return s.entryRepo.Find(context.Background(), querySpec)
}

func (s *Service) GetEntries(find string) ([]entities.Entry, error) {
	spec, err := s.queryHelperService.ParseFind(find)
	if err != nil {
		return nil, err
	}
	return s.entryRepo.Find(context.Background(), spec)
}

func (s *Service) RemoveEntries(find string) error {
	spec, err := s.queryHelperService.ParseFind(find)
	if err != nil {
		return err
	}
	return s.entryRepo.Delete(context.Background(), spec)
}
