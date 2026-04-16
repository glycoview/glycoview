# Appliance Implementation Plan

## Phase 1: Build and Release

- Build `glycoview` multi-arch images for `linux/amd64` and `linux/arm64`
- Build `glycoview-agent` multi-arch images
- Publish both to GHCR only, not Docker Hub
- Produce release metadata with immutable digests
- Publish Raspberry Pi image artifacts to GitHub Releases

Target image names:

- `ghcr.io/glycoview/glycoview`
- `ghcr.io/glycoview/glycoview-agent`

For PAT-free publishing, the repository should live under the `glycoview` owner/org so GitHub Actions can publish with `GITHUB_TOKEN`.

## Phase 2: Runtime Packaging

- Add production Dockerfiles
- Add Compose appliance manifests
- Add env/config templates
- Add first-boot bootstrap script
- Add systemd unit for appliance bootstrap

## Phase 3: Agent Foundation

Implement `glycoview-agent` APIs:

- `GET /healthz`
- `GET /v1/system/status`
- `GET /v1/update/check`
- `POST /v1/update/apply`
- `POST /v1/update/rollback`
- `GET /v1/tls/providers`
- `POST /v1/tls/configure`

Initial version may return `501` for update and TLS write flows while the compose/build plumbing lands.

Status:

- implemented
- protected by shared `GLYCOVIEW_AGENT_TOKEN`
- appliance state encryption available via `GLYCOVIEW_AGENT_STATE_KEY` or the shared agent token

## Phase 4: Dashboard Integration

Add admin-only dashboard pages:

- Appliance status
- TLS setup
- Update center
- Backup status

## Phase 5: TLS Providers

Start with:

- Cloudflare
- Route53
- Hetzner

Each provider needs:

- form schema
- server-side validation
- secret storage
- Traefik env/secret mapping
- provisioning status feedback

## Phase 6: In-Place Updates

Use Compose-driven appliance updates:

- pull updated images
- recreate services with `docker compose up -d`
- use service healthchecks to detect startup failures
- keep manual rollback to the previous tag

Dashboard flow:

1. check available update
2. display release notes
3. click update
4. poll rollout status
5. display success or rollback state

## Phase 7: Pi Image

Bake an image with:

- Raspberry Pi OS Lite
- Docker Engine
- bootstrap service
- stack/config templates
- optional cloud-init or boot partition config support

## Non-Goals for v1

- automatic Postgres major upgrades
- Kubernetes
- high-availability multi-node clustering
- arbitrary reverse proxy support outside Traefik
