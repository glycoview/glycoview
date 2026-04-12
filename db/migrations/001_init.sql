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

CREATE INDEX IF NOT EXISTS documents_collection_modified_idx
  ON documents (collection, srv_modified DESC);

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
