# bscout

`bscout` is a planned Go + Postgres reimplementation of the Nightscout API with a clinician-oriented UI.

This repository currently contains:

- a concrete rebuild plan in [docs/nightscout-rebuild-plan.md](/Users/henryschwerdtner/dev/bscout/docs/nightscout-rebuild-plan.md)
- an upstream test inventory in [docs/upstream-test-inventory.md](/Users/henryschwerdtner/dev/bscout/docs/upstream-test-inventory.md)
- a repeatable snapshot script in [scripts/fetch_nightscout_tests.sh](/Users/henryschwerdtner/dev/bscout/scripts/fetch_nightscout_tests.sh)
- vendored upstream Nightscout test artifacts in `third_party/nightscout/`

The immediate goal is to preserve Nightscout API behavior first, then replace the legacy UI with a smaller Go-native stack that better fits clinical workflows.

Deployment and appliance scaffolding now lives in:

- [docs/appliance-architecture.md](/Users/henryschwerdtner/dev/bscout/docs/appliance-architecture.md)
- [docs/appliance-implementation-plan.md](/Users/henryschwerdtner/dev/bscout/docs/appliance-implementation-plan.md)
- [deploy/swarm/stack.yml](/Users/henryschwerdtner/dev/bscout/deploy/swarm/stack.yml)
- [deploy/bootstrap/bootstrap.sh](/Users/henryschwerdtner/dev/bscout/deploy/bootstrap/bootstrap.sh)
- [.github/workflows/docker-release.yml](/Users/henryschwerdtner/dev/bscout/.github/workflows/docker-release.yml)
- [.github/workflows/pi-image.yml](/Users/henryschwerdtner/dev/bscout/.github/workflows/pi-image.yml)
