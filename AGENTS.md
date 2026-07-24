# CLAUDE.md

This file provides guidance to AI agents when working with code in this repository.

## Repository overview

This is an **NS8 module** (`ns8-nethsecurity-controller`) that packages and configures an instance of [nethsecurity-controller](https://github.com/NethServer/nethsecurity-controller) on a NethServer 8 node. A single controller instance manages a fleet of NethSecurity firewall units over an OpenVPN tunnel, collecting their logs/metrics and exposing a Vue cluster-admin UI.

The module's containers run as one systemd-managed Podman pod:
- `nethsecurity-api` — Go REST API (source lives in the separate `NethServer/nethsecurity-controller` repo, **not** in this repo; only consumed as a prebuilt image)
- `nethsecurity-vpn` — OpenVPN server (external image)
- `nethsecurity-ui` — lighttpd static UI server (external image; distinct from this repo's own `ui/`)
- `nethsecurity-proxy` — Traefik reverse proxy (external image)
- `promtail` / `loki` — log shipping and storage
- `prometheus` — metrics scraping
- `timescale` — TimescaleDB for network-traffic/DPI/VPN time-series data
- `grafana` — dashboards over Prometheus + Loki + Timescale
- `webssh` — web SSH client, built locally from upstream `huashengdun/webssh` with a custom template

Only the module glue (`imageroot/`), the cluster-admin frontend (`ui/`), and the `webssh` UI override are implemented in this repo; the API server, VPN, UI-proxy, and other images are pulled prebuilt and pinned in `build-images.sh`.

## Build

`build-images.sh` builds images with **buildah** (no top-level Containerfile — built imperatively via `buildah from/add/config/commit`). It:
1. Builds `webssh` from `python:3.13.14-alpine`, unpacking the upstream `huashengdun/webssh` release and replacing its UI with `webssh/index.html`.
2. Builds the Vue `ui/` app in a `node:24.17.0-slim` builder container (`corepack enable && yarn install --frozen-lockfile && yarn build`), then assembles the main `nethsecurity-controller` image from `scratch` with `imageroot/` → `/imageroot` and `ui/dist` → `/ui`, plus NS8 image labels (`org.nethserver.authorizations`, `org.nethserver.min-core`, `org.nethserver.images`, etc.).

Version pins for all external images (`controller_version`, `promtail_image`, `loki_image`, `prometheus_image`, `grafana_image`, `timescale_image`, `webssh_version`) live at the top of `build-images.sh`. `controller_version` is kept in sync automatically both by Renovate (custom regex manager in `renovate.json`) and by the nightly `update-controller.yml` workflow.

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
- `update-controller.yml` — nightly cron that bumps `controller_version` in `build-images.sh` and opens a PR.
- `build-apidoc.yml` / `clean-apidoc.yml` — build/clean API docs from `validate-input.json`/`validate-output.json` changes.
