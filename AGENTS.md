# CLAUDE.md

This file provides guidance to AI agents when working with code in this repository.

## Repository overview

This is an **NS8 module** (`ns8-nethsecurity-controller`) that packages and configures an instance of [nethsecurity-controller](https://github.com/NethServer/nethsecurity-controller) on a NethServer 8 node. A single controller instance manages a fleet of NethSecurity firewall units over an OpenVPN tunnel, collecting their logs/metrics and exposing a Vue cluster-admin UI.

The module's containers run as one systemd-managed Podman pod:
- `nethsecurity-api` — Go REST API, built here from `controller/api/` (vendored source)
- `nethsecurity-vpn` — OpenVPN server, built here from `controller/vpn/`
- `nethsecurity-ui` — lighttpd static UI server, built here from `controller/ui/` (clones `NethServer/nethsecurity-ui` at build time; distinct from this repo's own `ui/`)
- `nethsecurity-proxy` — Traefik reverse proxy, built here from `controller/proxy/`
- `promtail` / `loki` — log shipping and storage
- `prometheus` — metrics scraping
- `timescale` — TimescaleDB for network-traffic/DPI/VPN time-series data
- `grafana` — dashboards over Prometheus + Loki + Timescale
- `webssh` — web SSH client, built locally from upstream `huashengdun/webssh` with a custom template

The controller sources (`nethsecurity-controller` API + VPN + proxy + UI) are **vendored** under `controller/` — a snapshot copy of `NethServer/nethsecurity-controller`, not a submodule. This repo builds and publishes those four images itself (to `ghcr.io/nethserver/nethsecurity-{api,vpn,proxy,ui}`), alongside the module glue (`imageroot/`), the cluster-admin frontend (`ui/`), and the `webssh` override. The observability images (promtail/loki/prometheus/grafana/timescale) are still pulled prebuilt and pinned in `build-images.sh`.

## Build

`build-images.sh` builds images with **buildah** (no top-level Containerfile — built imperatively via `buildah from/add/config/commit`, plus `buildah build` for the vendored controller Containerfiles). It:
1. Builds `webssh` from `python:3.13.14-alpine`, unpacking the upstream `huashengdun/webssh` release and replacing its UI with `webssh/index.html`.
2. Builds the four controller images — `nethsecurity-{api,vpn,proxy,ui}` — from `controller/<svc>/Containerfile` via `buildah build --layers` (default/last stage `dist`, so the api `test` stage does not run at build time).
3. Builds the Vue `ui/` app in a `node:24.17.0-slim` builder container (`corepack enable && yarn install --frozen-lockfile && yarn build`), then assembles the main `nethsecurity-controller` image from `scratch` with `imageroot/` → `/imageroot` and `ui/dist` → `/ui`, plus NS8 image labels (`org.nethserver.authorizations`, `org.nethserver.min-core`, `org.nethserver.images`, etc.).

The four controller images are referenced in the `org.nethserver.images` label at `${IMAGETAG:-latest}` (the module's own tag, like `webssh`); their basenames drive the `NETHSECURITY_{API,VPN,UI,PROXY}_IMAGE` env vars that the systemd units consume. Version pins for the still-external images (`promtail_image`, `loki_image`, `prometheus_image`, `grafana_image`, `timescale_image`, `webssh_version`) live at the top of `build-images.sh`. Base-image `FROM` pins inside the `controller/` Containerfiles and the `UI_VERSION` ARG (`NethServer/nethsecurity-ui`) are updated by Renovate (`customManagers:dockerfileVersions` in `renovate.json`).

## UI development (`ui/`)

Vue 2.6 app using Vuex, vue-router, vue-i18n, Carbon Design (`@carbon/vue`) and `@nethserver/ns8-ui-lib`. Package manager is **yarn (via corepack)**.

```bash
cd ui
yarn install --frozen-lockfile
yarn serve   # vue-cli-service serve, live dev server
yarn build   # vue-cli-service build -> dist/
yarn lint    # vue-cli-service lint (eslint: vue/essential + eslint:recommended + @vue/prettier)
```

`ui/src` layout is flat/singular: one `store/index.js`, one `router/index.js`, one `i18n/index.js`, and `views/` holds the three top-level pages (`Status.vue`, `Settings.vue`, `About.vue`). `ui/vue.config.js` disables image base64-inlining (module logo must stay a real file) and tunes webpack `splitChunks`.

The devcontainer (`.devcontainer/devcontainer.json`) builds from `ui/Containerfile` and sets `NODE_OPTIONS=--openssl-legacy-provider` (needed for this Vue-CLI/webpack version against modern Node/OpenSSL) — also set in `.vscode/settings.json` for integrated terminals.

## imageroot/ (module action scripts)

NS8 modules are driven by numbered scripts per action/event, executed in numeric order; language is bash or Python depending on whether the step needs structured config/JSON handling:

- `imageroot/actions/<action-name>/NNname` — e.g. `create-module/` (`10pip_install`, `20initialize`, `30set_prometheus_secret`, `40firewall`), `configure-module/` (`20configure`, `30subscription`, `70platform_info`, `80start_services`), `destroy-module/`, `get-configuration/`, `get-facts/`, `clone-module/`, `restore-module/`.
- `validate-input.json` / `validate-output.json` alongside an action's scripts define its JSON Schema (also used to generate API docs via `build-apidoc.yml`, triggered on changes to these files).
- `imageroot/update-module.d/` follows the same numbered convention for module updates; several entries are **symlinks back into `actions/`** to reuse scripts (e.g. `10pip_install -> ../actions/create-module/10pip_install`).
- `imageroot/events/<event-name>/` — event handlers (e.g. `support-session-started/10add_support_user`), same numbering convention.
- `imageroot/bin/` — standalone helper scripts invoked directly, not part of the action dispatch (`expand-grafana-config`, `expand-prometheus-config`, `module-cleanup-state`, `module-dump-state`).
- `imageroot/systemd/user/` — one unit per container; `controller.service` creates the shared Podman pod and `Requires=`/orders `Before=` all the other unit files, so it's the entry point for start/stop/restart.
- `imageroot/etc/` — static config templates (`promtail.yml`, `loki.yaml`, `dashboards.yml`) and the default Grafana dashboard JSON defs (`etc/dashboards/default/*.json`).

When changing an action's inputs/outputs, update its `validate-input.json`/`validate-output.json` in the same commit — CI treats them as the API contract.

## Tests (`tests/`)

Integration tests use **Robot Framework** driven over SSH against a live NS8 node (not unit tests against this repo's code in isolation):

- `tests/__init__.robot` — suite setup: opens an SSH connection to `${NODE_ADDR}` as root, waits for `systemctl is-system-running`, dumps `journalctl` per suite.
- `tests/nextsec-controller.robot` — installs the module (`add-module`), configures it (`api-cli run module/<id>/configure-module`), checks each systemd unit is active via `runagent -m <id> systemctl --user is-active <service>.service`, verifies admin login/JWT/health endpoints, and (tagged `ui`) uses the Playwright-backed `Browser` library to screenshot the `Status`/`Settings` pages headlessly.
- Test execution itself is not scripted locally — it runs through the reusable `test-module.yml@v1` workflow from `NethServer/ns8-github-actions` (see `.github/workflows/test-module.yml`, `ui_tests_strategy: on_renovate_ui_change`). Follow that repo's "Running tests locally" instructions (linked from this repo's README) to run the suite against a local NS8 node.

## CI (`.github/workflows/`)

All workflows are thin wrappers around reusable workflows in `NethServer/ns8-github-actions`; there is no dedicated local lint job:
- `publish-images.yml` — on push, runs `build-images.sh` via `publish-branch.yml@v1`; also runs `module-info.yml@v1` and (on stable/latest releases) `scan-with-trivy.yml@v1`.
- `test-module.yml` — runs the Robot Framework suite after images publish, or manually via `workflow_dispatch`.
- `test-controller.yml` — on changes under `controller/**`, runs the vendored controller's Go unit tests (against a TimescaleDB service) and a full-stack `test/smoke.sh` (builds the four images, starts the pod over a `tunsec` TUN device, checks login/health/unit endpoints).
- `build-apidoc.yml` / `clean-apidoc.yml` — build/clean API docs from `validate-input.json`/`validate-output.json` changes.

## Vendored controller (`controller/`)

`controller/` is a snapshot of `NethServer/nethsecurity-controller` (`api/`, `vpn/`, `proxy/`, `ui/`, `test/`, plus its own `build.sh`/`dev.sh` for local iteration). Edit the controller sources here directly; there is no sync-back to the upstream repo. The four backend services are the Go API, the OpenVPN server, and the Traefik proxy; the served firewall admin UI comes from the separate [`NethServer/nethsecurity-ui`](https://github.com/NethServer/nethsecurity-ui) repo, cloned at build time by `controller/ui/Containerfile`.

All commands below run from inside `controller/`.

### Dev environment (`dev.sh`)
`dev.sh` manages a Podman pod with every service (API, VPN, UI, proxy, DB) and generates an `api.env`.

1. Create the `tunsec` TUN device once (root):
   ```bash
   sudo ip tuntap add dev tunsec mod tun
   sudo ip addr add 172.21.0.1/16 dev tunsec
   sudo ip link set dev tunsec up
   ```
2. `./dev.sh start` — create/start the pod (also writes `api.env`).
3. `./dev.sh stop` — stop and remove the pod.

### Build (`build.sh`)
`./build.sh` builds all four service images locally with buildah (dir→image map), tagging by branch. This mirrors the CI build; the module's top-level `build-images.sh` builds the same images for publishing.

### Tests
The DB is **required** — do not run tests without it.
```bash
podman run --rm -d --name timescaledb -p 5432:5432 -e POSTGRES_PASSWORD=password -e POSTGRES_USER=report timescale/timescaledb:latest-pg16
for i in {1..30}; do podman exec timescaledb pg_isready -U report && break; sleep 1; done
cd api && go test ./...          # run the full Go suite
podman stop timescaledb          # stop the DB after
```
Add or update tests for any code you change. `test/smoke.sh` runs the whole stack end-to-end (build + `dev.sh start` + API checks).

### Docs & commits
- Keep each service's README current when changing it.
- Conventional Commits; scope = one of the services (`api`, `vpn`, `ui`, `proxy`), e.g. `feat(api): add endpoint for user preferences`, `fix(vpn): resolve auth handshake failure`.
