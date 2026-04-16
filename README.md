# glycoview

`glycoview` is a Go + Postgres application with a clinician-oriented UI and Nightscout-compatible API support.

This repository currently contains:

- the GlycoView application and dashboard
- appliance and deployment scaffolding
- integration of the external `github.com/glycoview/nightscout-api` library
- the `frontend` git submodule pointing to `github.com/glycoview/glycoview-ui`

Nightscout API behavior is provided through the external `nightscout-api` module, while this repository owns the application shell, storage implementations, appliance agent, and clinician workflows.

The appliance agent now expects a shared control-plane secret via `GLYCOVIEW_AGENT_TOKEN`. In the default appliance layout, that token is used both to authenticate dashboard-to-agent requests and, unless overridden by `GLYCOVIEW_AGENT_STATE_KEY`, to encrypt appliance state that contains DNS challenge credentials.

Deployment and appliance scaffolding now lives in:

- `docs/appliance-architecture.md`
- `docs/appliance-implementation-plan.md`
- `deploy/swarm/stack.yml`
- `deploy/bootstrap/bootstrap.sh`
- `.github/workflows/docker-release.yml`
- `.github/workflows/pi-image.yml`

Container publishing is GHCR-only. The deployment defaults expect:

- `ghcr.io/glycoview/glycoview`
- `ghcr.io/glycoview/glycoview-agent`

Clone with submodules:

```bash
git clone --recurse-submodules https://github.com/glycoview/glycoview.git
```
