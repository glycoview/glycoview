package profile

import (
	"context"

	"github.com/better-monitoring/bscout/pkg/common"
	"github.com/better-monitoring/bscout/pkg/entities"
)

type IService interface {
	Create(items []entities.Profile) error
	Search(spec *common.QuerySpec) ([]entities.Profile, error)
	Remove(spec *common.QuerySpec) error
}

type Service struct{ repo IRepository }

func NewService(repo IRepository) *Service { return &Service{repo: repo} }

func (s *Service) Create(items []entities.Profile) error { return s.repo.Insert(items) }

func (s *Service) Search(spec *common.QuerySpec) ([]entities.Profile, error) {
	return s.repo.Find(context.Background(), spec)
}

func (s *Service) Remove(spec *common.QuerySpec) error {
	return s.repo.Delete(context.Background(), spec)
}
