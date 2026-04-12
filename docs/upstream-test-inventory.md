# Upstream Test Inventory

Source repository: `nightscout/cgm-remote-monitor`

Snapshot in use:

- commit: `9cd304f78a5c12401c9711cbd56d2a12eaca0632`
- commit date: `2026-03-03 14:57:48 -0800`
- source URL: `https://github.com/nightscout/cgm-remote-monitor`

Imported artifacts live under `third_party/nightscout/`.

## Acceptance And Contract Suites

These are the most useful starting point for a Go compatibility port because they exercise the HTTP contract rather than just isolated helper functions.

API v1 and legacy HTTP-oriented suites:

- `tests/api.entries.test.js`
- `tests/api.treatments.test.js`
- `tests/api.status.test.js`
- `tests/api.security.test.js`
- `tests/api.devicestatus.test.js`
- `tests/api.root.test.js`
- `tests/api.verifyauth.test.js`
- `tests/api.unauthorized.test.js`
- `tests/api.alexa.test.js`
- `tests/verifyauth.test.js`
- `tests/security.test.js`
- `tests/notifications-api.test.js`

API v3 suites:

- `tests/api3.basic.test.js`
- `tests/api3.create.test.js`
- `tests/api3.read.test.js`
- `tests/api3.update.test.js`
- `tests/api3.patch.test.js`
- `tests/api3.delete.test.js`
- `tests/api3.search.test.js`
- `tests/api3.security.test.js`
- `tests/api3.generic.workflow.test.js`
- `tests/api3.socket.test.js`
- `tests/api3.renderer.test.js`

Supporting fixtures and helpers:

- `tests/hooks.js`
- `tests/fixtures/api/instance.js`
- `tests/fixtures/api3/instance.js`
- `tests/fixtures/api3/authSubject.js`
- `tests/fixtures/api3/cacheMonitor.js`
- `tests/fixtures/example.json`
- `tests/fixtures/load.js`
- `testing/populate.js`
- `testing/populate_rest.js`

## Pure Logic And Domain Suites

These are good candidates for direct semantic ports into Go unit tests after the HTTP contract is under control.

- `tests/query.test.js`
- `tests/profile.test.js`
- `tests/iob.test.js`
- `tests/openaps.test.js`
- `tests/openaps-storage.test.js`
- `tests/data.calcdelta.test.js`
- `tests/data.treatmenttocurve.test.js`
- `tests/settings.test.js`
- `tests/times.test.js`
- `tests/units.test.js`
- `tests/utils.test.js`
- `tests/language.test.js`

There are many more upstream unit-style suites in the snapshot. The imported `third_party/nightscout/tests/` directory is the complete source set for future triage.

## Recommended Porting Order

1. `api.entries.test.js`
2. `api.treatments.test.js`
3. `api.status.test.js`
4. `api.security.test.js`
5. `api3.create.test.js`
6. `api3.read.test.js`
7. `api3.update.test.js`
8. `api3.delete.test.js`
9. `api3.search.test.js`
10. `api3.generic.workflow.test.js`
11. `query.test.js`
12. `profile.test.js`
13. `iob.test.js`

## Notes From The Snapshot

- The upstream repo currently contains 20 legacy API-oriented test files and 11 API v3-oriented test files.
- `api.entries.test.js` asserts default count behavior, sort order, current entry lookup, slicing, and time-pattern queries.
- `api.treatments.test.js` asserts sanitization, deduplication, date parsing, and delete workflows.
- `api3.generic.workflow.test.js` is the strongest end-to-end CRUD and history contract for the Go port.
- `lib/api3/swagger.yaml` is imported with the snapshot and should be treated as an additional contract reference, not as the only source of truth.
