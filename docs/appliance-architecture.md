# GlycoView Appliance Architecture

This document defines the target self-hosted Raspberry Pi appliance for `glycoview`.

## Goals

- Single-device install for non-technical self-hosters.
- No routine SSH access required for normal operation.
- Nightscout-compatible API access retained.
- Clinician dashboard hosted by the Go server.
- TLS onboarding from the dashboard, including DNS-01 flows.
- App updates initiated from the dashboard with automatic rollback.

## Runtime Topology

The appliance runtime is a single-node Docker Swarm on Raspberry Pi OS Lite.

Services:

- `traefik`
  - Public edge on `:80` and `:443`
  - ACME certificate management
  - HTTP-01 and DNS-01 support
- `glycoview`
  - Main application
  - 2 replicas for start-first rolling updates
  - No public port exposure
- `postgres`
  - Single replica
  - Persistent local volume
- `glycoview-agent`
  - Local management API
  - Docker socket access
  - Certificate provider config writes
  - Update orchestration and rollback
  - Shared-token auth for dashboard control requests

## Why Swarm

Single-node Swarm is the lowest-complexity Docker runtime that still gives:

- rolling updates
- `start-first` update order
- health-aware rollout
- rollback primitives
- secrets and configs

Those properties matter for dashboard-triggered updates without SSH.

## Network Layout

- `edge` overlay network: `traefik`, `glycoview`
- `control` overlay network: `glycoview`, `glycoview-agent`, `postgres`
- `glycoview-agent` is not exposed publicly

## Persistent State

- `postgres_data`
- `traefik_acme`
- `glycoview_backups`
- `glycoview_agent_state`

## TLS Strategy

Traefik is the certificate manager.

Supported modes:

- `HTTP-01`
  - simplest path for direct public domain routing
- `DNS-01`
  - required for wildcard certs
  - preferred for users behind CGNAT or strict routers

Provider support should be implemented through Traefik's ACME DNS support, which is backed by lego providers:

- Cloudflare
- Route53
- Hetzner
- DigitalOcean
- OVH
- Gandi
- Google Cloud DNS

Credentials are written by `glycoview-agent` into Docker secrets or encrypted local config, never stored in plaintext in Postgres. The agent state file is encrypted at rest with `GLYCOVIEW_AGENT_STATE_KEY` or, by default, the shared `GLYCOVIEW_AGENT_TOKEN`.

## Update Model

The dashboard talks to `glycoview-agent`.

Agent responsibilities:

- check current image tag
- check latest release/image digest
- pre-pull images
- run `docker service update` with `start-first`
- wait for service health
- rollback on failure
- report progress back to the dashboard

Postgres image updates are not automatic in v1. App and Traefik updates can be automated; database major version changes must remain explicit maintenance operations.

## First Boot

The Raspberry Pi image boots into a preinstalled bootstrap service that:

1. configures hostname/timezone/network from boot config
2. ensures Docker is available
3. initializes Swarm if needed
4. writes stack env/config
5. deploys the stack
6. exposes setup UI at local IP or `glycoview.local`

## Dashboard Setup Flow

1. Create first admin account
2. Show generated Nightscout-compatible API secret
3. Configure hostname/domain
4. Select TLS mode
5. If DNS-01: select provider and enter credentials
6. Agent writes provider config and redeploys Traefik
7. Dashboard starts serving on HTTPS

## Boundaries

This repository will contain:

- the app image
- the agent image
- stack definitions
- bootstrap assets
- GitHub Actions for release automation

The full certificate provider UX and rolling update orchestration exist in a first usable version; remaining work is hardening, richer progress reporting, and more provider coverage.
