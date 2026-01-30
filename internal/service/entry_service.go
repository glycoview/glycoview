package service

import (
	"github.com/better-monitoring/bscout/internal/model"
	"github.com/better-monitoring/bscout/internal/repository"
)

type IEntryService interface {
	CreateEntry(entry model.Entry) error
}

type EntryService struct {
	entryRepo repository.IEntryRepository
}

func NewEntryService(entryRepo repository.IEntryRepository) *EntryService {
	return &EntryService{entryRepo: entryRepo}
}

func (s *EntryService) CreateEntry(entry model.Entry) error {
	return s.entryRepo.InsertEntry(entry)
}
