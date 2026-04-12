# Nightscout Rebuild Plan

## Goal

Build a Go service backed by Postgres that preserves Nightscout API behavior for existing clients while replacing the legacy Node/Mongo stack with a smaller binary and a clinician-oriented UI.

This plan is anchored to the upstream Nightscout snapshot at commit `9cd304f78a5c12401c9711cbd56d2a12eaca0632`, dated `2026-03-03 14:57:48 -0800`.

## Scope

- Replicate Nightscout API v1 endpoints used by existing uploaders and clients.
- Replicate Nightscout API v3 CRUD, history, auth, and status semantics.
- Preserve Nightscout query behavior closely enough that upstream acceptance tests can be ported and passed.
- Replace MongoDB with Postgres.
- Build a clinician-facing UI instead of cloning the existing caregiver UI verbatim.

## Non-Goals For Phase 1

- Rebuilding every Nightscout plugin before the core API contract is stable.
- Perfect pixel parity with the current frontend.
- Migrating directly from every community customization without a compatibility shim.

## Architecture Direction

- One Go service with modular packages, not a split Go backend plus large JS SPA.
- Postgres as the system of record.
- JSONB retained for unmodeled Nightscout fields so custom uploader payloads do not break immediately.
- HTTP API compatibility prioritized before UI work.
- Server-rendered HTML plus selective client interactivity to keep the deployed artifact small.

## Suggested Backend Shape

- `cmd/bscout`: service entrypoint
- `internal/platform/http`: router, middleware, auth, content negotiation
- `internal/platform/store`: Postgres access via `pgx`
- `internal/nightscout/v1`: legacy `/api/v1` compatibility layer
- `internal/nightscout/v3`: `/api/v3` resources, history, lastModified, auth
- `internal/nightscout/query`: translation from Nightscout query parameters to SQL
- `internal/ui`: clinician-focused pages and components

## Postgres Model

Use explicit tables for the high-value collections and keep a `payload JSONB` column for unknown fields.

- `entries`
- `treatments`
- `profiles`
- `device_status`
- `foods`
- `settings`
- `auth_subjects`
- `auth_tokens`
- `audit_events`

Common server-managed fields should include:

- `identifier`
- `date`
- `created_at`
- `srv_created`
- `srv_modified`
- `is_valid`
- `deleted_at`
- `subject`
- `payload`

## Compatibility Requirements Pulled From Upstream Tests

- Default list limits and sort order for entries and treatments.
- Legacy query DSL such as `find[field][$gte]`.
- Record lookup by current value, id, model, slices, and time-based patterns.
- V1 secret-based auth and authorization behavior.
- V3 JWT auth and collection-scoped permissions.
- V3 conditional reads with `If-Modified-Since`.
- V3 deduplication, soft delete, permanent delete, and history feeds.
- Date parsing, timezone handling, and `utcOffset` validation.
- Sanitization behavior for treatment notes.

## Recommended Delivery Order

1. Snapshot upstream contracts and fixtures into this repo.
2. Port the highest-value acceptance tests first:
   `api.entries`, `api.treatments`, `api.status`, `api.security`, `api3.create`, `api3.read`, `api3.update`, `api3.delete`, `api3.search`, `api3.generic.workflow`.
3. Implement the Postgres schema and migration set.
4. Build the query translation layer that maps Nightscout semantics to SQL.
5. Implement `/api/v1/status.json`, `/api/v1/entries`, `/api/v1/treatments`, `/api/v1/profile`.
6. Implement `/api/v3` generic collection handlers and auth.
7. Add streaming or websocket updates only after read/write correctness is stable.
8. Build the clinician UI on top of the now-stable API and domain model.

## Clinician UI Direction

The current Nightscout UI is optimized for caregivers and hobbyist operators. A clinician-facing UI should instead emphasize:

- fast review of the last 24 hours, 72 hours, and 14 days
- clear treatment timeline with glucose, carbs, insulin, and device state in one view
- event provenance and auditability
- report generation and PDF export
- safer defaults for accessibility, typography, and print layouts
- role-based access with read-only and review workflows

## Risks To Handle Early

- Mongo query behavior does not map cleanly to SQL unless we define a compatibility layer deliberately.
- Nightscout stores a large amount of semi-structured data; dropping unknown fields will break clients.
- API v1 and API v3 do not share exactly the same document semantics.
- Time handling is easy to get wrong, especially for `created_at`, `date`, and `utcOffset`.
- Imported upstream tests are AGPL-3.0 work and should remain clearly attributed.

## Definition Of Done For The Port

- Imported upstream contract tests are tracked in this repo and reproducible from a fixed Nightscout commit.
- Go acceptance tests cover the same endpoint behaviors as the upstream tests that matter for compatibility.
- Postgres-backed handlers pass those Go acceptance tests without MongoDB.
- The clinician UI uses the new Go domain model and does not depend on the legacy Nightscout frontend code.
