package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/glycoview/glycoview/internal/model"
	"github.com/glycoview/glycoview/internal/store"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS documents (
  collection TEXT NOT NULL,
  identifier TEXT NOT NULL,
  data JSONB NOT NULL,
  srv_created BIGINT NOT NULL,
  srv_modified BIGINT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  is_valid BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at BIGINT,
  PRIMARY KEY (collection, identifier)
);
CREATE INDEX IF NOT EXISTS documents_collection_modified_idx ON documents (collection, srv_modified DESC);

CREATE TABLE IF NOT EXISTS entries (
  identifier TEXT PRIMARY KEY,
  type TEXT,
  date BIGINT,
  date_string TEXT,
  utc_offset INT,
  device TEXT,
  payload JSONB NOT NULL,
  srv_created BIGINT NOT NULL,
  srv_modified BIGINT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  is_valid BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at BIGINT
);
CREATE TABLE IF NOT EXISTS treatments (
  identifier TEXT PRIMARY KEY,
  event_type TEXT,
  created_at TEXT,
  date BIGINT,
  utc_offset INT,
  device TEXT,
  payload JSONB NOT NULL,
  srv_created BIGINT NOT NULL,
  srv_modified BIGINT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  is_valid BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at BIGINT
);
CREATE TABLE IF NOT EXISTS devicestatus (
  identifier TEXT PRIMARY KEY,
  created_at TEXT,
  date BIGINT,
  utc_offset INT,
  device TEXT,
  payload JSONB NOT NULL,
  srv_created BIGINT NOT NULL,
  srv_modified BIGINT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  is_valid BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at BIGINT
);
CREATE TABLE IF NOT EXISTS profile (
  identifier TEXT PRIMARY KEY,
  created_at TEXT,
  default_profile TEXT,
  start_date TEXT,
  payload JSONB NOT NULL,
  srv_created BIGINT NOT NULL,
  srv_modified BIGINT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  is_valid BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at BIGINT
);
CREATE TABLE IF NOT EXISTS food (
  identifier TEXT PRIMARY KEY,
  created_at TEXT,
  name TEXT,
  category TEXT,
  payload JSONB NOT NULL,
  srv_created BIGINT NOT NULL,
  srv_modified BIGINT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  is_valid BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at BIGINT
);
CREATE TABLE IF NOT EXISTS settings (
  identifier TEXT PRIMARY KEY,
  key TEXT,
  payload JSONB NOT NULL,
  srv_created BIGINT NOT NULL,
  srv_modified BIGINT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  is_valid BOOLEAN NOT NULL DEFAULT TRUE,
  deleted_at BIGINT
);
`

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// Pool returns the underlying pgxpool. Other packages that want to share the
// same connection pool (e.g. internal/goals) can call this instead of opening
// a second pool against the same database.
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

func (s *Store) Create(ctx context.Context, collection string, data map[string]any, subject string) (model.Record, bool, error) {
	clean, err := store.NormalizeData(collection, data)
	if err != nil {
		return model.Record{}, false, err
	}
	rows, err := s.loadCollection(ctx, collection)
	if err != nil {
		return model.Record{}, false, err
	}
	for _, existing := range rows {
		if existing.Identifier() == clean["identifier"] || store.DedupeKey(collection, existing.Data) == store.DedupeKey(collection, clean) {
			oldIdentifier := existing.Identifier()
			existing.Data = model.Merge(existing.Data, clean)
			updatedID := existing.Identifier()
			if updatedID == "" {
				updatedID = oldIdentifier
			}
			existing.ID = updatedID
			existing.Data["identifier"] = updatedID
			existing.Data["_id"] = updatedID
			existing.SrvModified = time.Now().UnixMilli()
			existing.Subject = subject
			existing.IsValid = true
			existing.DeletedAt = nil
			if err := s.save(ctx, existing); err != nil {
				return model.Record{}, false, err
			}
			if oldIdentifier != updatedID {
				if err := s.deleteByID(ctx, collection, oldIdentifier); err != nil {
					return model.Record{}, false, err
				}
			}
			return existing, false, nil
		}
	}
	identifier, _ := model.StringField(clean, "identifier")
	if identifier == "" {
		identifier = store.GenerateIdentifier()
	}
	clean["identifier"] = identifier
	clean["_id"] = identifier
	now := time.Now().UnixMilli()
	record := model.Record{
		Collection:  model.NormalizeCollection(collection),
		ID:          identifier,
		Data:        clean,
		SrvCreated:  now,
		SrvModified: now,
		Subject:     subject,
		IsValid:     true,
	}
	if err := s.save(ctx, record); err != nil {
		return model.Record{}, false, err
	}
	return record, true, nil
}

func (s *Store) Get(ctx context.Context, collection, identifier string) (model.Record, error) {
	record, err := s.loadOne(ctx, collection, identifier)
	if err != nil {
		return model.Record{}, err
	}
	if !record.IsValid {
		return model.Record{}, store.ErrGone
	}
	return record, nil
}

func (s *Store) Search(ctx context.Context, collection string, query store.Query) ([]model.Record, error) {
	records, err := s.loadCollection(ctx, collection)
	if err != nil {
		return nil, err
	}
	return store.ApplyQuery(records, query), nil
}

func (s *Store) Replace(ctx context.Context, collection, identifier string, data map[string]any, subject string) (model.Record, bool, error) {
	clean, err := store.NormalizeData(collection, data)
	if err != nil {
		return model.Record{}, false, err
	}
	now := time.Now().UnixMilli()
	record, err := s.loadOne(ctx, collection, identifier)
	created := false
	if err == store.ErrNotFound {
		record = model.Record{
			Collection: model.NormalizeCollection(collection),
			ID:         identifier,
			SrvCreated: now,
			IsValid:    true,
		}
		created = true
	} else if err != nil && err != store.ErrGone {
		return model.Record{}, false, err
	}
	clean["identifier"] = identifier
	clean["_id"] = identifier
	record.Data = clean
	record.Subject = subject
	record.SrvModified = now
	record.IsValid = true
	record.DeletedAt = nil
	if record.SrvCreated == 0 {
		record.SrvCreated = now
	}
	if err := s.save(ctx, record); err != nil {
		return model.Record{}, false, err
	}
	return record, created, nil
}

func (s *Store) Patch(ctx context.Context, collection, identifier string, patch map[string]any, subject string) (model.Record, error) {
	record, err := s.Get(ctx, collection, identifier)
	if err != nil {
		return model.Record{}, err
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
	if err := s.save(ctx, record); err != nil {
		return model.Record{}, err
	}
	return record, nil
}

func (s *Store) Delete(ctx context.Context, collection, identifier string, permanent bool, subject string) error {
	record, err := s.loadOne(ctx, collection, identifier)
	if err != nil {
		return err
	}
	if permanent {
		_, err = s.pool.Exec(ctx, `DELETE FROM documents WHERE collection=$1 AND identifier=$2`, model.NormalizeCollection(collection), identifier)
		if err != nil {
			return err
		}
		return s.deleteMirror(ctx, collection, identifier)
	}
	now := time.Now().UnixMilli()
	record.IsValid = false
	record.DeletedAt = &now
	record.Subject = subject
	record.SrvModified = now
	return s.save(ctx, record)
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

func (s *Store) History(ctx context.Context, collection string, since int64, limit int) ([]model.Record, error) {
	records, err := s.loadCollection(ctx, collection)
	if err != nil {
		return nil, err
	}
	filtered := make([]model.Record, 0, len(records))
	for _, record := range records {
		if record.SrvModified > since {
			filtered = append(filtered, record)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].SrvModified < filtered[j].SrvModified
	})
	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func (s *Store) LastModified(ctx context.Context, collections []string) (map[string]int64, error) {
	result := make(map[string]int64, len(collections))
	for _, collection := range collections {
		rows, err := s.loadCollection(ctx, collection)
		if err != nil {
			return nil, err
		}
		var max int64
		for _, record := range rows {
			if record.SrvModified > max {
				max = record.SrvModified
			}
			if date, ok := model.Int64Field(record.Data, "date"); ok && date > max {
				max = date
			}
		}
		if max > 0 {
			result[model.NormalizeCollection(collection)] = max
		}
	}
	return result, nil
}

func (s *Store) save(ctx context.Context, record model.Record) error {
	payload, err := json.Marshal(record.Data)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO documents (collection, identifier, data, srv_created, srv_modified, subject, is_valid, deleted_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (collection, identifier)
DO UPDATE SET data=EXCLUDED.data, srv_modified=EXCLUDED.srv_modified, subject=EXCLUDED.subject, is_valid=EXCLUDED.is_valid, deleted_at=EXCLUDED.deleted_at
`, record.Collection, record.Identifier(), payload, record.SrvCreated, record.SrvModified, record.Subject, record.IsValid, record.DeletedAt)
	if err != nil {
		return err
	}
	return s.saveMirror(ctx, record, payload)
}

func (s *Store) loadCollection(ctx context.Context, collection string) ([]model.Record, error) {
	rows, err := s.pool.Query(ctx, `SELECT identifier, data, srv_created, srv_modified, subject, is_valid, deleted_at FROM documents WHERE collection=$1`, model.NormalizeCollection(collection))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []model.Record
	for rows.Next() {
		record, err := scanRecord(model.NormalizeCollection(collection), rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) loadOne(ctx context.Context, collection, identifier string) (model.Record, error) {
	row := s.pool.QueryRow(ctx, `SELECT identifier, data, srv_created, srv_modified, subject, is_valid, deleted_at FROM documents WHERE collection=$1 AND identifier=$2`, model.NormalizeCollection(collection), identifier)
	return scanRecord(model.NormalizeCollection(collection), row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRecord(collection string, row scanner) (model.Record, error) {
	var identifier string
	var payload []byte
	var srvCreated, srvModified int64
	var subject string
	var isValid bool
	var deletedAt *int64
	if err := row.Scan(&identifier, &payload, &srvCreated, &srvModified, &subject, &isValid, &deletedAt); err != nil {
		if err.Error() == "no rows in result set" {
			return model.Record{}, store.ErrNotFound
		}
		return model.Record{}, fmt.Errorf("scan record: %w", err)
	}
	data := map[string]any{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return model.Record{}, err
	}
	return model.Record{
		Collection:  collection,
		ID:          identifier,
		Data:        data,
		SrvCreated:  srvCreated,
		SrvModified: srvModified,
		Subject:     subject,
		IsValid:     isValid,
		DeletedAt:   deletedAt,
	}, nil
}

func DatabaseURL() string {
	return os.Getenv("DATABASE_URL")
}

func (s *Store) saveMirror(ctx context.Context, record model.Record, payload []byte) error {
	table := model.NormalizeCollection(record.Collection)
	switch table {
	case "entries":
		date, _ := model.Int64Field(record.Data, "date")
		dateString, _ := model.StringField(record.Data, "dateString")
		kind, _ := model.StringField(record.Data, "type")
		device, _ := model.StringField(record.Data, "device")
		offset, _ := model.Int64Field(record.Data, "utcOffset")
		_, err := s.pool.Exec(ctx, `
INSERT INTO entries (identifier, type, date, date_string, utc_offset, device, payload, srv_created, srv_modified, subject, is_valid, deleted_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (identifier) DO UPDATE SET type=EXCLUDED.type, date=EXCLUDED.date, date_string=EXCLUDED.date_string, utc_offset=EXCLUDED.utc_offset, device=EXCLUDED.device, payload=EXCLUDED.payload, srv_modified=EXCLUDED.srv_modified, subject=EXCLUDED.subject, is_valid=EXCLUDED.is_valid, deleted_at=EXCLUDED.deleted_at
`, record.Identifier(), kind, date, dateString, int(offset), device, payload, record.SrvCreated, record.SrvModified, record.Subject, record.IsValid, record.DeletedAt)
		return err
	case "treatments":
		date, _ := model.Int64Field(record.Data, "date")
		createdAt, _ := model.StringField(record.Data, "created_at")
		eventType, _ := model.StringField(record.Data, "eventType")
		device, _ := model.StringField(record.Data, "device")
		offset, _ := model.Int64Field(record.Data, "utcOffset")
		_, err := s.pool.Exec(ctx, `
INSERT INTO treatments (identifier, event_type, created_at, date, utc_offset, device, payload, srv_created, srv_modified, subject, is_valid, deleted_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (identifier) DO UPDATE SET event_type=EXCLUDED.event_type, created_at=EXCLUDED.created_at, date=EXCLUDED.date, utc_offset=EXCLUDED.utc_offset, device=EXCLUDED.device, payload=EXCLUDED.payload, srv_modified=EXCLUDED.srv_modified, subject=EXCLUDED.subject, is_valid=EXCLUDED.is_valid, deleted_at=EXCLUDED.deleted_at
`, record.Identifier(), eventType, createdAt, date, int(offset), device, payload, record.SrvCreated, record.SrvModified, record.Subject, record.IsValid, record.DeletedAt)
		return err
	case "devicestatus":
		date, _ := model.Int64Field(record.Data, "date")
		createdAt, _ := model.StringField(record.Data, "created_at")
		device, _ := model.StringField(record.Data, "device")
		offset, _ := model.Int64Field(record.Data, "utcOffset")
		_, err := s.pool.Exec(ctx, `
INSERT INTO devicestatus (identifier, created_at, date, utc_offset, device, payload, srv_created, srv_modified, subject, is_valid, deleted_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (identifier) DO UPDATE SET created_at=EXCLUDED.created_at, date=EXCLUDED.date, utc_offset=EXCLUDED.utc_offset, device=EXCLUDED.device, payload=EXCLUDED.payload, srv_modified=EXCLUDED.srv_modified, subject=EXCLUDED.subject, is_valid=EXCLUDED.is_valid, deleted_at=EXCLUDED.deleted_at
`, record.Identifier(), createdAt, date, int(offset), device, payload, record.SrvCreated, record.SrvModified, record.Subject, record.IsValid, record.DeletedAt)
		return err
	case "profile":
		createdAt, _ := model.StringField(record.Data, "created_at")
		defaultProfile, _ := model.StringField(record.Data, "defaultProfile")
		startDate, _ := model.StringField(record.Data, "startDate")
		_, err := s.pool.Exec(ctx, `
INSERT INTO profile (identifier, created_at, default_profile, start_date, payload, srv_created, srv_modified, subject, is_valid, deleted_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (identifier) DO UPDATE SET created_at=EXCLUDED.created_at, default_profile=EXCLUDED.default_profile, start_date=EXCLUDED.start_date, payload=EXCLUDED.payload, srv_modified=EXCLUDED.srv_modified, subject=EXCLUDED.subject, is_valid=EXCLUDED.is_valid, deleted_at=EXCLUDED.deleted_at
`, record.Identifier(), createdAt, defaultProfile, startDate, payload, record.SrvCreated, record.SrvModified, record.Subject, record.IsValid, record.DeletedAt)
		return err
	case "food":
		createdAt, _ := model.StringField(record.Data, "created_at")
		name, _ := model.StringField(record.Data, "name")
		category, _ := model.StringField(record.Data, "category")
		_, err := s.pool.Exec(ctx, `
INSERT INTO food (identifier, created_at, name, category, payload, srv_created, srv_modified, subject, is_valid, deleted_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (identifier) DO UPDATE SET created_at=EXCLUDED.created_at, name=EXCLUDED.name, category=EXCLUDED.category, payload=EXCLUDED.payload, srv_modified=EXCLUDED.srv_modified, subject=EXCLUDED.subject, is_valid=EXCLUDED.is_valid, deleted_at=EXCLUDED.deleted_at
`, record.Identifier(), createdAt, name, category, payload, record.SrvCreated, record.SrvModified, record.Subject, record.IsValid, record.DeletedAt)
		return err
	case "settings":
		key, _ := model.StringField(record.Data, "key")
		_, err := s.pool.Exec(ctx, `
INSERT INTO settings (identifier, key, payload, srv_created, srv_modified, subject, is_valid, deleted_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (identifier) DO UPDATE SET key=EXCLUDED.key, payload=EXCLUDED.payload, srv_modified=EXCLUDED.srv_modified, subject=EXCLUDED.subject, is_valid=EXCLUDED.is_valid, deleted_at=EXCLUDED.deleted_at
`, record.Identifier(), key, payload, record.SrvCreated, record.SrvModified, record.Subject, record.IsValid, record.DeletedAt)
		return err
	default:
		return nil
	}
}

func (s *Store) deleteMirror(ctx context.Context, collection, identifier string) error {
	table := model.NormalizeCollection(collection)
	switch table {
	case "entries", "treatments", "devicestatus", "profile", "food", "settings":
		_, err := s.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE identifier=$1`, table), identifier)
		return err
	default:
		return nil
	}
}

func (s *Store) deleteByID(ctx context.Context, collection, identifier string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM documents WHERE collection=$1 AND identifier=$2`, model.NormalizeCollection(collection), identifier)
	if err != nil {
		return err
	}
	return s.deleteMirror(ctx, collection, identifier)
}
