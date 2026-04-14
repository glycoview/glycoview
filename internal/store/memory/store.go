package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/glycoview/glycoview/internal/model"
	"github.com/glycoview/glycoview/internal/store"
)

type Store struct {
	mu          sync.RWMutex
	collections map[string]map[string]model.Record
}

func New() *Store {
	return &Store{collections: map[string]map[string]model.Record{}}
}

func (s *Store) Create(_ context.Context, collection string, data map[string]any, subject string) (model.Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	collection = model.NormalizeCollection(collection)
	clean, err := store.NormalizeData(collection, data)
	if err != nil {
		return model.Record{}, false, err
	}
	coll := s.ensureCollection(collection)
	if existing := s.findDuplicateLocked(collection, coll, clean); existing != nil {
		oldIdentifier := existing.Identifier()
		updated := existing.Clone()
		updated.Data = model.Merge(updated.Data, clean)
		updatedID := updated.Identifier()
		if updatedID == "" {
			updatedID = oldIdentifier
		}
		updated.ID = updatedID
		updated.Data["identifier"] = updatedID
		updated.Data["_id"] = updatedID
		updated.SrvModified = time.Now().UnixMilli()
		updated.Subject = subject
		updated.IsValid = true
		updated.DeletedAt = nil
		if oldIdentifier != updatedID {
			delete(coll, oldIdentifier)
		}
		coll[updatedID] = updated
		return updated.Clone(), false, nil
	}
	identifier, _ := model.StringField(clean, "identifier")
	if identifier == "" {
		identifier = store.GenerateIdentifier()
	}
	clean["identifier"] = identifier
	clean["_id"] = identifier
	now := time.Now().UnixMilli()
	record := model.Record{
		Collection:  collection,
		ID:          identifier,
		Data:        clean,
		SrvCreated:  now,
		SrvModified: now,
		Subject:     subject,
		IsValid:     true,
	}
	coll[identifier] = record
	return record.Clone(), true, nil
}

func (s *Store) Get(_ context.Context, collection, identifier string) (model.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	coll := s.collections[model.NormalizeCollection(collection)]
	record, ok := coll[identifier]
	if !ok {
		return model.Record{}, store.ErrNotFound
	}
	if !record.IsValid {
		return model.Record{}, store.ErrGone
	}
	return record.Clone(), nil
}

func (s *Store) Search(_ context.Context, collection string, query store.Query) ([]model.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	coll := s.collections[model.NormalizeCollection(collection)]
	records := make([]model.Record, 0, len(coll))
	for _, record := range coll {
		records = append(records, record.Clone())
	}
	return store.ApplyQuery(records, query), nil
}

func (s *Store) Replace(_ context.Context, collection, identifier string, data map[string]any, subject string) (model.Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	collection = model.NormalizeCollection(collection)
	clean, err := store.NormalizeData(collection, data)
	if err != nil {
		return model.Record{}, false, err
	}
	coll := s.ensureCollection(collection)
	now := time.Now().UnixMilli()
	record, exists := coll[identifier]
	if !exists {
		clean["identifier"] = identifier
		clean["_id"] = identifier
		record = model.Record{
			Collection:  collection,
			ID:          identifier,
			Data:        clean,
			SrvCreated:  now,
			SrvModified: now,
			Subject:     subject,
			IsValid:     true,
		}
		coll[identifier] = record
		return record.Clone(), true, nil
	}
	record.Data = clean
	record.Data["identifier"] = identifier
	record.Data["_id"] = identifier
	record.SrvModified = now
	record.Subject = subject
	record.IsValid = true
	record.DeletedAt = nil
	coll[identifier] = record
	return record.Clone(), false, nil
}

func (s *Store) Patch(_ context.Context, collection, identifier string, patch map[string]any, subject string) (model.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	collection = model.NormalizeCollection(collection)
	coll := s.collections[collection]
	record, ok := coll[identifier]
	if !ok {
		return model.Record{}, store.ErrNotFound
	}
	if !record.IsValid {
		return model.Record{}, store.ErrGone
	}
	merged := model.Merge(record.Data, patch)
	clean, err := store.NormalizeData(collection, merged)
	if err != nil {
		return model.Record{}, err
	}
	record.Data = clean
	record.Data["identifier"] = identifier
	record.Data["_id"] = identifier
	record.Data["modifiedBy"] = subject
	record.SrvModified = time.Now().UnixMilli()
	coll[identifier] = record
	return record.Clone(), nil
}

func (s *Store) Delete(_ context.Context, collection, identifier string, permanent bool, subject string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	collection = model.NormalizeCollection(collection)
	coll := s.collections[collection]
	record, ok := coll[identifier]
	if !ok {
		return store.ErrNotFound
	}
	if permanent {
		delete(coll, identifier)
		return nil
	}
	now := time.Now().UnixMilli()
	record.IsValid = false
	record.DeletedAt = &now
	record.SrvModified = now
	record.Subject = subject
	coll[identifier] = record
	return nil
}

func (s *Store) DeleteMatching(ctx context.Context, collection string, query store.Query, permanent bool, subject string) (int, error) {
	records, err := s.Search(ctx, collection, query)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, record := range records {
		if err := s.Delete(ctx, collection, record.Identifier(), permanent, subject); err == nil {
			count++
		}
	}
	return count, nil
}

func (s *Store) History(_ context.Context, collection string, since int64, limit int) ([]model.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	coll := s.collections[model.NormalizeCollection(collection)]
	records := make([]model.Record, 0, len(coll))
	for _, record := range coll {
		if record.SrvModified > since {
			records = append(records, record.Clone())
		}
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].SrvModified < records[j].SrvModified
	})
	if limit > 0 && limit < len(records) {
		records = records[:limit]
	}
	return records, nil
}

func (s *Store) LastModified(_ context.Context, collections []string) (map[string]int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int64, len(collections))
	for _, name := range collections {
		var max int64
		for _, record := range s.collections[model.NormalizeCollection(name)] {
			if record.SrvModified > max {
				max = record.SrvModified
			}
			if date, ok := model.Int64Field(record.Data, "date"); ok && date > max {
				max = date
			}
		}
		if max > 0 {
			result[model.NormalizeCollection(name)] = max
		}
	}
	return result, nil
}

func (s *Store) ensureCollection(name string) map[string]model.Record {
	coll := s.collections[name]
	if coll == nil {
		coll = map[string]model.Record{}
		s.collections[name] = coll
	}
	return coll
}

func (s *Store) findDuplicateLocked(collection string, coll map[string]model.Record, data map[string]any) *model.Record {
	if identifier, ok := model.StringField(data, "identifier"); ok && identifier != "" {
		if record, exists := coll[identifier]; exists {
			clone := record.Clone()
			return &clone
		}
	}
	dedupeKey := store.DedupeKey(collection, data)
	if dedupeKey == "" {
		return nil
	}
	for _, record := range coll {
		if store.DedupeKey(collection, record.Data) == dedupeKey {
			clone := record.Clone()
			return &clone
		}
	}
	return nil
}

func (s *Store) Seed(collection string, docs ...map[string]any) error {
	for _, doc := range docs {
		if _, _, err := s.Create(context.Background(), collection, doc, "seed"); err != nil {
			return fmt.Errorf("seed %s: %w", collection, err)
		}
	}
	return nil
}
